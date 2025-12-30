//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const noPinServerURL = "http://localhost:18085"

// startNoPinServer starts a server with --allow-no-pin enabled.
func startNoPinServer(t *testing.T) func() {
	t.Helper()

	cmd := exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e-nopin",
		"--domain=localhost:18085",
		"--protocol=http",
		"--listen=:18085",
		"--pinsize=5",
		"--expire=1h",
		"--pinattempts=3",
		"--allow-no-pin",
		"--dbg",
	)
	env := []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "AUTH_HASH=") {
			env = append(env, e)
		}
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start no-pin server: %v", err)
	}

	if err := waitForServer(noPinServerURL+"/ping", 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("no-pin server not ready: %v", err)
	}

	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

// TestNoPin_FormShowsOptionalLabel verifies the PIN field shows "(optional)" when AllowNoPin is enabled.
func TestNoPin_FormShowsOptionalLabel(t *testing.T) {
	cleanup := startNoPinServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(noPinServerURL)
	require.NoError(t, err)

	// check PIN label shows "(optional)"
	pinLabel := page.Locator("label[for='pin']")
	text, err := pinLabel.TextContent()
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(text), "optional", "PIN label should indicate optional")

	// check PIN input does not have 'required' attribute
	required, err := page.Locator("#pin").GetAttribute("required")
	require.NoError(t, err)
	assert.Empty(t, required, "PIN input should not have required attribute")
}

// TestNoPin_ModalAppearsOnEmptyPin verifies the confirmation modal appears when submitting with empty PIN.
func TestNoPin_ModalAppearsOnEmptyPin(t *testing.T) {
	cleanup := startNoPinServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(noPinServerURL)
	require.NoError(t, err)

	// fill message but leave PIN empty
	require.NoError(t, page.Locator("#message").Fill("test message without pin"))

	// submit form
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// modal should appear
	modal := page.Locator("#no-pin-modal")
	waitVisible(t, modal)

	// verify modal content
	title := page.Locator("#no-pin-modal h2")
	titleText, err := title.TextContent()
	require.NoError(t, err)
	assert.Contains(t, titleText, "without PIN", "modal should mention creating without PIN")
}

// TestNoPin_ModalCancelFocusesPIN verifies clicking "Add PIN" closes modal and focuses PIN field.
func TestNoPin_ModalCancelFocusesPIN(t *testing.T) {
	cleanup := startNoPinServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(noPinServerURL)
	require.NoError(t, err)

	// fill message but leave PIN empty
	require.NoError(t, page.Locator("#message").Fill("test message"))

	// submit form
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// wait for modal to appear
	modal := page.Locator("#no-pin-modal")
	waitVisible(t, modal)

	// click "Add PIN" button
	cancelBtn := page.Locator("#no-pin-cancel")
	require.NoError(t, cancelBtn.Click())

	// modal should be hidden
	waitHidden(t, modal)

	// PIN input should be focused
	focused, err := page.Locator("#pin").Evaluate("el => document.activeElement === el", nil)
	require.NoError(t, err)
	assert.True(t, focused.(bool), "PIN input should be focused after cancel")
}

// TestNoPin_ModalConfirmCreatesSecret verifies clicking "Continue without PIN" creates the secret.
func TestNoPin_ModalConfirmCreatesSecret(t *testing.T) {
	cleanup := startNoPinServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(noPinServerURL)
	require.NoError(t, err)

	// fill message but leave PIN empty
	require.NoError(t, page.Locator("#message").Fill("secret without pin"))

	// submit form
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// wait for modal and confirm
	modal := page.Locator("#no-pin-modal")
	waitVisible(t, modal)
	confirmBtn := page.Locator("#no-pin-confirm")
	require.NoError(t, confirmBtn.Click())

	// modal should close
	waitHidden(t, modal)

	// secret link should appear
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	assert.Contains(t, secretLink, "/message/", "secret link should be generated")
}

// TestNoPin_RevealButtonShown verifies PIN-less secret shows "Reveal Secret" button instead of PIN form.
func TestNoPin_RevealButtonShown(t *testing.T) {
	cleanup := startNoPinServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(noPinServerURL)
	require.NoError(t, err)

	// create secret without PIN
	require.NoError(t, page.Locator("#message").Fill("reveal button test"))
	require.NoError(t, page.Locator("button[type='submit']").Click())

	modal := page.Locator("#no-pin-modal")
	waitVisible(t, modal)
	require.NoError(t, page.Locator("#no-pin-confirm").Click())

	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// navigate to secret page
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// should show "Reveal Secret" button, not PIN form
	decryptBtn := page.Locator("#decrypt-btn")
	waitVisible(t, decryptBtn)

	btnText, err := decryptBtn.TextContent()
	require.NoError(t, err)
	assert.Contains(t, btnText, "Reveal", "button should say Reveal for PIN-less secrets")

	// PIN input should NOT be visible
	pinInput := page.Locator("#client-pin[type='text']")
	visible, err := pinInput.IsVisible()
	require.NoError(t, err)
	assert.False(t, visible, "PIN input should not be visible for PIN-less secrets")
}

// TestNoPin_RevealButtonRevealsSecret verifies clicking reveal button shows the secret content.
func TestNoPin_RevealButtonRevealsSecret(t *testing.T) {
	cleanup := startNoPinServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(noPinServerURL)
	require.NoError(t, err)

	// create secret without PIN
	testMessage := "this is a secret without pin protection"
	require.NoError(t, page.Locator("#message").Fill(testMessage))
	require.NoError(t, page.Locator("button[type='submit']").Click())

	modal := page.Locator("#no-pin-modal")
	waitVisible(t, modal)
	require.NoError(t, page.Locator("#no-pin-confirm").Click())

	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// navigate to secret page
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// click reveal button
	decryptBtn := page.Locator("#decrypt-btn")
	waitVisible(t, decryptBtn)
	require.NoError(t, decryptBtn.Click())

	// message should be revealed
	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Contains(t, content, testMessage, "revealed message should match original")
}

// TestNoPin_SecretWithPinStillWorks verifies that creating secrets WITH PIN still works when AllowNoPin is enabled.
func TestNoPin_SecretWithPinStillWorks(t *testing.T) {
	cleanup := startNoPinServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(noPinServerURL)
	require.NoError(t, err)

	// create secret WITH PIN
	testMessage := "secret with pin"
	require.NoError(t, page.Locator("#message").Fill(testMessage))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// no modal should appear when PIN is provided
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// navigate to secret page
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// should show PIN form (button says "Decrypt")
	decryptBtn := page.Locator("#decrypt-btn")
	waitVisible(t, decryptBtn)

	btnText, err := decryptBtn.TextContent()
	require.NoError(t, err)
	assert.Contains(t, btnText, "Decrypt", "button should say Decrypt for secrets with PIN")

	// PIN input should be visible
	pinInput := page.Locator("#client-pin")
	visible, err := pinInput.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "PIN input should be visible for secrets with PIN")

	// enter PIN and reveal
	require.NoError(t, pinInput.Fill(testPin))
	require.NoError(t, decryptBtn.Click())

	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Contains(t, content, testMessage, "revealed message should match original")
}

// TestNoPin_DeletedSecretShows404 verifies accessing a deleted no-pin message shows 404 instead of PIN form.
func TestNoPin_DeletedSecretShows404(t *testing.T) {
	cleanup := startNoPinServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(noPinServerURL)
	require.NoError(t, err)

	// create secret without PIN
	testMessage := "secret to be deleted"
	require.NoError(t, page.Locator("#message").Fill(testMessage))
	require.NoError(t, page.Locator("button[type='submit']").Click())

	modal := page.Locator("#no-pin-modal")
	waitVisible(t, modal)
	require.NoError(t, page.Locator("#no-pin-confirm").Click())

	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// navigate to secret page
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// reveal the secret (this deletes it)
	decryptBtn := page.Locator("#decrypt-btn")
	waitVisible(t, decryptBtn)
	require.NoError(t, decryptBtn.Click())

	// verify message is revealed
	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	// now try to access the same link again - should show 404, not PIN form
	// first navigate away to ensure a fresh navigation
	_, err = page.Goto(noPinServerURL + "/about")
	require.NoError(t, err)
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// should show "Message Unavailable" error card (not raw 404)
	errorCard := page.Locator("h2:has-text('Message Unavailable')")
	waitVisible(t, errorCard)

	// verify error message
	errorMsg := page.Locator(".card-description")
	text, err := errorMsg.TextContent()
	require.NoError(t, err)
	assert.Contains(t, text, "expired or deleted")

	// should NOT show PIN form or reveal button
	pinInput := page.Locator("#client-pin")
	visible, err := pinInput.IsVisible()
	require.NoError(t, err)
	assert.False(t, visible, "PIN input should NOT be visible for deleted messages")

	decryptVisible, err := page.Locator("#decrypt-btn").IsVisible()
	require.NoError(t, err)
	assert.False(t, decryptVisible, "decrypt button should NOT be visible for deleted messages")
}

// TestNoPin_DisabledBlocksEmptyPin verifies empty PIN is blocked when AllowNoPin is disabled.
func TestNoPin_DisabledBlocksEmptyPin(t *testing.T) {
	// use the default server (without --allow-no-pin)
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// check PIN input has 'required' attribute (boolean attributes return empty string when present)
	err = page.Locator("#pin[required]").WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateAttached,
		Timeout: playwright.Float(5000),
	})
	require.NoError(t, err, "PIN input should have required attribute when AllowNoPin is disabled")

	// fill message but leave PIN empty
	require.NoError(t, page.Locator("#message").Fill("test message"))

	// submit form - should be blocked by HTML5 validation
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// should stay on form (no link generated)
	visible, err := page.Locator("#pin").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "should stay on form when PIN is required but empty")

	// no modal should appear
	modal := page.Locator("#no-pin-modal")
	visible, _ = modal.IsVisible()
	assert.False(t, visible, "modal should not exist when AllowNoPin is disabled")
}
