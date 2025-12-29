package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/secrets/v2/app/email"
	"github.com/umputun/secrets/v2/app/messager"
	"github.com/umputun/secrets/v2/app/server/assets"
	"github.com/umputun/secrets/v2/app/server/validator"
	"github.com/umputun/secrets/v2/app/store"
)

const (
	baseTmpl  = "base"
	mainTmpl  = "main"
	errorTmpl = "error"

	msgKey       = "message"
	pinKey       = "pin"
	expKey       = "exp"
	expUnitKey   = "expUnit"
	pathKeyParam = "key" // URL path parameter for message key
)

type createMsgForm struct {
	Message  string
	Exp      int
	MaxExp   string
	ExpUnit  string
	IsFile   bool   // true if this is a file upload
	FileName string // original filename for display
	FileSize int64  // file size in bytes for display
	validator.Validator
}

type showMsgForm struct {
	Key     string
	Message string
	validator.Validator
}

// emailPopupData contains data for email popup template rendering
type emailPopupData struct {
	Link     string
	Subject  string
	FromName string
	Error    string
	To       string // preserved on validation error
}

type templateData struct {
	Form           any
	PinSize        int
	CurrentYear    int
	Theme          string
	Branding       string
	URL            string // canonical URL for the page
	BaseURL        string // base URL for the site (protocol://domain)
	PageTitle      string // SEO-optimized page title
	PageDesc       string // page description for meta tags
	BreadcrumbName string // breadcrumb name for current page (empty for home)
	IsMessagePage  bool   // true for message pages (should not be indexed)
	FilesEnabled   bool   // true if file uploads are enabled
	MaxFileSize    int64  // max file size in bytes
	IsFile         bool   // true if the message is a file (for show-message template)
	AllowNoPin     bool   // true if PIN-less secrets are allowed
	HasPin         bool   // true if the message requires PIN (for show-message template)
}

// render renders a template
func (s Server) render(w http.ResponseWriter, status int, page, tmplName string, data any) {
	ts, ok := s.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		log.Printf("[ERROR] %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)

	if tmplName == "" {
		tmplName = baseTmpl
	}

	err := ts.ExecuteTemplate(buf, tmplName, data)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, err = buf.WriteTo(w)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// renders the home page
// GET /
func (s Server) indexCtrl(w http.ResponseWriter, r *http.Request) {
	data := s.newTemplateData(r, createMsgForm{
		Exp:    15,
		MaxExp: humanDuration(s.cfg.MaxExpire),
	})
	data.PageTitle = "Secret Sharing - Self-Destructing Encrypted Messages"
	data.PageDesc = "Share sensitive information securely with self-destructing messages protected by PIN codes. Free, open-source, and privacy-focused."

	s.render(w, http.StatusOK, "home.tmpl.html", baseTmpl, data)
}

// renders the generate link page
// POST /generate-link
// Request Body: This function expects a POST request body containing the following fields:
//   - "message" (string): The message content to be associated with the secure link.
//   - "expUnit" (string): The unit of expiration time (e.g., "m" for minutes, "h" for hours, "d" for days).
//   - "pin" (slice of strings): An array of PIN values.
//
// For file uploads (multipart/form-data):
//   - "file" (file): The file to upload
func (s Server) generateLinkCtrl(w http.ResponseWriter, r *http.Request) {
	// check auth if enabled
	if s.cfg.AuthHash != "" && !s.isAuthenticated(r) {
		s.renderLoginPopupWithStatus(w, r, "", http.StatusUnauthorized)
		return
	}

	// reject multipart requests - all file uploads must use client-side JS encryption
	// (JS encrypts file into blob and sends as text)
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		log.Printf("[WARN] multipart upload rejected, use JS encryption")
		s.render(w, http.StatusBadRequest, "error.tmpl.html", errorTmpl, "file uploads require JavaScript encryption")
		return
	}

	// handle text message (existing logic)
	err := r.ParseForm()
	if err != nil {
		s.render(w, http.StatusOK, "error.tmpl.html", errorTmpl, err.Error())
		return
	}

	form := createMsgForm{
		Message: r.PostForm.Get(msgKey),
		ExpUnit: r.PostForm.Get(expUnitKey),
		MaxExp:  humanDuration(s.cfg.MaxExpire),
	}

	pinValues := r.Form["pin"]
	pin := strings.Join(pinValues, "")
	pinIsEmpty := pin == ""

	// validate PIN: skip validation if AllowNoPin and PIN is empty, otherwise require valid digits
	if !pinIsEmpty {
		if err := validatePIN(pin, pinValues, s.cfg.PinSize); err != "" {
			form.AddFieldError(pinKey, err)
		}
	}
	if pinIsEmpty && !s.cfg.AllowNoPin {
		form.AddFieldError(pinKey, fmt.Sprintf("Pin must be %d digits long", s.cfg.PinSize))
	}

	form.CheckField(validator.NotBlank(form.Message), msgKey, "Message can't be empty")
	form.CheckField(validator.IsBase64URL(form.Message), msgKey, "invalid encrypted format")

	exp := r.PostFormValue(expKey)
	form.CheckField(validator.NotBlank(exp), expKey, "Expire can't be empty")
	form.CheckField(validator.IsNumber(exp), expKey, "Expire must be a number")
	form.CheckField(slices.Contains([]string{"m", "h", "d"}, form.ExpUnit), expUnitKey, "Only Minutes, Hours and Days are allowed")

	expInt, err := strconv.Atoi(exp)
	if err != nil {
		form.AddFieldError(expKey, "Expire must be a number")
	}
	form.Exp = expInt
	expDuration := duration(expInt, r.PostFormValue(expUnitKey))

	form.CheckField(validator.MaxDuration(expDuration, s.cfg.MaxExpire), expKey, "Expire must be less than "+humanDuration(s.cfg.MaxExpire))

	if !form.Valid() {
		data := s.newTemplateData(r, form)

		// return 400 for htmx to handle with hx-target-400
		if r.Header.Get("HX-Request") == "true" {
			s.render(w, http.StatusBadRequest, "home.tmpl.html", mainTmpl, data)
		} else {
			s.render(w, http.StatusOK, "home.tmpl.html", mainTmpl, data)
		}
		return
	}

	msg, err := s.messager.MakeMessage(r.Context(), messager.MsgReq{
		Duration:      expDuration,
		Message:       form.Message,
		Pin:           pin,
		ClientEnc:     true, // UI always uses client-side encryption
		AllowEmptyPin: s.cfg.AllowNoPin && pinIsEmpty,
	})
	if err != nil {
		s.render(w, http.StatusOK, "secure-link.tmpl.html", errorTmpl, err.Error())
		return
	}

	log.Printf("[INFO] created message %s, type=text, size=%d, exp=%s, ip=%s",
		msg.Key, len(form.Message), msg.Exp.Format(time.RFC3339), GetHashedIP(r))
	s.renderSecureLink(w, r, msg.Key, form)
}

// renderSecureLink renders the secure link page with the generated URL
func (s Server) renderSecureLink(w http.ResponseWriter, r *http.Request, key string, form createMsgForm) {
	validatedHost := s.getValidatedHost(r)

	// ensure IPv6 addresses are properly bracketed for URL construction
	host, port, err := net.SplitHostPort(validatedHost)
	if err != nil {
		if ip := net.ParseIP(validatedHost); ip != nil && ip.To4() == nil {
			validatedHost = "[" + validatedHost + "]"
		}
	} else {
		if ip := net.ParseIP(host); ip != nil && ip.To4() == nil && !strings.HasPrefix(host, "[") {
			validatedHost = "[" + host + "]:" + port
		}
	}

	msgURL := (&url.URL{
		Scheme: s.cfg.Protocol,
		Host:   validatedHost,
		Path:   path.Join("/message", key),
	}).String()

	// pass form data for file info display in template
	data := struct {
		URL          string
		IsFile       bool
		FileName     string
		FileSize     int64
		EmailEnabled bool
	}{
		URL:          msgURL,
		IsFile:       form.IsFile,
		FileName:     form.FileName,
		FileSize:     form.FileSize,
		EmailEnabled: s.cfg.EmailEnabled && s.emailSender != nil,
	}

	s.render(w, http.StatusOK, "secure-link.tmpl.html", "secure-link", data)
}

// renders the show decoded message page
// GET /message/{key}
// URL Parameters:
//   - "key" (string): A path parameter representing the unique key of the message to be displayed.
func (s Server) showMessageViewCtrl(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue(pathKeyParam)

	// set X-Robots-Tag header for defense-in-depth beyond HTML meta
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")

	data := s.newTemplateData(r, showMsgForm{
		Key: key,
	})
	data.IsMessagePage = true                         // prevent indexing of sensitive message pages
	data.IsFile = s.messager.IsFile(r.Context(), key) // check if message is a file for button label

	// check if message requires PIN (for conditional UI rendering)
	// default to true (show PIN form) if message not found - let loadMessageCtrl handle the error
	hasPin, err := s.messager.HasPin(r.Context(), key)
	if err != nil {
		hasPin = true // default to PIN form if message not found
	}
	data.HasPin = hasPin

	s.render(w, http.StatusOK, "show-message.tmpl.html", baseTmpl, data)
}

// renders the about page
// GET /about
func (s Server) aboutViewCtrl(w http.ResponseWriter, r *http.Request) {
	data := s.newTemplateData(r, nil)
	data.PageTitle = "How It Works - Encrypted Message Sharing"
	data.PageDesc = "Learn how SafeSecret protects your sensitive information with PIN-protected encryption, self-destructing messages, and zero-knowledge architecture."
	data.BreadcrumbName = "How it works"
	s.render(w, http.StatusOK, "about.tmpl.html", baseTmpl, data)
}

// renders the decoded message page or serves file download
// POST /load-message
// Request Body: This function expects a POST request body containing the following fields:
//   - "key" (string): A path parameter representing the unique key of the message to be displayed.
//   - "pin" (slice of strings): An array of PIN values.
func (s Server) loadMessageCtrl(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.render(w, http.StatusOK, "error.tmpl.html", errorTmpl, err.Error())
		return
	}

	form := showMsgForm{
		Key: r.PostForm.Get("key"),
	}

	// check if message requires PIN before validating PIN input
	// default to true (require PIN) if HasPin fails - let LoadMessage handle the actual error
	hasPin, hasPinErr := s.messager.HasPin(r.Context(), form.Key)
	if hasPinErr != nil {
		hasPin = true // default to PIN form if message not found
	}

	pinValues := r.Form["pin"]
	pin := strings.Join(pinValues, "")

	// validate PIN only if message requires it (or if we couldn't check)
	if hasPin {
		if err := validatePIN(pin, pinValues, s.cfg.PinSize); err != "" {
			form.AddFieldError(pinKey, err)
		}
	}

	if !form.Valid() {
		data := s.newTemplateData(r, form)
		data.HasPin = hasPin

		s.render(w, http.StatusOK, "show-message.tmpl.html", mainTmpl, data)
		return
	}

	// check if message is file BEFORE loading (LoadMessage may delete on max attempts)
	isFile := s.messager.IsFile(r.Context(), form.Key)

	msg, err := s.messager.LoadMessage(r.Context(), form.Key, pin)
	if err != nil {
		s.handleLoadMessageError(w, r, &form, err, isFile)
		return
	}

	// client-encrypted message: return raw encrypted blob for client-side decryption
	if msg.ClientEnc {
		// if this is an HTMX request (from server-side form), the user doesn't have the decryption key
		// this happens when URL fragment (#key) was stripped during sharing
		if r.Header.Get("HX-Request") == "true" {
			log.Printf("[WARN] accessed client-enc message %s via HTMX (missing key), ip=%s", form.Key, GetHashedIP(r))
			s.render(w, http.StatusOK, "error.tmpl.html", errorTmpl,
				"The decryption key is missing from the URL. This can happen when sharing links through apps that strip URL fragments. Please ask the sender to share the complete link including the # portion.")
			return
		}
		// non-HTMX request (from client-side JS with fetch) - return blob for decryption
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(msg.Data)
		log.Printf("[INFO] accessed message %s, type=client-enc, status=200 (success), ip=%s", form.Key, GetHashedIP(r))
		return
	}

	// check if decrypted data is a file message
	if messager.IsFileMessage(msg.Data) {
		// reject file messages when files are disabled
		if !s.cfg.EnableFiles {
			log.Printf("[WARN] file download rejected for %s, files disabled", form.Key)
			s.render(w, http.StatusForbidden, "error.tmpl.html", errorTmpl, "file downloads disabled")
			return
		}

		filename, _, dataStart := messager.ParseFileHeader(msg.Data)
		if dataStart < 0 {
			log.Printf("[ERROR] failed to parse file header for %s", form.Key)
			s.render(w, http.StatusOK, "error.tmpl.html", errorTmpl, "invalid file format")
			return
		}

		// serve file directly as download
		// sanitize filename for Content-Disposition to prevent header injection
		safeName := strings.Map(func(r rune) rune {
			if r < 32 || r == '"' || r == '\\' || r > 127 {
				return '_'
			}
			return r
		}, filename)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", safeName))
		// force binary download to prevent browser content interpretation (XSS mitigation)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(msg.Data)-dataStart))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(msg.Data[dataStart:])
		log.Printf("[INFO] accessed message %s, type=file, status=200 (success), ip=%s", form.Key, GetHashedIP(r))
		return
	}

	// text message - render decoded message template
	s.render(w, http.StatusOK, "decoded-message.tmpl.html", "decoded-message", string(msg.Data))
	log.Printf("[INFO] accessed message %s, type=text, status=200 (success), ip=%s", form.Key, GetHashedIP(r))
}

// handleLoadMessageError handles errors from LoadMessage, rendering appropriate responses.
// isFile indicates whether the message was a file (checked before LoadMessage which may delete it).
func (s Server) handleLoadMessageError(w http.ResponseWriter, r *http.Request, form *showMsgForm, err error, isFile bool) {
	isHTMX := r.Header.Get("HX-Request") == "true"

	msgType := "text"
	if isFile {
		msgType = "file"
	}

	if errors.Is(err, messager.ErrExpired) || errors.Is(err, store.ErrLoadRejected) {
		// message not found or expired - return 404
		// always use proper status code (JS fetch handles it correctly)
		s.render(w, http.StatusNotFound, "error.tmpl.html", errorTmpl, err.Error())
		log.Printf("[INFO] accessed message %s, type=%s, status=404 (not found), ip=%s", form.Key, msgType, GetHashedIP(r))
		return
	}

	// wrong PIN - add error to form
	form.AddFieldError("pin", err.Error())
	data := s.newTemplateData(r, form)
	data.IsFile = isFile // use pre-checked value (message may be deleted after max attempts)
	data.HasPin = true   // message has PIN (otherwise we wouldn't get wrong PIN error)
	tmpl := baseTmpl     // full page for non-HTMX (JS fetch handles full page)
	if isHTMX {
		tmpl = mainTmpl // partial for HTMX swap
	}
	// always return 403 for wrong PIN - both HTMX and JS fetch handle it correctly
	s.render(w, http.StatusForbidden, "show-message.tmpl.html", tmpl, data)
	log.Printf("[INFO] accessed message %s, type=%s, status=403 (wrong pin), ip=%s", form.Key, msgType, GetHashedIP(r))
}

// duration converts a number and unit into a time.Duration
func duration(n int, unit string) time.Duration {
	switch unit {
	case "m":
		return time.Duration(n) * time.Minute
	case "h":
		return time.Duration(n) * time.Hour
	case "d":
		return time.Duration(n*24) * time.Hour
	default:
		return time.Duration(0)
	}
}

// validatePIN checks if PIN has correct length and contains only digits.
// returns error message or empty string if valid.
func validatePIN(pin string, pinValues []string, pinSize int) string {
	if len(pin) != pinSize {
		return fmt.Sprintf("Pin must be exactly %d digits", pinSize)
	}
	for _, p := range pinValues {
		if validator.Blank(p) || !validator.IsNumber(p) {
			return fmt.Sprintf("Pin must be %d digits long without empty values", pinSize)
		}
	}
	return ""
}

// humanDuration converts a time.Duration into a human readable string like "5 minutes"
func humanDuration(d time.Duration) string {
	pluralPostfix := func(val time.Duration) string {
		if val == 1 {
			return ""
		}
		return "s"
	}

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d second%s", d/time.Second, pluralPostfix(d/time.Second))
	case d < time.Hour:
		return fmt.Sprintf("%d minute%s", d/time.Minute, pluralPostfix(d/time.Minute))
	case d < time.Hour*24:
		return fmt.Sprintf("%d hour%s", d/time.Hour, pluralPostfix(d/time.Hour))
	default:
		return fmt.Sprintf("%d day%s", d/(time.Hour*24), pluralPostfix(d/(time.Hour*24)))
	}
}

// getTheme gets the theme from cookie or returns "auto" as default
func getTheme(r *http.Request) string {
	cookie, err := r.Cookie("theme")
	if err != nil {
		return "auto" // default theme - respects system preference
	}
	// validate theme value
	switch cookie.Value {
	case "light", "dark":
		return cookie.Value
	default:
		return "auto"
	}
}

// themeToggleCtrl handles theme switching
// POST /theme
func (s Server) themeToggleCtrl(w http.ResponseWriter, r *http.Request) {
	currentTheme := getTheme(r)

	// toggle between light and dark themes
	var nextTheme string
	switch currentTheme {
	case "light":
		nextTheme = "dark"
	default: // "dark" or "auto" (first visit)
		nextTheme = "light"
	}

	// set cookie (client-side storage)
	http.SetCookie(w, &http.Cookie{
		Name:     "theme",
		Value:    nextTheme,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60, // 1 year
		HttpOnly: false,              // allow JS access for immediate UI update if needed
		SameSite: http.SameSiteLaxMode,
	})

	// trigger full page refresh to apply new theme
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

// copyFeedbackCtrl shows copy feedback popup
// POST /copy-feedback
func (s Server) copyFeedbackCtrl(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.render(w, http.StatusOK, "popup.tmpl.html", "popup-closed", nil)
		return
	}

	copyType := r.PostForm.Get("type")
	// sanitize copyType to prevent XSS
	if copyType != "Link" && copyType != "Message" {
		copyType = "Content"
	}

	// pass structured data to template for safe rendering
	data := struct {
		CopyType string
	}{
		CopyType: copyType,
	}

	// trigger auto-close using HX-Trigger header
	w.Header().Set("HX-Trigger-After-Settle", `{"closePopup": true}`)
	s.render(w, http.StatusOK, "popup.tmpl.html", "popup", data)
}

// closePopupCtrl closes the popup
// GET /close-popup
func (s Server) closePopupCtrl(w http.ResponseWriter, _ *http.Request) {
	s.render(w, http.StatusOK, "popup.tmpl.html", "popup-closed", nil)
}

// emailPopupCtrl renders the email popup with preview
// GET /email-popup?link=...
func (s Server) emailPopupCtrl(w http.ResponseWriter, r *http.Request) {
	// check auth if enabled - email sharing requires same auth as secret creation
	if s.cfg.AuthHash != "" && !s.isAuthenticated(r) {
		s.renderLoginPopupWithStatus(w, r, "", http.StatusUnauthorized)
		return
	}

	if s.emailSender == nil {
		http.Error(w, "email not configured", http.StatusServiceUnavailable)
		return
	}

	link := r.URL.Query().Get("link")
	if link == "" {
		http.Error(w, "link parameter required", http.StatusBadRequest)
		return
	}

	// validate the link points to this server to prevent phishing relay
	if !s.isValidSecretLink(link) {
		http.Error(w, "invalid link", http.StatusBadRequest)
		return
	}

	// get default from name from email sender
	defaultFromName := s.emailSender.GetDefaultFromName()

	s.render(w, http.StatusOK, "email-popup.tmpl.html", "email-popup", emailPopupData{
		Link:     link,
		Subject:  "Someone shared a secret with you",
		FromName: defaultFromName,
	})
}

// sendEmailCtrl sends the email with the secret link
// POST /send-email
func (s Server) sendEmailCtrl(w http.ResponseWriter, r *http.Request) { //nolint:gocyclo // linear validation logic
	// check auth if enabled - email sharing requires same auth as secret creation
	if s.cfg.AuthHash != "" && !s.isAuthenticated(r) {
		s.renderLoginPopupWithStatus(w, r, "", http.StatusUnauthorized)
		return
	}

	if s.emailSender == nil {
		http.Error(w, "email not configured", http.StatusServiceUnavailable)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	link := r.FormValue("link")
	subject := r.FormValue("subject")
	fromName := r.FormValue("from_name")
	to := strings.TrimSpace(r.FormValue("to"))

	// helper to render validation error (returns 200 so HTMX processes the response)
	renderError := func(errMsg string) {
		s.render(w, http.StatusOK, "email-popup.tmpl.html", "email-popup", emailPopupData{
			Link: link, Subject: subject, FromName: fromName, To: to, Error: errMsg,
		})
	}

	// validate required fields
	if link == "" || to == "" || subject == "" {
		renderError("please fill in all required fields")
		return
	}

	// validate field lengths
	if len(subject) > 200 {
		renderError("subject is too long (max 200 characters)")
		return
	}
	if len(fromName) > 100 {
		renderError("from name is too long (max 100 characters)")
		return
	}

	// validate the link points to this server
	if !s.isValidSecretLink(link) {
		renderError("invalid link")
		return
	}

	// validate email format
	if !email.IsValidEmail(to) {
		renderError("invalid email address")
		return
	}

	req := email.Request{To: to, Subject: subject, FromName: fromName, Link: link}
	if err := s.emailSender.Send(r.Context(), req); err != nil {
		log.Printf("[WARN] failed to send email: %v", err)
		// extract user-friendly error message
		errMsg := "failed to send email"
		errStr := strings.ToLower(err.Error())
		switch {
		case strings.Contains(errStr, "535") || strings.Contains(errStr, "501") || strings.Contains(errStr, "auth"):
			errMsg = "email authentication failed - check SMTP config"
		case strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host"):
			errMsg = "cannot connect to email server"
		case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline"):
			errMsg = "email server timeout - try again later"
		case strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate"):
			errMsg = "email server TLS/SSL error"
		}
		renderError(errMsg)
		return
	}

	log.Printf("[INFO] email sent successfully to %s", email.MaskEmail(to))

	// render success
	data := struct {
		To string
	}{
		To: to,
	}
	w.Header().Set("HX-Trigger-After-Settle", `{"closePopup": true}`)
	s.render(w, http.StatusOK, "email-sent.tmpl.html", "email-sent", data)
}

// isValidSecretLink validates that a link points to a message on this server
func (s Server) isValidSecretLink(link string) bool {
	parsed, err := url.Parse(link)
	if err != nil {
		return false
	}

	// check protocol matches
	if parsed.Scheme != s.cfg.Protocol {
		return false
	}

	// check host matches one of configured domains
	linkHost := parsed.Hostname()
	validHost := false
	for _, domain := range s.cfg.Domain {
		// extract hostname without port for comparison
		domainHost := domain
		if h, _, err := net.SplitHostPort(domain); err == nil {
			domainHost = h
		}
		if strings.EqualFold(linkHost, domainHost) {
			validHost = true
			break
		}
	}
	if !validHost {
		return false
	}

	// check for path traversal attempts (both raw and URL-encoded)
	if strings.Contains(parsed.Path, "..") || strings.Contains(parsed.RawPath, "..") {
		return false
	}

	// check path starts with /message/
	if !strings.HasPrefix(parsed.Path, "/message/") {
		return false
	}

	return true
}

// newTemplateCache creates a template cache as a map
func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := fs.Glob(assets.Files, "html/*/*.tmpl.html")
	if err != nil {
		return nil, fmt.Errorf("glob templates: %w", err)
	}

	for _, page := range pages {
		name := filepath.Base(page)

		patterns := []string{
			"html/index.tmpl.html",
			"html/partials/*.tmpl.html",
			page,
		}

		ts, err := template.New(name).Funcs(template.FuncMap{
			"until":      until,
			"add":        func(a, b int) int { return a + b },
			"jsonEscape": jsonEscape,
			"formatSize": formatSize,
			"urlquery":   url.QueryEscape,
		}).ParseFS(assets.Files, patterns...)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}
		cache[name] = ts
	}

	return cache, nil
}

// until is a helper function for templates to generate a slice of numbers
func until(n int) []int {
	result := make([]int, n)
	for i := range result {
		result[i] = i
	}
	return result
}

// formatSize formats file size in human-readable format
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// getValidatedHost returns the request host if it's in the allowed domains list, otherwise returns the first configured domain
func (s Server) getValidatedHost(r *http.Request) string {
	requestHost := r.Host

	// use net.SplitHostPort for proper IPv6 support
	host, port, err := net.SplitHostPort(requestHost)
	if err != nil {
		// no port present, use the whole host
		host = requestHost
		port = ""
	}

	// check if the host is in allowed domains (case-insensitive per RFC)
	for _, domain := range s.cfg.Domain {
		if strings.EqualFold(domain, host) {
			// protocol-aware port stripping
			if port != "" {
				if (s.cfg.Protocol == "http" && port == "80") ||
					(s.cfg.Protocol == "https" && port == "443") {
					return host // strip standard port
				}
				return requestHost // keep non-standard port
			}
			return host
		}
	}

	// host not in allowed domains, return the first configured domain as fallback
	// server.New() validates at least one domain is configured
	return s.cfg.Domain[0]
}

// jsonEscape safely escapes a string for use in JSON-LD script tags
func jsonEscape(s string) template.JS {
	// use Go's json.Marshal to properly escape the string
	b, err := json.Marshal(s)
	if err != nil {
		// this should never happen for string input, but return empty if it does
		return template.JS("")
	}
	// check length to avoid panic on empty strings
	if len(b) < 2 {
		return template.JS("")
	}
	// remove the surrounding quotes that Marshal adds
	//nolint:gosec // json.Marshal properly escapes content, template.JS prevents double escaping in JSON-LD
	return template.JS(b[1 : len(b)-1])
}
