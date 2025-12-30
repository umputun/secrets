//go:build e2e

package e2e

import (
	"encoding/json"
	"io"
	"net/http"
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
	hybridServerURL     = "http://localhost:18085"
	hybridAuthServerURL = "http://localhost:18086"
	// bcrypt hash for "testpass123" (same as auth_test.go)
	hybridTestAuthHash = "$2a$10$q8UbvhCC/LFB4kCxenUGQ.34UuyUVg.7otCerbwj9xkrNXO9Fd2S2"
)

// startHybridServer starts a server for hybrid mode testing (UI=client-enc, API=server-enc).
// Returns a cleanup function that stops the server.
func startHybridServer(t *testing.T) func() {
	t.Helper()

	cmd := exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e-hybrid",
		"--domain=localhost:18085",
		"--protocol=http",
		"--listen=:18085",
		"--pinsize=5",
		"--expire=1h",
		"--pinattempts=3",
		"--files.enabled",
		"--files.max-size=1048576",
		"--dbg",
	)
	// disable auth for hybrid tests
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
		t.Fatalf("failed to start hybrid server: %v", err)
	}

	if err := waitForServer(hybridServerURL+"/ping", 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("hybrid server not ready: %v", err)
	}

	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

// startHybridAuthServer starts a server with auth enabled.
func startHybridAuthServer(t *testing.T) func() {
	t.Helper()

	cmd := exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e-hybrid-auth",
		"--domain=localhost:18086",
		"--protocol=http",
		"--listen=:18086",
		"--pinsize=5",
		"--expire=1h",
		"--pinattempts=3",
		"--files.enabled",
		"--files.max-size=1048576",
		"--auth.hash="+hybridTestAuthHash,
		"--dbg",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start hybrid+auth server: %v", err)
	}

	if err := waitForServer(hybridAuthServerURL+"/ping", 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("hybrid+auth server not ready: %v", err)
	}

	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

// --- UI flow tests (client-side encryption) ---

func TestHybrid_UIFlow_CryptoAvailable(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridServerURL)
	require.NoError(t, err)

	// crypto.js should always be loaded (no longer conditional)
	result, err := page.Evaluate("typeof checkCryptoAvailable === 'function'")
	require.NoError(t, err)
	assert.True(t, result.(bool), "checkCryptoAvailable function should be defined")

	// web Crypto should be available on localhost
	result, err = page.Evaluate("checkCryptoAvailable()")
	require.NoError(t, err)
	assert.True(t, result.(bool), "Web Crypto should be available on localhost")
}

func TestHybrid_UIFlow_CreateTextSecret_HasFragment(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridServerURL)
	require.NoError(t, err)

	// fill in message
	require.NoError(t, page.Locator("#message").Fill("client-encrypted secret message"))

	// fill in PIN
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form and wait for result
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	// get the generated link
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// UI-created messages should always have fragment with key
	assert.Contains(t, secretLink, "/message/", "URL should contain /message/ path")
	assert.Contains(t, secretLink, "#", "UI-created URL should have fragment with encryption key")

	// extract fragment and verify it's a valid base64url key (22 chars)
	parts := strings.Split(secretLink, "#")
	require.Len(t, parts, 2, "URL should have exactly one fragment")
	key := parts[1]
	assert.Len(t, key, 22, "key should be 22 characters (128-bit base64url)")
	assert.Regexp(t, `^[A-Za-z0-9_-]+$`, key, "key should be base64url encoded")
}

func TestHybrid_UIFlow_CreateAndRetrieve_RoundTrip(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridServerURL)
	require.NoError(t, err)

	originalMessage := "client-side round-trip test message with special chars: <>&\"'"

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

	// enter PIN (client-side form uses #client-pin when URL has #key)
	clientPin := page.Locator("#client-pin")
	waitVisible(t, clientPin)
	require.NoError(t, clientPin.Fill(testPin))
	require.NoError(t, page.Locator("#decrypt-btn").Click())

	// wait for decoded message - client decrypts and displays
	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	// verify content matches
	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Equal(t, originalMessage, content, "decrypted message should match original")
}

func TestHybrid_UIFlow_WrongPin(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridServerURL)
	require.NoError(t, err)

	// create secret
	require.NoError(t, page.Locator("#message").Fill("wrong pin test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// navigate to message page
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// enter wrong PIN (client-side form uses #client-pin when URL has #key)
	clientPin := page.Locator("#client-pin")
	waitVisible(t, clientPin)
	require.NoError(t, clientPin.Fill("99999"))
	require.NoError(t, page.Locator("#decrypt-btn").Click())
	errorSpan := page.Locator("#client-pin-error .error")
	waitVisible(t, errorSpan)

	errorText, err := errorSpan.TextContent()
	require.NoError(t, err)
	assert.Contains(t, errorText, "wrong pin", "expected 'wrong pin' error, got: %s", errorText)
	assert.NotContains(t, errorText, "Decryption failed", "should show actual error, not decryption failure")
}

func TestHybrid_UIFlow_OneTimeRead(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridServerURL)
	require.NoError(t, err)

	// create secret
	require.NoError(t, page.Locator("#message").Fill("one-time read test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// first access - should succeed and delete the message
	_, err = page.Goto(secretLink)
	require.NoError(t, err)
	clientPin := page.Locator("#client-pin")
	waitVisible(t, clientPin)
	require.NoError(t, clientPin.Fill(testPin))
	require.NoError(t, page.Locator("#decrypt-btn").Click())

	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Equal(t, "one-time read test", content, "decrypted message should match")

	// second access - message should be deleted, show 404 immediately
	_, err = page.Goto(hybridServerURL)
	require.NoError(t, err)
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// should show "Message Unavailable" error page immediately (no PIN form)
	errorCard := page.Locator("h2:has-text('Message Unavailable')")
	waitVisible(t, errorCard)

	// verify error message content
	errorMsg := page.Locator(".card-description")
	errorText, err := errorMsg.TextContent()
	require.NoError(t, err)
	assert.Contains(t, errorText, "expired or deleted")

	// verify no PIN form shown
	pinInput := page.Locator("#client-pin")
	visible, err := pinInput.IsVisible()
	require.NoError(t, err)
	assert.False(t, visible, "PIN input should NOT be visible for deleted messages")
}

// --- API flow tests (server-side encryption) ---

func TestHybrid_APIFlow_CreateAndRetrieve(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	// create message via API (exp is in seconds)
	reqBody := `{"Message":"API server-encrypted message","Pin":"12345","Exp":900}`
	resp, err := http.Post(hybridServerURL+"/api/v1/message", "application/json", strings.NewReader(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Key string `json:"key"`
		Exp string `json:"exp"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&createResp))
	require.NotEmpty(t, createResp.Key, "API should return message key")
	t.Logf("created API message with key: %s", createResp.Key)

	// retrieve via API - should return decrypted plaintext
	resp, err = http.Get(hybridServerURL + "/api/v1/message/" + createResp.Key + "/12345")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var getResp struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(body, &getResp))
	assert.Equal(t, "API server-encrypted message", getResp.Message, "API should return decrypted message")
}

func TestHybrid_APIFlow_WrongPin(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	// create message via API (exp is in seconds)
	reqBody := `{"Message":"API message for wrong pin test","Pin":"12345","Exp":900}`
	resp, err := http.Post(hybridServerURL+"/api/v1/message", "application/json", strings.NewReader(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Key string `json:"key"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&createResp))

	// retrieve with wrong PIN (server returns 417 Expectation Failed for wrong PIN)
	resp, err = http.Get(hybridServerURL + "/api/v1/message/" + createResp.Key + "/99999")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusExpectationFailed, resp.StatusCode, "wrong PIN should return 417")
}

// --- cross-mode tests ---

func TestHybrid_CrossMode_UICreateAPIRetrieve(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridServerURL)
	require.NoError(t, err)

	// create secret via UI (client-side encrypted)
	require.NoError(t, page.Locator("#message").Fill("UI message for API retrieval"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	t.Logf("UI-created link: %s", secretLink)

	// extract message key from URL
	messageKey := extractMessageKey(t, secretLink)

	// retrieve via API - should return the raw encrypted blob (ClientEnc=true)
	resp, err := http.Get(hybridServerURL + "/api/v1/message/" + messageKey + "/" + testPin)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var getResp struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(body, &getResp))

	// the message should be the encrypted blob (base64url encoded ciphertext)
	// it should NOT be the plaintext since it was client-encrypted
	assert.NotEqual(t, "UI message for API retrieval", getResp.Message, "API should return encrypted blob, not plaintext")
	assert.NotEmpty(t, getResp.Message, "API should return the encrypted blob")
	t.Logf("API returned encrypted blob (length=%d): %s...", len(getResp.Message), getResp.Message[:min(50, len(getResp.Message))])
}

func TestHybrid_CrossMode_APICreateUIRetrieve(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	// create message via API (server-side encrypted, exp is in seconds)
	reqBody := `{"Message":"API message for UI retrieval","Pin":"12345","Exp":900}`
	resp, err := http.Post(hybridServerURL+"/api/v1/message", "application/json", strings.NewReader(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp struct {
		Key string `json:"key"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&createResp))
	t.Logf("created API message with key: %s", createResp.Key)

	// retrieve via UI (no fragment in URL - server decrypts)
	page := newPage(t)
	_, err = page.Goto(hybridServerURL + "/message/" + createResp.Key)
	require.NoError(t, err)

	// enter PIN (server-side form uses #pin when no #key in URL)
	pinInput := page.Locator("#pin")
	waitVisible(t, pinInput)
	require.NoError(t, pinInput.Fill(testPin))
	// use specific form button selector (load-msg-form is the HTMX server-side form)
	require.NoError(t, page.Locator("#load-msg-form button[type='submit']").Click())

	// wait for decoded message - server decrypts and displays
	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Equal(t, "API message for UI retrieval", content, "UI should display server-decrypted message")
}

// --- file upload tests ---

func TestHybrid_UIFile_ClientSideEncryption_RoundTrip(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	page := newPage(t)

	// capture console messages for debugging
	page.OnConsole(func(msg playwright.ConsoleMessage) {
		t.Logf("CONSOLE [%s]: %s", msg.Type(), msg.Text())
	})

	_, err := page.Goto(hybridServerURL)
	require.NoError(t, err)

	// switch to file tab
	fileTab := page.Locator("#file-tab")
	require.NoError(t, fileTab.Click())

	// create temp file with test content
	tmpFile, err := os.CreateTemp("", "hybrid-file-test-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	testContent := "file upload content with special chars: <>&\"'"
	_, err = tmpFile.WriteString(testContent)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

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

	// verify URL has fragment (client-side encrypted)
	assert.Contains(t, secretLink, "/message/", "URL should contain /message/ path")
	assert.Contains(t, secretLink, "#", "UI file upload URL should have fragment")

	// navigate to message page
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// enter PIN (client-side form uses #client-pin when URL has #key)
	clientPin := page.Locator("#client-pin")
	waitVisible(t, clientPin)
	require.NoError(t, clientPin.Fill(testPin))
	require.NoError(t, page.Locator("#decrypt-btn").Click())

	// for file messages, after decryption a success card appears
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

func TestHybrid_UIFile_SizeValidation(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridServerURL)
	require.NoError(t, err)

	// switch to file tab first
	fileTab := page.Locator("#file-tab")
	require.NoError(t, fileTab.Click())

	// create temp file slightly over 1MB
	tmpFile, err := os.CreateTemp("", "hybrid-test-*.bin")
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

// --- requireHTMX middleware test ---

func TestHybrid_WebWithoutHTMX_Returns400(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	// POST to /generate-link without HX-Request header should return 400
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, hybridServerURL+"/generate-link", strings.NewReader("message=test&pin=12345&exp=15&expUnit=m"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// deliberately NOT setting HX-Request header

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "POST without HX-Request should return 400")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "JavaScript is required", "should indicate JavaScript is required")
}

// --- auth integration tests ---

func TestHybrid_WithAuth_RoundTrip(t *testing.T) {
	cleanup := startHybridAuthServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridAuthServerURL)
	require.NoError(t, err)

	originalMessage := "auth + client-encryption test message"

	// fill in message and PIN
	require.NoError(t, page.Locator("#message").Fill(originalMessage))
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form - should trigger auth popup
	require.NoError(t, page.Locator("button[type='submit']").Click())
	popup := page.Locator("#popup.active")
	waitVisible(t, popup)

	// verify login popup appeared
	visible, err := popup.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "login popup should appear with auth enabled")

	// enter password and submit login
	require.NoError(t, page.Locator("#password").Fill("testpass123"))
	require.NoError(t, page.Locator("#popup button[type='submit']").Click())

	// after successful auth, should see the secure link with fragment
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	t.Logf("secret link: %s", secretLink)

	// verify URL has fragment (client-side encrypted)
	assert.Contains(t, secretLink, "/message/", "URL should contain /message/ path")
	assert.Contains(t, secretLink, "#", "URL should have fragment")

	// navigate to message and decrypt
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	clientPin := page.Locator("#client-pin")
	waitVisible(t, clientPin)
	require.NoError(t, clientPin.Fill(testPin))
	require.NoError(t, page.Locator("#decrypt-btn").Click())

	// verify decryption works
	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Equal(t, originalMessage, content, "decrypted message should match original")
}

func TestHybrid_ReEncryptsAfterValidationError(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridServerURL)
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

	// wait for error to appear
	errorSpan := page.Locator(".error")
	waitVisible(t, errorSpan)

	errorText, err := errorSpan.TextContent()
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(errorText), "expire", "expected expiration validation error")
	t.Logf("got expected validation error: %s", errorText)

	// verify message field was cleared (prevents double-encryption bug)
	msgValue, err := page.Locator("#message").InputValue()
	require.NoError(t, err)
	assert.Empty(t, msgValue, "message field should be cleared after 400 error to prevent double-encryption")

	// fill in a new message
	newMessage := "DIFFERENT message after fixing error"
	require.NoError(t, page.Locator("#message").Fill(newMessage))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("#exp").Clear())
	require.NoError(t, page.Locator("#exp").Fill("15"))
	_, err = page.Locator("#expUnit").SelectOption(playwright.SelectOptionValues{Values: &[]string{"m"}})
	require.NoError(t, err)

	// re-submit
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// wait for success
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	assert.Contains(t, secretLink, "#", "should have encryption key fragment")
	t.Logf("secret link after retry: %s", secretLink)

	// verify the NEW message was encrypted by decrypting it
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	clientPin := page.Locator("#client-pin")
	waitVisible(t, clientPin)
	require.NoError(t, clientPin.Fill(testPin))
	require.NoError(t, page.Locator("#decrypt-btn").Click())

	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)

	// critical assertion: the decrypted content must be the NEW message
	assert.Equal(t, newMessage, content, "decrypted message should be the NEW message, proving re-encryption worked")
	assert.NotEqual(t, originalMessage, content, "should NOT be the original message")
}

func TestHybrid_TextSizeValidation(t *testing.T) {
	cleanup := startHybridServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridServerURL)
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

func TestHybrid_WithAuth_WrongPasswordRetry(t *testing.T) {
	cleanup := startHybridAuthServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(hybridAuthServerURL)
	require.NoError(t, err)

	originalMessage := "message preserved after wrong password"

	// fill in message and PIN
	require.NoError(t, page.Locator("#message").Fill(originalMessage))
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form - should trigger auth popup
	require.NoError(t, page.Locator("button[type='submit']").Click())
	popup := page.Locator("#popup.active")
	waitVisible(t, popup)

	// enter WRONG password
	require.NoError(t, page.Locator("#password").Fill("wrongpassword"))
	require.NoError(t, page.Locator("#popup button[type='submit']").Click())

	// wait for error message in popup
	errorDiv := page.Locator("#popup .form-error")
	waitVisible(t, errorDiv)

	errorText, err := errorDiv.TextContent()
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(errorText), "invalid", "should show invalid password error")
	t.Logf("got expected login error: %s", errorText)

	// now enter CORRECT password
	require.NoError(t, page.Locator("#password").Clear())
	require.NoError(t, page.Locator("#password").Fill("testpass123"))
	require.NoError(t, page.Locator("#popup button[type='submit']").Click())

	// after successful auth, should see the secure link with fragment
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	t.Logf("secret link after retry: %s", secretLink)

	// verify URL has fragment
	assert.Contains(t, secretLink, "#", "URL should have fragment")

	// navigate to message and decrypt to verify original message was preserved
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	clientPin := page.Locator("#client-pin")
	waitVisible(t, clientPin)
	require.NoError(t, clientPin.Fill(testPin))
	require.NoError(t, page.Locator("#decrypt-btn").Click())

	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Equal(t, originalMessage, content, "decrypted message should match original after wrong password retry")
}
