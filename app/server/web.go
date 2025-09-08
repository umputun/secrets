package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
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
	form.CheckField(validator.PermittedValue(form.ExpUnit, "m", "h", "d"), expUnitKey, "Only Minutes, Hours and Days are allowed")

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

	msgURL := fmt.Sprintf("%s://%s/message/%s", s.cfg.Protocol, s.cfg.Domain, msg.Key)

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

	msg, err := s.messager.LoadMessage(form.Key, strings.Join(pinValues, ""))
	if err != nil {
		if errors.Is(err, messager.ErrExpired) || errors.Is(err, store.ErrLoadRejected) {
			// message not found or expired - return 404
			status := http.StatusNotFound
			if r.Header.Get("HX-Request") == "true" {
				s.render(w, status, "error.tmpl.html", errorTmpl, err.Error())
			} else {
				s.render(w, http.StatusOK, "error.tmpl.html", errorTmpl, err.Error())
			}
			log.Printf("[INFO] accessed message %s, status 404 (not found)", form.Key)
			return
		}
		// wrong PIN - add error to form
		form.AddFieldError("pin", err.Error())

		data := s.newTemplateData(r, form)
		// for HTMX requests, return 403 for wrong PIN
		status := http.StatusOK
		if r.Header.Get("HX-Request") == "true" {
			status = http.StatusForbidden
		}
		s.render(w, status, "show-message.tmpl.html", mainTmpl, data)
		log.Printf("[INFO] accessed message %s, status 403 (wrong pin)", form.Key)
		return
	}

	s.render(w, http.StatusOK, "decoded-message.tmpl.html", "decoded-message", string(msg.Data))
	log.Printf("[INFO] accessed message %s, status 200 (success)", form.Key)
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
		return "auto" // default theme
	}
	// validate theme value
	switch cookie.Value {
	case "light", "dark", "auto":
		return cookie.Value
	default:
		return "auto"
	}
}

// themeToggleCtrl handles theme switching
// POST /theme
func (s Server) themeToggleCtrl(w http.ResponseWriter, r *http.Request) {
	currentTheme := getTheme(r)

	// cycle through themes: light -> dark -> auto -> light
	nextTheme := "light"
	switch currentTheme {
	case "light":
		nextTheme = "dark"
	case "dark":
		nextTheme = "auto"
	case "auto":
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

// jsonEscape safely escapes a string for use in JSON-LD script tags
func jsonEscape(s string) template.JS {
	// use Go's json.Marshal to properly escape the string
	b, err := json.Marshal(s)
	if err != nil {
		// this should never happen for string input, but return empty if it does
		return template.JS("")
	}
	// remove the surrounding quotes that Marshal adds
	// nolint:gosec // json.Marshal properly escapes content, template.JS prevents double escaping in JSON-LD
	return template.JS(b[1 : len(b)-1])
}
