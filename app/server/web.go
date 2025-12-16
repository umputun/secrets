package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"mime"
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
	Message string
	Exp     int
	MaxExp  string
	ExpUnit string
	validator.Validator
}

type showMsgForm struct {
	Key     string
	Message string
	validator.Validator
}

type templateData struct {
	Form          any
	PinSize       int
	CurrentYear   int
	Theme         string
	Branding      string
	URL           string // canonical URL for the page
	BaseURL       string // base URL for the site (protocol://domain)
	PageTitle     string // SEO-optimized page title
	PageDesc      string // page description for meta tags
	IsMessagePage bool   // true for message pages (should not be indexed)
	FilesEnabled  bool   // true when file upload is enabled
	MaxFileSize   int64  // max file size in bytes
}

type createFileForm struct {
	FileName string
	FileSize int64
	Exp      int
	MaxExp   string
	ExpUnit  string
	validator.Validator
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

// validatePinValues checks if all PIN values are valid (non-blank numeric).
// returns the concatenated PIN and any validation error message.
func (s Server) validatePinValues(pinValues []string) (pin, errMsg string) {
	for _, p := range pinValues {
		if validator.Blank(p) || !validator.IsNumber(p) {
			return "", fmt.Sprintf("Pin must be %d digits long without empty values", s.cfg.PinSize)
		}
	}
	return strings.Join(pinValues, ""), ""
}

// renders the home page
// GET /
func (s Server) indexCtrl(w http.ResponseWriter, r *http.Request) { // nolint
	data := s.newTemplateData(r, createMsgForm{
		Exp:    15,
		MaxExp: humanDuration(s.cfg.MaxExpire),
	})
	data.PageTitle = "Secure Password Sharing - Self-Destructing Messages"
	data.PageDesc = "Share sensitive information securely with self-destructing messages protected by PIN codes. Free, open-source, and privacy-focused password sharing."

	s.render(w, http.StatusOK, "home.tmpl.html", baseTmpl, data)
}

// renders the generate link page
// POST /generate-link
// Request Body: This function expects a POST request body containing the following fields:
//   - "message" (string): The message content to be associated with the secure link.
//   - "expUnit" (string): The unit of expiration time (e.g., "m" for minutes, "h" for hours, "d" for days).
//   - "pin" (slice of strings): An array of PIN values.
func (s Server) generateLinkCtrl(w http.ResponseWriter, r *http.Request) {
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
	pin, pinErr := s.validatePinValues(pinValues)
	if pinErr != "" {
		form.AddFieldError(pinKey, pinErr)
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

	msg, err := s.messager.MakeMessage(expDuration, form.Message, pin)
	if err != nil {
		s.render(w, http.StatusOK, "secure-link.tmpl.html", errorTmpl, err.Error())
		return
	}

	msgURL := (&url.URL{
		Scheme: s.cfg.Protocol,
		Host:   bracketIPv6Host(s.getValidatedHost(r)),
		Path:   path.Join("/message", msg.Key),
	}).String()

	s.render(w, http.StatusOK, "secure-link.tmpl.html", "secure-link", msgURL)
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
	data.IsMessagePage = true // prevent indexing of sensitive message pages

	s.render(w, http.StatusOK, "show-message.tmpl.html", baseTmpl, data)
}

// renders the about page
// GET /about
func (s Server) aboutViewCtrl(w http.ResponseWriter, r *http.Request) { // nolint
	data := s.newTemplateData(r, nil)
	data.PageTitle = "How It Works - Encrypted Message Sharing"
	data.PageDesc = "Learn how SafeSecret protects your sensitive information with PIN-protected encryption, self-destructing messages, and zero-knowledge architecture."
	s.render(w, http.StatusOK, "about.tmpl.html", baseTmpl, data)
}

// renders the decoded message page
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
	pin, pinErr := s.validatePinValues(pinValues)
	if pinErr != "" {
		form.AddFieldError(pinKey, pinErr)
	}

	if !form.Valid() {
		data := s.newTemplateData(r, form)

		s.render(w, http.StatusOK, "show-message.tmpl.html", mainTmpl, data)
		return
	}

	// constant-time wrapper to prevent timing attacks
	serveRequest := func() (*store.Message, error) {
		return s.messager.LoadMessage(form.Key, pin)
	}

	st := time.Now()
	msg, err := serveRequest()
	time.Sleep(time.Millisecond*100 - time.Since(st))

	if err != nil {
		s.handleLoadMessageError(w, r, &form, err)
		return
	}

	// handle file messages - render download template for HTMX, direct download otherwise
	if msg.IsFile {
		if r.Header.Get("HX-Request") == "true" {
			// for HTMX requests, render template with base64 data for JS download
			data := struct {
				FileName    string
				FileSize    string
				ContentType string
				DataBase64  string
			}{
				FileName:    msg.FileName,
				FileSize:    formatFileSize(int64(len(msg.Data))),
				ContentType: msg.ContentType,
				DataBase64:  base64.StdEncoding.EncodeToString(msg.Data),
			}
			s.render(w, http.StatusOK, "decoded-file.tmpl.html", "decoded-file", data)
			log.Printf("[INFO] served file %s via load-message (htmx), name=%s, size=%d", form.Key, msg.FileName, len(msg.Data))
			return
		}
		s.serveFileDownload(w, msg)
		log.Printf("[INFO] served file %s via load-message, name=%s, size=%d", form.Key, msg.FileName, len(msg.Data))
		return
	}

	s.render(w, http.StatusOK, "decoded-message.tmpl.html", "decoded-message", string(msg.Data))
	log.Printf("[INFO] accessed message %s, status 200 (success)", form.Key)
}

// handleLoadMessageError handles errors from LoadMessage, rendering appropriate responses
func (s Server) handleLoadMessageError(w http.ResponseWriter, r *http.Request, form *showMsgForm, err error) {
	isHTMX := r.Header.Get("HX-Request") == "true"

	// for non-HTMX requests, render full page with base template
	tmpl := baseTmpl
	if isHTMX {
		tmpl = mainTmpl
	}

	if errors.Is(err, messager.ErrExpired) || errors.Is(err, store.ErrLoadRejected) {
		// message not found or expired - show error on form
		form.AddFieldError("pin", "message expired or deleted")
		data := s.newTemplateData(r, form)
		status := http.StatusOK
		if isHTMX {
			status = http.StatusNotFound
		}
		s.render(w, status, "show-message.tmpl.html", tmpl, data)
		log.Printf("[INFO] accessed message %s, status 404 (not found)", form.Key)
		return
	}

	// wrong PIN - add error to form
	form.AddFieldError("pin", err.Error())
	data := s.newTemplateData(r, form)
	status := http.StatusOK
	if isHTMX {
		status = http.StatusForbidden
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

	// render popup with message and auto-close after 2 seconds
	s.render(w, http.StatusOK, "popup.tmpl.html", "popup", data)

	// trigger auto-close after 2 seconds using HX-Trigger header
	w.Header().Set("HX-Trigger-After-Settle", `{"closePopup": "2s"}`)
}

// closePopupCtrl closes the popup
// GET /close-popup
func (s Server) closePopupCtrl(w http.ResponseWriter, _ *http.Request) {
	s.render(w, http.StatusOK, "popup.tmpl.html", "popup-closed", nil)
}

// textFormPartialCtrl returns the text input partial for form switching
// GET /form/text
func (s Server) textFormPartialCtrl(w http.ResponseWriter, r *http.Request) {
	data := s.newTemplateData(r, createMsgForm{
		Exp:    15,
		MaxExp: humanDuration(s.cfg.MaxExpire),
	})
	s.render(w, http.StatusOK, "text-input.tmpl.html", "text-input", data)
}

// fileFormPartialCtrl returns the file input partial for form switching
// GET /form/file
func (s Server) fileFormPartialCtrl(w http.ResponseWriter, r *http.Request) {
	data := s.newTemplateData(r, createFileForm{
		Exp:    15,
		MaxExp: humanDuration(s.cfg.MaxExpire),
	})
	s.render(w, http.StatusOK, "file-input.tmpl.html", "file-input", data)
}

// generateFileLinkCtrl handles file upload and generates secure link
// POST /generate-file-link
func (s Server) generateFileLinkCtrl(w http.ResponseWriter, r *http.Request) {
	// size limit is enforced by sizeLimitMiddleware
	if err := r.ParseMultipartForm(s.cfg.MaxFileSize); err != nil {
		log.Printf("[WARN] can't parse multipart form: %v", err)
		form := createFileForm{MaxExp: humanDuration(s.cfg.MaxExpire)}
		form.AddFieldError("file", "file too large or invalid form")
		data := s.newTemplateData(r, form)
		if r.Header.Get("HX-Request") == "true" {
			s.render(w, http.StatusBadRequest, "file-input.tmpl.html", "file-input", data)
		} else {
			s.render(w, http.StatusOK, "file-input.tmpl.html", "file-input", data)
		}
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[WARN] no file in request: %v", err)
		form := createFileForm{MaxExp: humanDuration(s.cfg.MaxExpire)}
		form.AddFieldError("file", "file is required")
		data := s.newTemplateData(r, form)
		if r.Header.Get("HX-Request") == "true" {
			s.render(w, http.StatusBadRequest, "file-input.tmpl.html", "file-input", data)
		} else {
			s.render(w, http.StatusOK, "file-input.tmpl.html", "file-input", data)
		}
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Printf("[ERROR] can't read file: %v", err)
		s.render(w, http.StatusOK, "error.tmpl.html", errorTmpl, "failed to read file")
		return
	}

	form := createFileForm{
		FileName: header.Filename,
		FileSize: int64(len(fileData)),
		ExpUnit:  r.PostFormValue(expUnitKey),
		MaxExp:   humanDuration(s.cfg.MaxExpire),
	}

	pinValues := r.Form["pin"]
	pin, pinErr := s.validatePinValues(pinValues)
	if pinErr != "" {
		form.AddFieldError(pinKey, pinErr)
	}

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
		if r.Header.Get("HX-Request") == "true" {
			s.render(w, http.StatusBadRequest, "file-input.tmpl.html", "file-input", data)
		} else {
			s.render(w, http.StatusOK, "file-input.tmpl.html", "file-input", data)
		}
		return
	}

	contentType := sanitizeContentType(header.Header.Get("Content-Type"))

	msg, err := s.messager.MakeFileMessage(messager.FileRequest{
		Duration:    expDuration,
		Data:        fileData,
		FileName:    header.Filename,
		ContentType: contentType,
		Pin:         pin,
	})
	if err != nil {
		s.render(w, http.StatusOK, "secure-link-file.tmpl.html", errorTmpl, err.Error())
		return
	}

	msgURL := (&url.URL{
		Scheme: s.cfg.Protocol,
		Host:   bracketIPv6Host(s.getValidatedHost(r)),
		Path:   path.Join("/message", msg.Key),
	}).String()

	// render with file info
	data := struct {
		URL      string
		FileName string
		FileSize string
	}{
		URL:      msgURL,
		FileName: header.Filename,
		FileSize: formatFileSize(int64(len(fileData))),
	}
	s.render(w, http.StatusOK, "secure-link-file.tmpl.html", "secure-link-file", data)
	log.Printf("[INFO] created file link %s, file=%s, size=%d", msg.Key, header.Filename, len(fileData))
}

// downloadFileCtrl handles file download from web UI
// POST /download-file
func (s Server) downloadFileCtrl(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.render(w, http.StatusOK, "error.tmpl.html", errorTmpl, err.Error())
		return
	}

	form := showMsgForm{
		Key: r.PostForm.Get("key"),
	}

	pinValues := r.Form["pin"]
	pin, pinErr := s.validatePinValues(pinValues)
	if pinErr != "" {
		form.AddFieldError(pinKey, pinErr)
	}

	if !form.Valid() {
		data := s.newTemplateData(r, form)
		s.render(w, http.StatusOK, "show-message.tmpl.html", baseTmpl, data)
		return
	}

	// constant-time wrapper to prevent timing attacks
	serveRequest := func() (*store.Message, error) {
		return s.messager.LoadMessage(form.Key, pin)
	}

	st := time.Now()
	msg, err := serveRequest()
	time.Sleep(time.Millisecond*100 - time.Since(st))

	if err != nil {
		s.handleLoadMessageError(w, r, &form, err)
		return
	}

	if !msg.IsFile {
		// not a file, render full page with decoded text
		data := s.newTemplateData(r, nil)
		data.Form = string(msg.Data) // message text for template
		s.render(w, http.StatusOK, "decoded-message.tmpl.html", baseTmpl, data)
		log.Printf("[INFO] accessed message %s, status 200 (success)", form.Key)
		return
	}

	// file message - render download page with file info and download button
	data := s.newTemplateData(r, nil)
	data.Form = struct {
		FileName    string
		FileSize    string
		ContentType string
		DataBase64  string
	}{
		FileName:    msg.FileName,
		FileSize:    formatFileSize(int64(len(msg.Data))),
		ContentType: msg.ContentType,
		DataBase64:  base64.StdEncoding.EncodeToString(msg.Data),
	}
	s.render(w, http.StatusOK, "decoded-file.tmpl.html", baseTmpl, data)
	log.Printf("[INFO] served file %s via download-file, name=%s, size=%d", form.Key, msg.FileName, len(msg.Data))
}

// serveFileDownload serves a file message as a download with security headers
func (s Server) serveFileDownload(w http.ResponseWriter, msg *store.Message) {
	contentType := msg.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// use mime.FormatMediaType for proper RFC 2231 encoding of filename
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": msg.FileName})
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(msg.Data)), 10))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(msg.Data); err != nil {
		log.Printf("[WARN] failed to write file data: %v", err)
	}
}

// sanitizeContentType validates and normalizes content type, returning a safe default if invalid.
// uses mime.ParseMediaType to properly parse and validate the content type.
func sanitizeContentType(contentType string) string {
	if contentType == "" {
		return "application/octet-stream"
	}

	// parse and validate content type
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType == "" {
		return "application/octet-stream"
	}

	return mediaType
}

// formatFileSize formats bytes to human readable string
func formatFileSize(size int64) string {
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
			"until":          until,
			"add":            func(a, b int) int { return a + b },
			"jsonEscape":     jsonEscape,
			"formatFileSize": formatFileSize,
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

// bracketIPv6Host ensures IPv6 addresses are properly bracketed for URL construction.
// handles both hosts with and without ports.
func bracketIPv6Host(host string) string {
	h, port, err := net.SplitHostPort(host)
	if err != nil {
		// no port present, check if the whole string is an IPv6 address
		if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
			return "[" + host + "]"
		}
		return host
	}
	// port present, check if host part is IPv6 and needs bracketing
	if ip := net.ParseIP(h); ip != nil && ip.To4() == nil && !strings.HasPrefix(h, "[") {
		return "[" + h + "]:" + port
	}
	return host
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
