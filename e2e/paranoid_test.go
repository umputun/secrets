//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	paranoidServerURL     = "http://localhost:18085"
	paranoidAuthServerURL = "http://localhost:18086"
	// bcrypt hash for "testpass123" (same as auth_test.go)
	paranoidTestAuthHash = "$2a$10$q8UbvhCC/LFB4kCxenUGQ.34UuyUVg.7otCerbwj9xkrNXO9Fd2S2"
)

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

func TestParanoid_TextSizeValidation(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	// server configured with --files.max-size=1048576 (1MB)
	// create text slightly over 1MB
	oversizedText := strings.Repeat("x", 1048577)

	// fill in oversized message
	require.NoError(t, page.Locator("#message").Fill(oversizedText))
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// should show size error in form-errors div
	errorSpan := page.Locator("#form-errors .error")
	waitVisible(t, errorSpan)

	errorText, err := errorSpan.TextContent()
	require.NoError(t, err)
	assert.Contains(t, errorText, "Message too large", "should show size validation error")
	assert.Contains(t, errorText, "1", "should mention size limit")
}

func TestParanoid_ColorScheme(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	// verify data-paranoid attribute is present on html element
	// GetAttribute returns empty string for missing attributes, so check locator exists
	htmlWithParanoid := page.Locator("html[data-paranoid]")
	count, err := htmlWithParanoid.Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count, "html element should have data-paranoid attribute")

	// verify theme-color meta tag has muted red color
	themeColor := page.Locator("meta[name='theme-color']")
	color, err := themeColor.GetAttribute("content")
	require.NoError(t, err)
	assert.Equal(t, "#c45850", color, "theme-color should be dusty crimson in paranoid mode")

	// verify CSS variable is applied - check the accent color variable
	result, err := page.Evaluate(`(() => {
		const style = getComputedStyle(document.documentElement);
		return style.getPropertyValue('--color-accent').trim();
	})()`)
	require.NoError(t, err)
	accentColor := result.(string)
	assert.Equal(t, "#c45850", accentColor, "CSS accent color should be dusty crimson")
}

func TestParanoid_FileSizeValidation(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	// switch to file tab first
	fileTab := page.Locator("#file-tab")
	require.NoError(t, fileTab.Click())

	// create temp file slightly over 1MB
	tmpFile, err := os.CreateTemp("", "paranoid-test-*.bin")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	oversizedData := make([]byte, 1048577)
	_, err = tmpFile.Write(oversizedData)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// upload file via file input
	fileInput := page.Locator("#file")
	require.NoError(t, fileInput.SetInputFiles(tmpFile.Name()))

	// file size validation happens immediately on file selection
	// error shown in file-info element, submit button gets disabled
	fileInfo := page.Locator("#file-info.error")
	waitVisible(t, fileInfo)

	errorText, err := fileInfo.TextContent()
	require.NoError(t, err)
	assert.Contains(t, errorText, "too large", "should show file size validation error")

	// submit button should be disabled
	submitBtn := page.Locator("button[type='submit']")
	isDisabled, err := submitBtn.IsDisabled()
	require.NoError(t, err)
	assert.True(t, isDisabled, "submit button should be disabled for oversized file")
}

// startParanoidAuthServer starts a server with both paranoid mode and auth enabled.
func startParanoidAuthServer(t *testing.T) func() {
	t.Helper()

	cmd := exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e-paranoid-auth",
		"--domain=localhost:18086",
		"--protocol=http",
		"--listen=:18086",
		"--pinsize=5",
		"--expire=1h",
		"--pinattempts=3",
		"--paranoid",
		"--files.enabled",
		"--files.max-size=1048576",
		"--auth.hash="+paranoidTestAuthHash,
		"--dbg",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start paranoid+auth server: %v", err)
	}

	if err := waitForServer(paranoidAuthServerURL+"/ping", 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("paranoid+auth server not ready: %v", err)
	}

	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func TestParanoid_FileUpload_RoundTrip(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	page := newPage(t)

	// capture ALL console messages for debugging
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		t.Logf("CONSOLE [%s]: %s", msg.Type(), msg.Text())
	})

	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	// switch to file tab
	fileTab := page.Locator("#file-tab")
	require.NoError(t, fileTab.Click())

	// create temp file with test content
	tmpFile, err := os.CreateTemp("", "paranoid-file-test-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	testContent := "paranoid file upload content with special chars: <>&\"'"
	_, err = tmpFile.WriteString(testContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	// get the filename for later verification
	expectedFilename := filepath.Base(tmpFile.Name())

	// upload file
	fileInput := page.Locator("#file")
	require.NoError(t, fileInput.SetInputFiles(tmpFile.Name()))

	// fill in PIN
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// wait for the secret link
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	t.Logf("secret link: %s", secretLink)

	// verify URL has paranoid mode fragment
	assert.Contains(t, secretLink, "/message/", "URL should contain /message/ path")
	assert.Contains(t, secretLink, "#", "paranoid mode URL should have fragment")

	// navigate to message page
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// enter PIN
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// for file messages in paranoid mode, after decryption a success card appears
	// with "File Downloaded" title and the filename displayed
	successCard := page.Locator(".success-card")
	waitVisible(t, successCard)

	title := page.Locator(".card-title:has-text('File Downloaded')")
	visible, err := title.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "should show 'File Downloaded' title")

	// verify the filename is displayed
	filenameSpan := page.Locator("#downloaded-filename")
	filename, err := filenameSpan.TextContent()
	require.NoError(t, err)
	assert.Equal(t, expectedFilename, filename, "downloaded filename should match")
	t.Logf("file downloaded successfully: %s", filename)
}

func TestParanoid_WithAuth_RoundTrip(t *testing.T) {
	cleanup := startParanoidAuthServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(paranoidAuthServerURL)
	require.NoError(t, err)

	originalMessage := "paranoid + auth test message"

	// fill in message and PIN
	require.NoError(t, page.Locator("#message").Fill(originalMessage))
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form - should trigger auth popup (not create secret yet)
	require.NoError(t, page.Locator("button[type='submit']").Click())
	popup := page.Locator("#popup.active")
	waitVisible(t, popup)

	// verify login popup appeared
	visible, err := popup.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "login popup should appear in paranoid mode with auth")

	// enter password and submit login
	require.NoError(t, page.Locator("#password").Fill("testpass123"))
	require.NoError(t, page.Locator("#popup button[type='submit']").Click())

	// after successful auth, should see the secure link with fragment
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	t.Logf("secret link: %s", secretLink)

	// verify URL has paranoid mode fragment
	assert.Contains(t, secretLink, "/message/", "URL should contain /message/ path")
	assert.Contains(t, secretLink, "#", "paranoid mode URL should have fragment")

	// extract and verify fragment key format
	parts := strings.Split(secretLink, "#")
	require.Len(t, parts, 2, "URL should have exactly one fragment")
	key := parts[1]
	assert.Len(t, key, 22, "key should be 22 characters (128-bit base64url)")

	// navigate to message and decrypt
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// verify decryption works
	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Equal(t, originalMessage, content, "decrypted message should match original")
}

// TestParanoid_ReEncryptsAfterValidationError verifies that after a 400 validation error,
// the next submission properly re-encrypts the message (bug fix for encryptionDone flag not resetting)
func TestParanoid_ReEncryptsAfterValidationError(t *testing.T) {
	cleanup := startParanoidServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(paranoidServerURL)
	require.NoError(t, err)

	// fill in message and PIN
	originalMessage := "message that should be re-encrypted after error"
	require.NoError(t, page.Locator("#message").Fill(originalMessage))
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// set expiration to exceed max (server max is 1h, we'll try 999 days)
	require.NoError(t, page.Locator("#exp").Fill("999"))
	_, err = page.Locator("#expUnit").SelectOption(playwright.SelectOptionValues{Values: &[]string{"d"}})
	require.NoError(t, err)

	// submit - should get 400 validation error
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// wait for error to appear (form is re-rendered with error)
	errorSpan := page.Locator(".error")
	waitVisible(t, errorSpan)

	errorText, err := errorSpan.TextContent()
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(errorText), "expire", "expected expiration validation error")
	t.Logf("got expected validation error: %s", errorText)

	// fix the form and change the message to something different
	newMessage := "DIFFERENT message after fixing error"
	require.NoError(t, page.Locator("#message").Clear())
	require.NoError(t, page.Locator("#message").Fill(newMessage))
	require.NoError(t, page.Locator("#pin").Clear())
	require.NoError(t, page.Locator("#pin").Fill(testPin)) // must re-fill PIN - server clears it on 400
	require.NoError(t, page.Locator("#exp").Clear())
	require.NoError(t, page.Locator("#exp").Fill("15"))
	_, err = page.Locator("#expUnit").SelectOption(playwright.SelectOptionValues{Values: &[]string{"m"}})
	require.NoError(t, err)

	// re-submit - should work and encrypt the NEW message
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// wait for success
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	assert.Contains(t, secretLink, "#", "should have encryption key fragment")
	t.Logf("secret link after retry: %s", secretLink)

	// now verify the NEW message was encrypted by decrypting it
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())

	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)

	// critical assertion: the decrypted content must be the NEW message, not the old one
	// if encryptionDone wasn't reset, we'd have sent plaintext or old encrypted blob
	assert.Equal(t, newMessage, content, "decrypted message should be the NEW message, proving re-encryption worked")
	assert.NotEqual(t, originalMessage, content, "should NOT be the original message")
}
