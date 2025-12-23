//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const emailServerURL = "http://localhost:18082"

// startEmailServer starts a server with email enabled (fake SMTP config).
// Returns a cleanup function that stops the server.
func startEmailServer(t *testing.T) func() {
	t.Helper()

	cmd := exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e-email",
		"--domain=localhost:18082",
		"--protocol=http",
		"--listen=:18082",
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
		if !strings.HasPrefix(e, "AUTH_HASH=") {
			env = append(env, e)
		}
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start email server: %v", err)
	}

	// wait for server readiness using shared helper
	if err := waitForServer(emailServerURL+"/ping", 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("email server not ready: %v", err)
	}

	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
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
	emailBtn := page.Locator("button:has-text('Email')")
	waitVisible(t, emailBtn)

	// check email button is visible (only when email is enabled)
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
	emailBtn := page.Locator("button:has-text('Email')")
	waitVisible(t, emailBtn)

	// click email button
	require.NoError(t, emailBtn.Click())
	popup := page.Locator("#popup.active")
	waitVisible(t, popup)

	// check email popup is visible
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
	emailBtn := page.Locator("button:has-text('Email')")
	waitVisible(t, emailBtn)

	// click email button to open popup
	require.NoError(t, emailBtn.Click())
	popup := page.Locator("#popup.active")
	waitVisible(t, popup)

	// fill email form
	require.NoError(t, page.Locator("#to").Fill("recipient@example.com"))
	require.NoError(t, page.Locator("#subject").Fill("Test subject"))

	// submit the form
	sendBtn := page.Locator("#popup button:has-text('Send')")
	require.NoError(t, sendBtn.Click())
	errorElement := page.Locator("#popup .form-error")
	waitVisible(t, errorElement)

	// should show error (SMTP connection will fail with fake config)
	visible, err := errorElement.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "error should be visible when SMTP fails")
	errorText, err := errorElement.TextContent()
	require.NoError(t, err)
	assert.NotEmpty(t, errorText, "error message should have content")
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
	emailBtn := page.Locator("button:has-text('Email')")
	waitVisible(t, emailBtn)

	// click email button
	require.NoError(t, emailBtn.Click())
	popup := page.Locator("#popup.active")
	waitVisible(t, popup)

	// click cancel button
	cancelBtn := page.Locator("#popup button:has-text('Cancel')")
	require.NoError(t, cancelBtn.Click())
	waitHidden(t, popup)

	// popup should be hidden
	visible, err := popup.IsVisible()
	require.NoError(t, err)
	assert.False(t, visible, "popup should be hidden after cancel")
}
