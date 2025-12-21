//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	authServerURL  = "http://localhost:18081"
	authServerPort = ":18081"
	// bcrypt hash for "testpass123"
	testAuthHash = "$2a$10$q8UbvhCC/LFB4kCxenUGQ.34UuyUVg.7otCerbwj9xkrNXO9Fd2S2"
)

// startAuthServer starts a separate server instance with auth enabled on port 18081.
// Returns a cleanup function that stops the server.
func startAuthServer(t *testing.T) func() {
	t.Helper()

	cmd := exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e-auth",
		"--domain=localhost"+authServerPort,
		"--protocol=http",
		"--listen="+authServerPort,
		"--pinsize=5",
		"--expire=1h",
		"--pinattempts=3",
		"--auth.hash="+testAuthHash,
		"--dbg",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start auth server: %v", err)
	}

	// wait for server readiness
	if err := waitForAuthServer(authServerURL+"/ping", 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("auth server not ready: %v", err)
	}

	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func waitForAuthServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) // #nosec G107 - test url
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("auth server not ready after %v", timeout)
}

func TestAuth_LoginPopupOnGenerate(t *testing.T) {
	cleanup := startAuthServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(authServerURL)
	require.NoError(t, err)

	// fill in message and PIN
	require.NoError(t, page.Locator("#message").Fill("auth test message"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form - should trigger auth popup
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond)

	// check login popup is visible (popup becomes active)
	popup := page.Locator("#popup.active")
	visible, err := popup.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "login popup should appear when auth is enabled")

	// check password input is present
	passwordInput := page.Locator("#password")
	visible, err = passwordInput.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "password input should be visible in login popup")
}

func TestAuth_LoginSuccess(t *testing.T) {
	cleanup := startAuthServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(authServerURL)
	require.NoError(t, err)

	// fill in message and PIN
	require.NoError(t, page.Locator("#message").Fill("auth success test message"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form - should trigger auth popup
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond)

	// enter correct password in the login form
	require.NoError(t, page.Locator("#password").Fill("testpass123"))
	require.NoError(t, page.Locator("#popup form button[type='submit']").Click())
	time.Sleep(500 * time.Millisecond) // wait for auth and form resubmit

	// after successful auth, should see the secure link
	linkTextarea := page.Locator("textarea#msg-text")
	visible, err := linkTextarea.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "secure link should be visible after successful auth")

	// verify it's a valid link
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	assert.Contains(t, secretLink, "/message/")
}

func TestAuth_LoginWrongPassword(t *testing.T) {
	cleanup := startAuthServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(authServerURL)
	require.NoError(t, err)

	// fill in message and PIN
	require.NoError(t, page.Locator("#message").Fill("wrong password test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form - should trigger auth popup
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond)

	// enter wrong password
	require.NoError(t, page.Locator("#password").Fill("wrongpassword"))
	require.NoError(t, page.Locator("#popup form button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond)

	// should show error message
	errorElement := page.Locator("#popup .form-error")
	visible, err := errorElement.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "error should be visible for wrong password")

	// popup should still be visible (not dismissed)
	popup := page.Locator("#popup.active")
	visible, err = popup.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "login popup should remain visible after wrong password")
}

func TestAuth_SessionPersists(t *testing.T) {
	cleanup := startAuthServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(authServerURL)
	require.NoError(t, err)

	// first secret - authenticate
	require.NoError(t, page.Locator("#message").Fill("first secret with auth"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond)

	// enter correct password
	require.NoError(t, page.Locator("#password").Fill("testpass123"))
	require.NoError(t, page.Locator("#popup form button[type='submit']").Click())
	time.Sleep(500 * time.Millisecond)

	// verify first secret created
	linkTextarea := page.Locator("textarea#msg-text")
	visible, err := linkTextarea.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "first secret should be created")

	// go back to home
	_, err = page.Goto(authServerURL)
	require.NoError(t, err)

	// create second secret - should NOT require re-authentication (session persists)
	require.NoError(t, page.Locator("#message").Fill("second secret with existing session"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(500 * time.Millisecond)

	// should directly show secure link without auth popup
	linkTextarea = page.Locator("textarea#msg-text")
	visible, err = linkTextarea.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "second secret should be created without re-authentication")
}

func TestAuth_PopupCancel(t *testing.T) {
	cleanup := startAuthServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(authServerURL)
	require.NoError(t, err)

	// fill in message and PIN
	require.NoError(t, page.Locator("#message").Fill("cancel popup test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form - should trigger auth popup
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond)

	// check login popup is visible
	popup := page.Locator("#popup.active")
	visible, err := popup.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "login popup should be visible")

	// close the popup by clicking the close button
	closeBtn := page.Locator("#popup .close-popup")
	visible, err = closeBtn.IsVisible()
	require.NoError(t, err)
	if visible {
		require.NoError(t, closeBtn.Click())
		time.Sleep(200 * time.Millisecond)

		// popup should be hidden (class "active" removed)
		popup = page.Locator("#popup.active")
		visible, err = popup.IsVisible()
		require.NoError(t, err)
		assert.False(t, visible, "login popup should be hidden after clicking close")
	}
}
