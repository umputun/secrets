//go:build e2e

package e2e

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// click file tab
	fileTab := page.Locator("#file-tab")
	visible, err := fileTab.IsVisible()
	require.NoError(t, err)
	if !visible {
		t.Skip("file tab not visible - files may be disabled")
	}

	require.NoError(t, fileTab.Click())
	time.Sleep(100 * time.Millisecond)

	// check drop zone is now visible
	dropZone := page.Locator("#drop-zone")
	visible, err = dropZone.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "drop zone should be visible after switching to file tab")

	// message textarea should be hidden
	visible, err = page.Locator("#message").IsVisible()
	require.NoError(t, err)
	assert.False(t, visible, "message textarea should be hidden in file mode")
}

func TestFile_SwitchBack(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// click file tab
	fileTab := page.Locator("#file-tab")
	visible, err := fileTab.IsVisible()
	require.NoError(t, err)
	if !visible {
		t.Skip("file tab not visible")
	}

	require.NoError(t, fileTab.Click())
	time.Sleep(100 * time.Millisecond)

	// click text tab to switch back
	textTab := page.Locator("#text-tab")
	require.NoError(t, textTab.Click())
	time.Sleep(100 * time.Millisecond)

	// message textarea should be visible again
	visible, err = page.Locator("#message").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "message textarea should be visible after switching back to text")
}

func TestFile_UploadAndDownload(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// switch to file tab
	fileTab := page.Locator("#file-tab")
	visible, err := fileTab.IsVisible()
	require.NoError(t, err)
	if !visible {
		t.Skip("file tab not visible - files may be disabled")
	}
	require.NoError(t, fileTab.Click())
	time.Sleep(100 * time.Millisecond)

	// create a test file in temp directory
	testContent := []byte("test file content for e2e testing")
	testFilePath := t.TempDir() + "/e2e-test-file.txt"
	require.NoError(t, os.WriteFile(testFilePath, testContent, 0o600))

	// upload file using file input
	fileInput := page.Locator("input[type='file']")
	require.NoError(t, fileInput.SetInputFiles(testFilePath))
	time.Sleep(100 * time.Millisecond)

	// fill PIN
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form
	require.NoError(t, page.Locator("button[type='submit']").Click())
	time.Sleep(300 * time.Millisecond) // file upload takes longer

	// check for secure link result
	linkTextarea := page.Locator("textarea#msg-text")
	visible, err = linkTextarea.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "secure link textarea should be visible after file upload")

	// get the generated link
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	assert.Contains(t, secretLink, "/message/")

	// extract key from link
	messageKey := extractMessageKey(t, secretLink)

	// navigate to message page
	_, err = page.Goto(baseURL + "/message/" + messageKey)
	require.NoError(t, err)

	// verify this is a file download page (button says "Download File")
	downloadBtn := page.Locator("button:has-text('Download File')")
	visible, err = downloadBtn.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "download file button should be visible for file messages")
}

func TestFile_InfoDisplay(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// switch to file tab
	fileTab := page.Locator("#file-tab")
	visible, err := fileTab.IsVisible()
	require.NoError(t, err)
	if !visible {
		t.Skip("file tab not visible - files may be disabled")
	}
	require.NoError(t, fileTab.Click())
	time.Sleep(100 * time.Millisecond)

	// create a test file in temp directory
	testContent := []byte("test file content for display test")
	testFilePath := t.TempDir() + "/e2e-test-display.txt"
	require.NoError(t, os.WriteFile(testFilePath, testContent, 0o600))

	// upload file
	fileInput := page.Locator("input[type='file']")
	require.NoError(t, fileInput.SetInputFiles(testFilePath))
	time.Sleep(100 * time.Millisecond)

	// check file info is displayed in drop zone
	fileInfo := page.Locator("#file-info")
	visible, err = fileInfo.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "file info should be visible after selecting file")

	// check filename is shown
	infoText, err := fileInfo.TextContent()
	require.NoError(t, err)
	assert.Contains(t, infoText, "e2e-test-display.txt", "should show filename")
}
