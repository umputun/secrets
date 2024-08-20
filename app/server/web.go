package server

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

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
	Form    any
	PinSize int
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
	data := templateData{
		Form: createMsgForm{
			Exp:    15,
			MaxExp: humanDuration(s.cfg.MaxExpire),
		},
		PinSize: s.cfg.PinSize,
	}

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
		data := templateData{
			Form:    form,
			PinSize: s.cfg.PinSize,
		}

		// attach event listeners to pin inputs
		w.Header().Add("HX-Trigger-After-Swap", "setUpPinInputListeners")

		s.render(w, http.StatusOK, "home.tmpl.html", mainTmpl, data)
		return
	}

	msg, err := s.messager.MakeMessage(expDuration, form.Message, strings.Join(pinValues, ""))
	if err != nil {
		s.render(w, http.StatusOK, "secure-link.tmpl.html", errorTmpl, err.Error())
		return
	}

	msgURL := fmt.Sprintf("http://%s/message/%s", s.cfg.Domain, msg.Key)

	s.render(w, http.StatusOK, "secure-link.tmpl.html", "secure-link", msgURL)
}

// renders the show decoded message page
// GET /message/{key}
// URL Parameters:
//   - "key" (string): A path parameter representing the unique key of the message to be displayed.
func (s Server) showMessageViewCtrl(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, keyKey)

	data := templateData{
		Form: showMsgForm{
			Key: key,
		},
		PinSize: s.cfg.PinSize,
	}

	w.Header().Add("HX-Trigger-After-Swap", "setUpPinInputListeners")

	s.render(w, http.StatusOK, "show-message.tmpl.html", baseTmpl, data)
}

// renders the about page
// GET /about
func (s Server) aboutViewCtrl(w http.ResponseWriter, r *http.Request) { // nolint
	s.render(w, http.StatusOK, "about.tmpl.html", baseTmpl, nil)
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
		data := templateData{
			Form:    form,
			PinSize: s.cfg.PinSize,
		}

		// attach event listeners to pin inputs
		w.Header().Add("HX-Trigger-After-Swap", "setUpPinInputListeners")

		s.render(w, http.StatusOK, "show-message.tmpl.html", mainTmpl, data)
		return
	}

	msg, err := s.messager.LoadMessage(form.Key, strings.Join(pinValues, ""))
	if err != nil {
		if errors.Is(err, messager.ErrExpired) || errors.Is(err, store.ErrLoadRejected) {
			s.render(w, http.StatusOK, "error.tmpl.html", errorTmpl, err.Error())
			return
		}
		log.Printf("[WARN] can't load message %v", err)
		form.AddFieldError("pin", err.Error())

		data := templateData{
			Form:    form,
			PinSize: s.cfg.PinSize,
		}
		// attach event listeners to pin inputs
		w.Header().Add("HX-Trigger-After-Swap", "setUpPinInputListeners")

		s.render(w, http.StatusOK, "show-message.tmpl.html", mainTmpl, data)

		return
	}

	s.render(w, http.StatusOK, "decoded-message.tmpl.html", "decoded-message", string(msg.Data))
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
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d seconds", d/time.Second)
	case d < time.Hour:
		return fmt.Sprintf("%d minutes", d/time.Minute)
	case d < time.Hour*24:
		return fmt.Sprintf("%d hours", d/time.Hour)
	default:
		return fmt.Sprintf("%d days", d/(time.Hour*24))
	}
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

		ts, err := template.New(name).Funcs(template.FuncMap{"until": until}).ParseFS(ui.Files, patterns...)
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
