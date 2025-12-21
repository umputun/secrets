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
	emailServerURL  = "http://localhost:18082"
	emailServerPort = ":18082"
)

// startEmailServer starts a server with email enabled (fake SMTP config).
// Returns a cleanup function that stops the server.
func startEmailServer(t *testing.T) func() {
	t.Helper()

	cmd := exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e-email",
		"--domain=localhost"+emailServerPort,
		"--protocol=http",
		"--listen="+emailServerPort,
		"--pinsize=5",
		"--expire=1h",
		"--pinattempts=3",
		"--email.enabled",
		"--email.host=localhost",
		"--email.port=25",
		"--email.from=test@example.com",
		"--dbg",
	)
	// create env without AUTH_HASH to disable auth
	env := []string{}
	for _, e := range os.Environ() {
		if len(e) < 9 || e[:9] != "AUTH_HASH" {
			env = append(env, e)
		}
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start email server: %v", err)
	}

	// wait for server readiness
	if err := waitForEmailServer(emailServerURL+"/ping", 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("email server not ready: %v", err)
	}

	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func waitForEmailServer(url string, timeout time.Duration) error {
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
	return fmt.Errorf("email server not ready after %v", timeout)
}

func TestEmail_ButtonVisible(t *testing.T) {
	cleanup := startEmailServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(emailServerURL)
	require.NoError(t, err)

	// create a secret first
	require.NoError(t, page.Locator("#message").Fill("email test message"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond)

	// check email button is visible (only when email is enabled)
	emailBtn := page.Locator("button:has-text('Email')")
	visible, err := emailBtn.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "email button should be visible when email is enabled")
}

func TestEmail_PopupOpens(t *testing.T) {
	cleanup := startEmailServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(emailServerURL)
	require.NoError(t, err)

	// create a secret
	require.NoError(t, page.Locator("#message").Fill("email popup test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond)

	// click email button
	emailBtn := page.Locator("button:has-text('Email')")
	require.NoError(t, emailBtn.Click())
	time.Sleep(200 * time.Millisecond)

	// check email popup is visible
	popup := page.Locator("#popup.active")
	visible, err := popup.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "email popup should be visible")

	// check form fields are present
	toField := page.Locator("#to")
	visible, err = toField.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "to email field should be visible")

	subjectField := page.Locator("#subject")
	visible, err = subjectField.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "subject field should be visible")
}

func TestEmail_SendFailsWithFakeSMTP(t *testing.T) {
	cleanup := startEmailServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(emailServerURL)
	require.NoError(t, err)

	// create a secret
	require.NoError(t, page.Locator("#message").Fill("email send test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond)

	// click email button to open popup
	emailBtn := page.Locator("button:has-text('Email')")
	require.NoError(t, emailBtn.Click())
	time.Sleep(200 * time.Millisecond)

	// fill email form
	require.NoError(t, page.Locator("#to").Fill("recipient@example.com"))
	require.NoError(t, page.Locator("#subject").Fill("Test subject"))

	// submit the form
	sendBtn := page.Locator("#popup button:has-text('Send')")
	require.NoError(t, sendBtn.Click())
	time.Sleep(500 * time.Millisecond) // wait for SMTP attempt to fail

	// should show error (SMTP connection will fail with fake config)
	errorElement := page.Locator("#popup .form-error")
	visible, err := errorElement.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "error should be visible when SMTP fails")
}

func TestEmail_PopupCancel(t *testing.T) {
	cleanup := startEmailServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(emailServerURL)
	require.NoError(t, err)

	// create a secret
	require.NoError(t, page.Locator("#message").Fill("email cancel test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond)

	// click email button
	emailBtn := page.Locator("button:has-text('Email')")
	require.NoError(t, emailBtn.Click())
	time.Sleep(200 * time.Millisecond)

	// click cancel button
	cancelBtn := page.Locator("#popup button:has-text('Cancel')")
	require.NoError(t, cancelBtn.Click())
	time.Sleep(200 * time.Millisecond)

	// popup should be hidden
	popup := page.Locator("#popup.active")
	visible, err := popup.IsVisible()
	require.NoError(t, err)
	assert.False(t, visible, "popup should be hidden after cancel")
}
