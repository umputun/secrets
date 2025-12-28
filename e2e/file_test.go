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

const noFilesServerURL = "http://localhost:18083"

// startNoFilesServer starts a server with files disabled on port 18083.
// Returns a cleanup function that stops the server.
func startNoFilesServer(t *testing.T) func() {
	t.Helper()

	cmd := exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e-nofiles",
		"--domain=localhost:18083",
		"--protocol=http",
		"--listen=:18083",
		"--pinsize=5",
		"--expire=1h",
		"--pinattempts=3",
		// files NOT enabled (default is disabled)
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
		t.Fatalf("failed to start no-files server: %v", err)
	}

	// wait for server readiness using shared helper
	if err := waitForServer(noFilesServerURL+"/ping", 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("no-files server not ready: %v", err)
	}

	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func TestFile_TabVisible(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// check file tab is visible (files are enabled in test server)
	visible, err := page.Locator("#file-tab").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "file tab should be visible when files enabled")
}

func TestFile_TabSwitch(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// click file tab (files are enabled in test server config)
	fileTab := page.Locator("#file-tab")
	visible, err := fileTab.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "file tab should be visible - files are enabled in test config")

	require.NoError(t, fileTab.Click())
	dropZone := page.Locator("#drop-zone")
	waitVisible(t, dropZone)

	// message textarea should be hidden
	visible, err = page.Locator("#message").IsVisible()
	require.NoError(t, err)
	assert.False(t, visible, "message textarea should be hidden in file mode")
}

func TestFile_SwitchBack(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// click file tab (files are enabled in test server config)
	fileTab := page.Locator("#file-tab")
	visible, err := fileTab.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "file tab should be visible - files are enabled in test config")

	require.NoError(t, fileTab.Click())
	dropZone := page.Locator("#drop-zone")
	waitVisible(t, dropZone)

	// click text tab to switch back
	textTab := page.Locator("#text-tab")
	require.NoError(t, textTab.Click())
	messageTextarea := page.Locator("#message")
	waitVisible(t, messageTextarea)
}

func TestFile_UploadAndDownload(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// switch to file tab (files are enabled in test server config)
	fileTab := page.Locator("#file-tab")
	visible, err := fileTab.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "file tab should be visible - files are enabled in test config")
	require.NoError(t, fileTab.Click())
	dropZone := page.Locator("#drop-zone")
	waitVisible(t, dropZone)

	// create a test file in temp directory
	testContent := []byte("test file content for e2e testing")
	testFilePath := t.TempDir() + "/e2e-test-file.txt"
	require.NoError(t, os.WriteFile(testFilePath, testContent, 0o600))

	// upload file using file input
	fileInput := page.Locator("input[type='file']")
	require.NoError(t, fileInput.SetInputFiles(testFilePath))
	fileInfo := page.Locator("#file-info")
	waitVisible(t, fileInfo)

	// fill PIN
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	// get the generated link (with #key for client-side decryption)
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	assert.Contains(t, secretLink, "/message/")
	assert.Contains(t, secretLink, "#", "file link should have encryption key fragment")

	// navigate to message page (use full link with #key for client-side decryption)
	_, err = page.Goto(secretLink)
	require.NoError(t, err)

	// enter PIN and trigger client-side decryption (file is embedded in encrypted blob)
	clientPin := page.Locator("#client-pin")
	waitVisible(t, clientPin)
	require.NoError(t, clientPin.Fill(testPin))
	require.NoError(t, page.Locator("#decrypt-btn").Click())

	// wait for file download success card
	successCard := page.Locator(".success-card")
	waitVisible(t, successCard)
}

func TestFile_InfoDisplay(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// switch to file tab (files are enabled in test server config)
	fileTab := page.Locator("#file-tab")
	visible, err := fileTab.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "file tab should be visible - files are enabled in test config")
	require.NoError(t, fileTab.Click())
	dropZone := page.Locator("#drop-zone")
	waitVisible(t, dropZone)

	// create a test file in temp directory
	testContent := []byte("test file content for display test")
	testFilePath := t.TempDir() + "/e2e-test-display.txt"
	require.NoError(t, os.WriteFile(testFilePath, testContent, 0o600))

	// upload file
	fileInput := page.Locator("input[type='file']")
	require.NoError(t, fileInput.SetInputFiles(testFilePath))
	fileInfo := page.Locator("#file-info")
	waitVisible(t, fileInfo)

	// check filename is shown
	infoText, err := fileInfo.TextContent()
	require.NoError(t, err)
	assert.Contains(t, infoText, "e2e-test-display.txt", "should show filename")
}

func TestFile_TabHiddenWhenDisabled(t *testing.T) {
	cleanup := startNoFilesServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(noFilesServerURL)
	require.NoError(t, err)

	// file tab should NOT be visible when files are disabled
	fileTab := page.Locator("#file-tab")
	visible, err := fileTab.IsVisible()
	require.NoError(t, err)
	assert.False(t, visible, "file tab should be hidden when files disabled")

	// message textarea should still be visible
	messageTextarea := page.Locator("#message")
	visible, err = messageTextarea.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "message textarea should be visible")
}
