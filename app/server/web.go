package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
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

	"github.com/umputun/secrets/app/email"
	"github.com/umputun/secrets/app/messager"
	"github.com/umputun/secrets/app/server/validator"
	"github.com/umputun/secrets/app/store"
	"github.com/umputun/secrets/ui"
)

const (
	baseTmpl  = "base"
	mainTmpl  = "main"
	errorTmpl = "error"

	msgKey     = "message"
	pinKey     = "pin"
	expKey     = "exp"
	expUnitKey = "expUnit"
	keyKey     = "key"
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
	Preview  string
	Error    string
	To       string // preserved on validation error
	CC       string // preserved on validation error
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
func (s Server) indexCtrl(w http.ResponseWriter, r *http.Request) { // nolint
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
		s.render(w, http.StatusUnauthorized, "login-popup.tmpl.html", "login-popup", struct {
			Error string
			Theme string
		}{Error: "", Theme: getTheme(r)})
		return
	}

	contentType := r.Header.Get("Content-Type")
	isMultipart := strings.HasPrefix(contentType, "multipart/form-data")

	// handle file upload if multipart and files enabled
	if isMultipart && s.cfg.EnableFiles {
		s.generateFileLinkCtrl(w, r)
		return
	}

	// reject multipart when files disabled
	if isMultipart && !s.cfg.EnableFiles {
		log.Printf("[WARN] file upload rejected, files disabled")
		s.render(w, http.StatusBadRequest, "error.tmpl.html", errorTmpl, "file uploads disabled")
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
	for _, p := range pinValues {
		if validator.Blank(p) || !validator.IsNumber(p) {
			form.AddFieldError(pinKey, fmt.Sprintf("Pin must be %d digits long without empty values", s.cfg.PinSize))
			break
		}
	}

	form.CheckField(validator.NotBlank(form.Message), msgKey, "Message can't be empty")

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

	form.CheckField(validator.MaxDuration(expDuration, s.cfg.MaxExpire), expKey, fmt.Sprintf("Expire must be less than %s", humanDuration(s.cfg.MaxExpire)))

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

	msg, err := s.messager.MakeMessage(expDuration, form.Message, strings.Join(pinValues, ""))
	if err != nil {
		s.render(w, http.StatusOK, "secure-link.tmpl.html", errorTmpl, err.Error())
		return
	}

	s.renderSecureLink(w, r, msg.Key, form)
}

// generateFileLinkCtrl handles file upload requests
func (s Server) generateFileLinkCtrl(w http.ResponseWriter, r *http.Request) {
	// note: request body size already limited by rest.SizeLimit middleware in routes()
	err := r.ParseMultipartForm(s.cfg.MaxFileSize)
	if err != nil {
		log.Printf("[WARN] failed to parse multipart form: %v", err)
		s.render(w, http.StatusOK, "error.tmpl.html", errorTmpl, "file too large or invalid form")
		return
	}

	form := createMsgForm{
		ExpUnit: r.PostForm.Get(expUnitKey),
		MaxExp:  humanDuration(s.cfg.MaxExpire),
		IsFile:  true,
	}

	// validate PIN
	pinValues := r.Form["pin"]
	for _, p := range pinValues {
		if validator.Blank(p) || !validator.IsNumber(p) {
			form.AddFieldError(pinKey, fmt.Sprintf("Pin must be %d digits long without empty values", s.cfg.PinSize))
			break
		}
	}

	// validate expiration
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

	form.CheckField(validator.MaxDuration(expDuration, s.cfg.MaxExpire), expKey, fmt.Sprintf("Expire must be less than %s", humanDuration(s.cfg.MaxExpire)))

	// get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		form.AddFieldError("file", "Please select a file to upload")
	} else {
		defer file.Close()
		form.FileName = header.Filename
		form.FileSize = header.Size
	}

	if !form.Valid() {
		data := s.newTemplateData(r, form)
		if r.Header.Get("HX-Request") == "true" {
			s.render(w, http.StatusBadRequest, "home.tmpl.html", mainTmpl, data)
		} else {
			s.render(w, http.StatusOK, "home.tmpl.html", mainTmpl, data)
		}
		return
	}

	// read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Printf("[ERROR] failed to read uploaded file: %v", err)
		s.render(w, http.StatusOK, "error.tmpl.html", errorTmpl, "failed to read file")
		return
	}

	// detect content type from header or fallback
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// create file message
	msg, err := s.messager.MakeFileMessage(messager.FileRequest{
		Duration:    expDuration,
		Pin:         strings.Join(pinValues, ""),
		FileName:    header.Filename,
		ContentType: contentType,
		Data:        fileData,
	})
	if err != nil {
		log.Printf("[WARN] failed to create file message: %v", err)
		s.render(w, http.StatusOK, "secure-link.tmpl.html", errorTmpl, err.Error())
		return
	}

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
	key := r.PathValue(keyKey)

	// set X-Robots-Tag header for defense-in-depth beyond HTML meta
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")

	data := s.newTemplateData(r, showMsgForm{
		Key: key,
	})
	data.IsMessagePage = true            // prevent indexing of sensitive message pages
	data.IsFile = s.messager.IsFile(key) // check if message is a file for button label

	s.render(w, http.StatusOK, "show-message.tmpl.html", baseTmpl, data)
}

// renders the about page
// GET /about
func (s Server) aboutViewCtrl(w http.ResponseWriter, r *http.Request) { // nolint
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

	pinValues := r.Form["pin"]
	for _, p := range pinValues {
		if validator.Blank(p) || !validator.IsNumber(p) {
			form.AddFieldError(pinKey, fmt.Sprintf("Pin must be %d digits long without empty values", s.cfg.PinSize))
			break
		}
	}

	if !form.Valid() {
		data := s.newTemplateData(r, form)

		s.render(w, http.StatusOK, "show-message.tmpl.html", mainTmpl, data)
		return
	}

	// check if message is file BEFORE loading (LoadMessage may delete on max attempts)
	isFile := s.messager.IsFile(form.Key)

	msg, err := s.messager.LoadMessage(form.Key, strings.Join(pinValues, ""))
	if err != nil {
		s.handleLoadMessageError(w, r, &form, err, isFile)
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
		log.Printf("[INFO] accessed file message %s (%s), status 200 (success)", form.Key, filename)
		return
	}

	// text message - render decoded message template
	s.render(w, http.StatusOK, "decoded-message.tmpl.html", "decoded-message", string(msg.Data))
	log.Printf("[INFO] accessed message %s, status 200 (success)", form.Key)
}

// handleLoadMessageError handles errors from LoadMessage, rendering appropriate responses.
// isFile indicates whether the message was a file (checked before LoadMessage which may delete it).
func (s Server) handleLoadMessageError(w http.ResponseWriter, r *http.Request, form *showMsgForm, err error, isFile bool) {
	isHTMX := r.Header.Get("HX-Request") == "true"

	if errors.Is(err, messager.ErrExpired) || errors.Is(err, store.ErrLoadRejected) {
		// message not found or expired - return 404
		status := http.StatusNotFound
		if !isHTMX {
			status = http.StatusOK
		}
		s.render(w, status, "error.tmpl.html", errorTmpl, err.Error())
		log.Printf("[INFO] accessed message %s, status 404 (not found)", form.Key)
		return
	}

	// wrong PIN - add error to form
	form.AddFieldError("pin", err.Error())
	data := s.newTemplateData(r, form)
	data.IsFile = isFile // use pre-checked value (message may be deleted after max attempts)
	status := http.StatusOK
	tmpl := baseTmpl // full page for non-HTMX (file download form uses hx-boost="false")
	if isHTMX {
		status = http.StatusForbidden
		tmpl = mainTmpl // partial for HTMX swap
	}
	s.render(w, status, "show-message.tmpl.html", tmpl, data)
	log.Printf("[INFO] accessed message %s, status 403 (wrong pin)", form.Key)
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

	// toggle between explicit light/dark only (auto -> light -> dark -> light)
	var nextTheme string
	switch currentTheme {
	case "light":
		nextTheme = "dark"
	default: // "dark" or "auto"
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
		s.render(w, http.StatusOK, "popup-closed", "popup-closed", nil)
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

	// render email body preview
	preview, err := s.emailSender.RenderBody(link, defaultFromName)
	if err != nil {
		log.Printf("[WARN] failed to render email preview: %v", err)
		preview = "preview not available"
	}
	// strip HTML for preview display (simple approach - show as text summary)
	preview = extractEmailPreviewText(preview)

	s.render(w, http.StatusOK, "email-popup.tmpl.html", "email-popup", emailPopupData{
		Link:     link,
		Subject:  "Someone shared a secret with you",
		FromName: defaultFromName,
		Preview:  preview,
	})
}

// emailPreviewCtrl renders just the email preview for HTMX updates
// POST /email-preview
func (s Server) emailPreviewCtrl(w http.ResponseWriter, r *http.Request) {
	if s.emailSender == nil {
		http.Error(w, "email not configured", http.StatusServiceUnavailable)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	link := r.FormValue("link")
	fromName := r.FormValue("from_name")
	if fromName == "" {
		fromName = s.emailSender.GetDefaultFromName()
	}

	// validate the link points to this server to prevent misuse
	if link != "" && !s.isValidSecretLink(link) {
		http.Error(w, "invalid link", http.StatusBadRequest)
		return
	}

	preview, err := s.emailSender.RenderBody(link, fromName)
	if err != nil {
		log.Printf("[WARN] failed to render email preview: %v", err)
		preview = "preview not available"
	}
	preview = extractEmailPreviewText(preview)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, preview)
}

// sendEmailCtrl sends the email with the secret link
// POST /send-email
func (s Server) sendEmailCtrl(w http.ResponseWriter, r *http.Request) {
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
	cc := strings.TrimSpace(r.FormValue("cc"))

	// validate required fields
	if link == "" || to == "" || subject == "" {
		s.render(w, http.StatusBadRequest, "email-popup.tmpl.html", "email-popup", emailPopupData{
			Link: link, Subject: subject, FromName: fromName, To: to, CC: cc, Error: "please fill in all required fields",
		})
		return
	}

	// validate field lengths to prevent email header issues
	if len(subject) > 200 {
		s.render(w, http.StatusBadRequest, "email-popup.tmpl.html", "email-popup", emailPopupData{
			Link: link, Subject: subject, FromName: fromName, To: to, CC: cc, Error: "subject is too long (max 200 characters)",
		})
		return
	}
	if len(fromName) > 100 {
		s.render(w, http.StatusBadRequest, "email-popup.tmpl.html", "email-popup", emailPopupData{
			Link: link, Subject: subject, FromName: fromName, To: to, CC: cc, Error: "from name is too long (max 100 characters)",
		})
		return
	}

	// validate the link points to this server to prevent phishing relay
	if !s.isValidSecretLink(link) {
		s.render(w, http.StatusBadRequest, "email-popup.tmpl.html", "email-popup", emailPopupData{
			Link: link, Subject: subject, FromName: fromName, To: to, CC: cc, Error: "invalid link",
		})
		return
	}

	// validate email format
	if !email.IsValidEmail(to) {
		s.render(w, http.StatusBadRequest, "email-popup.tmpl.html", "email-popup", emailPopupData{
			Link: link, Subject: subject, FromName: fromName, To: to, CC: cc, Error: "invalid email address",
		})
		return
	}

	// build recipients list
	toList := []string{to}
	var ccList []string
	if cc != "" {
		if !email.IsValidEmail(cc) {
			s.render(w, http.StatusBadRequest, "email-popup.tmpl.html", "email-popup", emailPopupData{
				Link: link, Subject: subject, FromName: fromName, To: to, CC: cc, Error: "invalid CC email address",
			})
			return
		}
		ccList = []string{cc}
	}

	// send email
	req := email.Request{
		To:       toList,
		CC:       ccList,
		Subject:  subject,
		FromName: fromName,
		Link:     link,
	}

	ctx := r.Context()
	if err := s.emailSender.Send(ctx, req); err != nil {
		log.Printf("[ERROR] failed to send email: %v", err)
		s.render(w, http.StatusInternalServerError, "email-popup.tmpl.html", "email-popup", emailPopupData{
			Link: link, Subject: subject, FromName: fromName, To: to, CC: cc, Error: "failed to send email, please try again",
		})
		return
	}

	log.Printf("[INFO] email sent to %s", to)

	// render success
	data := struct {
		To string
	}{
		To: to,
	}
	w.Header().Set("HX-Trigger-After-Settle", `{"closePopup": true}`)
	s.render(w, http.StatusOK, "email-sent.tmpl.html", "email-sent", data)
}

// extractEmailPreviewText extracts a simple text preview from HTML email body
func extractEmailPreviewText(htmlBody string) string {
	// simple extraction - look for the key message parts
	// in practice this strips most HTML and keeps the essence
	preview := htmlBody

	// remove style and script tags
	preview = removeHTMLTags(preview)

	// trim and limit length (use runes to avoid cutting multi-byte UTF-8 characters)
	preview = strings.TrimSpace(preview)
	runes := []rune(preview)
	if len(runes) > 300 {
		preview = string(runes[:300]) + "..."
	}

	return preview
}

// removeHTMLTags is a simple HTML tag stripper
func removeHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			result.WriteRune(' ')
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	// collapse multiple spaces
	return strings.Join(strings.Fields(result.String()), " ")
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

	// check path starts with /message/
	if !strings.HasPrefix(parsed.Path, "/message/") {
		return false
	}

	return true
}

// newTemplateCache creates a template cache as a map
func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := fs.Glob(ui.Files, "html/*/*.tmpl.html")

	if err != nil {
		return nil, err
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
		}).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
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
	if len(s.cfg.Domain) > 0 {
		return s.cfg.Domain[0]
	}

	// should not happen with required validation - fail loudly if reached
	panic("no domains configured: validation should occur in server.New() at startup")
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
	// nolint:gosec // json.Marshal properly escapes content, template.JS prevents double escaping in JSON-LD
	return template.JS(b[1 : len(b)-1])
}
