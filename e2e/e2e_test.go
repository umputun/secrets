//go:build e2e

// Package e2e contains end-to-end tests for the secrets web application.
// Tests are organized into three files:
//   - e2e_test.go: setup, helpers, and basic functionality tests
//   - file_test.go: file upload and download tests
//   - auth_test.go: authentication tests
package e2e

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL = "http://localhost:18080"
	testPin = "12345"
)

var (
	pw        *playwright.Playwright
	browser   playwright.Browser
	serverCmd *exec.Cmd
)

func TestMain(m *testing.M) {
	// build test binary from project root
	build := exec.Command("go", "build", "-o", "/tmp/secrets-e2e", "./app")
	build.Dir = ".." // run from e2e directory, so go up to project root
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Printf("failed to build: %v\n", err)
		os.Exit(1)
	}

	// start server with test config
	serverCmd = exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e",
		"--domain=localhost:18080",
		"--protocol=http",
		"--listen=:18080",
		"--pinsize=5",
		"--expire=1h",
		"--pinattempts=3",
		"--files.enabled",
		"--files.max-size=1048576",
		"--dbg",
	)
	serverCmd.Env = append(os.Environ(),
		"AUTH_HASH=", // disable auth for basic tests
	)
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	if err := serverCmd.Start(); err != nil {
		fmt.Printf("failed to start server: %v\n", err)
		os.Exit(1)
	}

	// wait for server readiness
	if err := waitForServer(baseURL+"/ping", 30*time.Second); err != nil {
		fmt.Printf("server not ready: %v\n", err)
		_ = serverCmd.Process.Kill()
		os.Exit(1)
	}

	// install playwright browsers
	if err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
	}); err != nil {
		fmt.Printf("failed to install playwright: %v\n", err)
		_ = serverCmd.Process.Kill()
		os.Exit(1)
	}

	// start playwright
	var err error
	pw, err = playwright.Run()
	if err != nil {
		fmt.Printf("failed to start playwright: %v\n", err)
		_ = serverCmd.Process.Kill()
		os.Exit(1)
	}

	// launch browser once (reused across all tests via contexts)
	headless := os.Getenv("E2E_HEADLESS") != "false"
	browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		SlowMo: playwright.Float(func() float64 {
			if headless {
				return 0
			}
			return 50 // 50ms slowdown for UI mode
		}()),
	})
	if err != nil {
		fmt.Printf("failed to launch browser: %v\n", err)
		_ = pw.Stop()
		_ = serverCmd.Process.Kill()
		os.Exit(1)
	}

	// run tests
	code := m.Run()

	// cleanup
	_ = browser.Close()
	_ = pw.Stop()
	_ = serverCmd.Process.Kill()
	_ = serverCmd.Wait()

	os.Exit(code)
}

func waitForServer(url string, timeout time.Duration) error { //nolint:unparam // timeout kept for API flexibility
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
	return fmt.Errorf("server not ready after %v", timeout)
}

func newPage(t *testing.T) playwright.Page {
	t.Helper()
	// create new context for test isolation (cookies, storage, etc.)
	ctx, err := browser.NewContext()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctx.Close() })

	page, err := ctx.NewPage()
	require.NoError(t, err)
	return page
}

// waitVisible waits for a locator to become visible, replacing time.Sleep() patterns
func waitVisible(t *testing.T, loc playwright.Locator) {
	t.Helper()
	require.NoError(t, loc.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateVisible,
	}))
}

// waitHidden waits for a locator to become hidden
func waitHidden(t *testing.T, loc playwright.Locator) {
	t.Helper()
	require.NoError(t, loc.WaitFor(playwright.LocatorWaitForOptions{
		State: playwright.WaitForSelectorStateHidden,
	}))
}

// extractMessageKey extracts the message key from a secret link URL
func extractMessageKey(t *testing.T, secretLink string) string {
	t.Helper()
	re := regexp.MustCompile(`/message/([a-zA-Z0-9-]+)`)
	matches := re.FindStringSubmatch(secretLink)
	require.Len(t, matches, 2, "should extract message key from link")
	return matches[1]
}

// --- home page tests ---

func TestHome_PageLoads(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	title, err := page.Title()
	require.NoError(t, err)
	assert.Contains(t, title, "Secret")
}

func TestHome_FormElements(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// check message textarea is visible
	visible, err := page.Locator("#message").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "message textarea should be visible")

	// check PIN input is visible
	visible, err = page.Locator("#pin").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "PIN input should be visible")

	// check expiration input is visible
	visible, err = page.Locator("#exp").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "expiration input should be visible")

	// check submit button is visible
	visible, err = page.Locator("button[type='submit']").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "submit button should be visible")
}

// --- secret creation and reveal tests ---

func TestSecret_CreateAndReveal(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// fill in message
	require.NoError(t, page.Locator("#message").Fill("test secret message for e2e"))

	// fill in PIN
	require.NoError(t, page.Locator("#pin").Fill(testPin))

	// submit form and wait for result
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	// get the generated link
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)
	assert.Contains(t, secretLink, "/message/")

	// extract key from link
	messageKey := extractMessageKey(t, secretLink)

	// navigate to message page
	_, err = page.Goto(baseURL + "/message/" + messageKey)
	require.NoError(t, err)

	// check PIN input is visible on message page
	visible, err := page.Locator("#pin").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "PIN input should be visible on message page")

	// enter PIN to reveal message
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	content, err := messageText.InputValue()
	require.NoError(t, err)
	assert.Contains(t, content, "test secret message for e2e")
}

func TestSecret_WrongPin(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// create secret
	require.NoError(t, page.Locator("#message").Fill("secret with wrong pin test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	// get the generated link
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// extract key from link
	messageKey := extractMessageKey(t, secretLink)

	// navigate to message page
	_, err = page.Goto(baseURL + "/message/" + messageKey)
	require.NoError(t, err)

	// enter wrong PIN
	require.NoError(t, page.Locator("#pin").Fill("99999"))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	errorSpan := page.Locator(".error")
	waitVisible(t, errorSpan)

	errorText, err := errorSpan.TextContent()
	require.NoError(t, err)
	assert.Contains(t, errorText, "wrong pin")
}

func TestSecret_MaxAttempts(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// create secret
	require.NoError(t, page.Locator("#message").Fill("max attempts test message"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	// get the generated link
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// extract key from link
	messageKey := extractMessageKey(t, secretLink)

	// try wrong PIN 3 times (max attempts)
	for range 3 {
		_, err = page.Goto(baseURL + "/message/" + messageKey)
		require.NoError(t, err)
		require.NoError(t, page.Locator("#pin").Fill("99999"))
		require.NoError(t, page.Locator("button[type='submit']").Click())
		errorSpan := page.Locator(".error")
		waitVisible(t, errorSpan)
	}

	// after max attempts, message should be deleted
	// navigate again and check for error
	_, err = page.Goto(baseURL + "/message/" + messageKey)
	require.NoError(t, err)
	require.NoError(t, page.Locator("#pin").Fill(testPin)) // even correct PIN
	require.NoError(t, page.Locator("button[type='submit']").Click())
	errorCard := page.Locator(".card:has-text('Message Unavailable')")
	waitVisible(t, errorCard)
}

func TestSecret_AlreadyViewed(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// create secret
	require.NoError(t, page.Locator("#message").Fill("one-time secret"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	// get the generated link
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// extract key from link
	messageKey := extractMessageKey(t, secretLink)

	// first access - should succeed
	_, err = page.Goto(baseURL + "/message/" + messageKey)
	require.NoError(t, err)
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	messageText := page.Locator("textarea#decoded-msg-text")
	waitVisible(t, messageText)

	// second access - should fail (message deleted after first view)
	_, err = page.Goto(baseURL + "/message/" + messageKey)
	require.NoError(t, err)
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	errorCard := page.Locator(".card:has-text('Message Unavailable')")
	waitVisible(t, errorCard)
}

// --- theme tests ---

func TestTheme_Toggle(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// get initial theme
	initialTheme, err := page.Locator("html").GetAttribute("data-theme")
	require.NoError(t, err)

	// find and click theme toggle button (should always be present)
	themeBtn := page.Locator("button[hx-post='/theme']")
	visible, err := themeBtn.IsVisible()
	require.NoError(t, err)
	require.True(t, visible, "theme toggle button should be visible")

	require.NoError(t, themeBtn.Click())
	// wait for page to fully reload by checking for message form
	messageTextarea := page.Locator("#message")
	waitVisible(t, messageTextarea)

	// check theme changed
	newTheme, err := page.Locator("html").GetAttribute("data-theme")
	require.NoError(t, err)
	assert.NotEmpty(t, newTheme, "theme attribute should be set after toggle")
	assert.NotEqual(t, initialTheme, newTheme, "theme should change after toggle")
}

// --- about page tests ---

func TestAbout_PageLoads(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL + "/about")
	require.NoError(t, err)

	title, err := page.Title()
	require.NoError(t, err)
	assert.Contains(t, title, "How It Works")

	// check page content is present
	heading := page.Locator("h1, h2").First()
	visible, err := heading.IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "heading should be visible on about page")
}

// --- navigation tests ---

func TestNavigation_HomeLink(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL + "/about")
	require.NoError(t, err)

	// click on branding/home link
	homeLink := page.Locator("a.brand-link")
	visible, err := homeLink.IsVisible()
	require.NoError(t, err)
	if !visible {
		// try alternative selector
		homeLink = page.Locator("a[href='/']").First()
	}

	require.NoError(t, homeLink.Click())
	// wait for navigation to complete by waiting for the message form to appear
	messageTextarea := page.Locator("#message")
	waitVisible(t, messageTextarea)

	// verify we're on home page
	currentURL := page.URL()
	assert.Equal(t, baseURL+"/", currentURL)
}

// --- validation tests ---

func TestValidation_EmptyMessage(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// try to submit with empty message
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	// html5 validation will prevent submission - no async action happens

	// should stay on same page, check we're still on form
	visible, err := page.Locator("#message").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "should stay on form with empty message")
}

func TestValidation_EmptyPin(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// fill message but not PIN
	require.NoError(t, page.Locator("#message").Fill("test message"))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	// html5 validation will prevent submission - no async action happens

	// should stay on same page
	visible, err := page.Locator("#pin").IsVisible()
	require.NoError(t, err)
	assert.True(t, visible, "should stay on form with empty PIN")
}

func TestValidation_InvalidPinFormat(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// fill message and invalid PIN (letters instead of numbers)
	require.NoError(t, page.Locator("#message").Fill("test message"))
	require.NoError(t, page.Locator("#pin").Fill("abcde"))

	// the oninput handler should strip non-numeric characters
	pinValue, err := page.Locator("#pin").InputValue()
	require.NoError(t, err)
	assert.Empty(t, pinValue, "PIN input should strip non-numeric characters")
}

// --- copy link tests ---

func TestCopyLink_ButtonVisible(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// create secret first
	require.NoError(t, page.Locator("#message").Fill("copy test message"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	copyBtn := page.Locator("button#copy-link-btn")
	waitVisible(t, copyBtn)
}

// --- new secret button tests ---

func TestNewSecret_ButtonAfterCreate(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// create secret
	require.NoError(t, page.Locator("#message").Fill("new secret test"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	newBtn := page.Locator("a:has-text('New')")
	waitVisible(t, newBtn)

	// click it and verify we're back to form
	require.NoError(t, newBtn.Click())
	messageTextarea := page.Locator("#message")
	waitVisible(t, messageTextarea)
}

// --- expiration tests ---

func TestExpiration_UnitSelection(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// check default unit
	expUnit := page.Locator("#expUnit")
	value, err := expUnit.InputValue()
	require.NoError(t, err)
	assert.Equal(t, "m", value, "default expiration unit should be minutes")

	// change to hours
	_, err = expUnit.SelectOption(playwright.SelectOptionValues{Values: &[]string{"h"}})
	require.NoError(t, err)
	value, err = expUnit.InputValue()
	require.NoError(t, err)
	assert.Equal(t, "h", value, "should be able to select hours")

	// change to days
	_, err = expUnit.SelectOption(playwright.SelectOptionValues{Values: &[]string{"d"}})
	require.NoError(t, err)
	value, err = expUnit.InputValue()
	require.NoError(t, err)
	assert.Equal(t, "d", value, "should be able to select days")
}

func TestValidation_PinTooShort(t *testing.T) {
	page := newPage(t)
	_, err := page.Goto(baseURL)
	require.NoError(t, err)

	// fill message and short PIN (test server expects 5 digits)
	require.NoError(t, page.Locator("#message").Fill("test message"))
	require.NoError(t, page.Locator("#pin").Fill("1234")) // only 4 digits, need 5

	// submit form
	require.NoError(t, page.Locator("button[type='submit']").Click())

	// should show validation error (PIN has minlength=5 in HTML)
	// the form should not submit successfully - check that link textarea is NOT shown
	linkTextarea := page.Locator("textarea#msg-text")

	// use a short timeout to check non-visibility
	visible, _ := linkTextarea.IsVisible()
	assert.False(t, visible, "secret link should not appear with invalid PIN length")
}

const multiDomainServerURL = "http://localhost:18084"

// startMultiDomainServer starts a server with multiple domains configured.
// Returns a cleanup function that stops the server.
func startMultiDomainServer(t *testing.T) func() {
	t.Helper()

	cmd := exec.Command("/tmp/secrets-e2e",
		"--key=test-sign-key-for-e2e-multi",
		"--domain=localhost:18084,127.0.0.1:18084",
		"--protocol=http",
		"--listen=:18084",
		"--pinsize=5",
		"--expire=1h",
		"--pinattempts=3",
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
		t.Fatalf("failed to start multi-domain server: %v", err)
	}

	// wait for server readiness using shared helper
	if err := waitForServer(multiDomainServerURL+"/ping", 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("multi-domain server not ready: %v", err)
	}

	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func TestSecret_MultiDomainLinkGeneration(t *testing.T) {
	cleanup := startMultiDomainServer(t)
	defer cleanup()

	page := newPage(t)
	_, err := page.Goto(multiDomainServerURL)
	require.NoError(t, err)

	// create a secret
	require.NoError(t, page.Locator("#message").Fill("multi-domain test message"))
	require.NoError(t, page.Locator("#pin").Fill(testPin))
	require.NoError(t, page.Locator("button[type='submit']").Click())
	linkTextarea := page.Locator("textarea#msg-text")
	waitVisible(t, linkTextarea)

	// get the generated link
	secretLink, err := linkTextarea.InputValue()
	require.NoError(t, err)

	// verify link uses the correct domain (first domain in the list or request domain)
	assert.Contains(t, secretLink, "/message/", "link should contain /message/ path")
	assert.True(t,
		strings.Contains(secretLink, "localhost:18084") || strings.Contains(secretLink, "127.0.0.1:18084"),
		"link should use one of the configured domains")
}
