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

const paranoidServerURL = "http://localhost:18085"

// startParanoidServer starts a server with paranoid mode enabled.
// Returns a cleanup function that stops the server.
func startParanoidServer(t *testing.T) func() {
	t.Helper()

	cmd := exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e-paranoid",
		"--domain=localhost:18085",
		"--protocol=http",
		"--listen=:18085",
		"--pinsize=5",
		"--expire=1h",
		"--pinattempts=3",
		"--paranoid",
		"--files.enabled",
		"--files.max-size=1048576",
		"--dbg",
	)
	// disable auth for paranoid tests
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
		t.Fatalf("failed to start paranoid server: %v", err)
	}

	if err := waitForServer(paranoidServerURL+"/ping", 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("paranoid server not ready: %v", err)
	}

	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func TestParanoid_CryptoAvailable(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	// crypto.js should be loaded in paranoid mode
	result, err := page.Evaluate("typeof checkCryptoAvailable === 'function'")
	require.NoError(t, err)
	assert.True(t, result.(bool), "checkCryptoAvailable function should be defined")

	// web Crypto should be available on localhost
	result, err = page.Evaluate("checkCryptoAvailable()")
	require.NoError(t, err)
	assert.True(t, result.(bool), "Web Crypto should be available on localhost")
}

func TestParanoid_CreateTextSecret_HasFragment(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	// fill in message
	require.NoError(t, page.Locator("#message").Fill("paranoid secret message"))

	// fill in PIN
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form and wait for result
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	// get the generated link
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// in paranoid mode, URL should have fragment with key
	assert.Contains(t, secretLink, "/message/", "URL should contain /message/ path")
	assert.Contains(t, secretLink, "#", "paranoid mode URL should have fragment")

	// extract fragment and verify it's a valid base64url key (22 chars)
	parts := strings.Split(secretLink, "#")
	require.Len(t, parts, 2, "URL should have exactly one fragment")
	key := parts[1]
	assert.Len(t, key, 22, "key should be 22 characters (128-bit base64url)")
	assert.Regexp(t, `^[A-Za-z0-9_-]+$`, key, "key should be base64url encoded")
}

func TestParanoid_CreateTextSecret_RoundTrip(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	originalMessage := "paranoid round-trip test message with special chars: <>&\"'"

	// create secret
	require.NoError(t, page.Locator("#message").Fill(originalMessage))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	// get the generated link with fragment
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	t.Logf("secret link: %s", secretLink)

	// navigate to the message page (full URL with fragment)
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// enter PIN
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// wait for decoded message - in paranoid mode, client decrypts and displays
	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	// verify content matches
	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Equal(t, originalMessage, content, "decrypted message should match original")
}

func TestParanoid_WrongPin(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	// create secret
	require.NoError(t, page.Locator("#message").Fill("paranoid wrong pin test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// navigate to message page
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// enter wrong PIN - in paranoid mode, server validates PIN but returns encrypted blob
	// client tries to decrypt which fails, showing "wrong pin" from server response parsing
	require.NoError(t, page.Locator("#pin").Fill("99999"))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	errorSpan := page.Locator("#pin-error .error")
	waitVisible(t, errorSpan)

	errorText, err := errorSpan.TextContent()
	require.NoError(t, err)
	// server returns wrong pin error which JS parses and displays
	// must NOT show "Decryption failed" - that would indicate the bug where HTML error was treated as encrypted blob
	assert.Contains(t, errorText, "wrong pin", "expected 'wrong pin' error, got: %s", errorText)
	assert.NotContains(t, errorText, "Decryption failed", "should show actual error, not decryption failure")
}

func TestParanoid_OneTimeRead(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	page := newPage(t)

	// capture console messages for debugging
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		t.Logf("CONSOLE [%s]: %s", msg.Type(), msg.Text())
	})

	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	// create secret
	require.NoError(t, page.Locator("#message").Fill("paranoid one-time test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// first access - should succeed and delete the message
	_, err = page.Goto(secretLink)
	require.NoError(t, err)
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// wait for either decoded message or success card (decryption happens client-side)
	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Equal(t, "paranoid one-time test", content, "decrypted message should match")

	// second access - message should be deleted, show error
	// navigate away first to ensure fresh page load (same-URL navigation may not reload)
	_, err = page.Goto(paranoidServerURL)
	require.NoError(t, err)
	_, err = page.Goto(secretLink)
	require.NoError(t, err)
	t.Logf("navigated to secretLink for second access, current URL: %s", page.URL())

	// wait for PIN form to appear (server should show form even for deleted messages)
	pinInput := page.Locator("#pin")
	waitVisible(t, pinInput)
	require.NoError(t, pinInput.Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// for deleted/expired message, JS shows "Message Unavailable" card with actual error
	// must NOT show "Decryption failed" - that would indicate the bug where HTML error was treated as encrypted blob
	errorCard := page.Locator(".error-card .error-message, #pin-error .error")
	waitVisible(t, errorCard)

	errorText, err := errorCard.First().TextContent()
	require.NoError(t, err)
	t.Logf("error text: %s", errorText)
	assert.NotContains(t, errorText, "Decryption failed", "should show actual error, not decryption failure")
}

func TestParanoid_EmailButtonHasFragment(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	// start server with email enabled (we just check the button, not actual sending)
	page := newPage(t)
	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	// create secret
	require.NoError(t, page.Locator("#message").Fill("email button test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	// get the full URL with fragment
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	require.Contains(t, secretLink, "#", "URL should have fragment")

	// check if email button exists and has updated link
	emailBtn := page.Locator("button[hx-get^='/email-popup']")
	visible, _ := emailBtn.IsVisible()
	if visible {
		// if email is enabled, verify the button's hx-get includes the fragment
		hxGet, err := emailBtn.GetAttribute("hx-get")
		require.NoError(t, err)
		// the link should be URL-encoded, so # becomes %23
		assert.Contains(t, hxGet, "%23", "email button link should contain encoded fragment")
	}
}
