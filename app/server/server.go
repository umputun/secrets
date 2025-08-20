// Package server provides rest-like api and serves static assets as well
package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/didip/tollbooth/v8"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/routegroup"

	"github.com/umputun/secrets/app/messager"
	"github.com/umputun/secrets/app/store"
	"github.com/umputun/secrets/ui"
)

// Config is a configuration for the server
type Config struct {
	Domain   string
	WebRoot  string
	Protocol string
	Branding string
	Port     string // server port, defaults to :8080
	// validation parameters
	PinSize        int
	MaxPinAttempts int
	MaxExpire      time.Duration
}

// Server is a rest with store
type Server struct {
	messager      Messager
	cfg           Config
	version       string
	templateCache map[string]*template.Template
}

// New creates a new server with template cache
func New(m Messager, version string, cfg Config) (Server, error) {
	cache, err := newTemplateCache()
	if err != nil {
		return Server{}, fmt.Errorf("can't create template cache: %w", err)
	}
	return Server{
		messager:      m,
		cfg:           cfg,
		version:       version,
		templateCache: cache,
	}, nil
}

// Messager interface making and loading messages
type Messager interface {
	MakeMessage(duration time.Duration, msg, pin string) (result *store.Message, err error)
	LoadMessage(key, pin string) (msg *store.Message, err error)
}

// newTemplateData creates a templateData with common fields populated
func (s Server) newTemplateData(r *http.Request, form any) templateData {
	return templateData{
		Form:        form,
		PinSize:     s.cfg.PinSize,
		CurrentYear: time.Now().Year(),
		Theme:       getTheme(r),
		Branding:    s.cfg.Branding,
	}
}

// Run the lister and request's router, activate rest server
func (s Server) Run(ctx context.Context) error {
	log.Printf("[INFO] activate rest server")

	port := s.cfg.Port
	if port == "" {
		port = ":8080"
	}

	httpServer := &http.Server{
		Addr:              port,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	go func() {
		<-ctx.Done()
		if httpServer != nil {
			if clsErr := httpServer.Close(); clsErr != nil {
				log.Printf("[ERROR] failed to close proxy http server, %v", clsErr)
			}
		}
	}()

	err := httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)

	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

func (s Server) routes() http.Handler {
	mux := http.NewServeMux()
	router := routegroup.New(mux)

	// global middleware - applied to all routes
	router.Use(
		rest.RealIP,
		rest.Recoverer(log.Default()),
		rest.Throttle(1000),
		Timeout(60*time.Second),
		rest.AppInfo("secrets", "Umputun", s.version),
		rest.Ping,
		rest.SizeLimit(64*1024),
		tollbooth.HTTPMiddleware(tollbooth.NewLimiter(10, nil)),
	)

	// API routes
	router.Mount("/api/v1").Route(func(apiGroup *routegroup.Bundle) {
		apiGroup.Use(Logger(log.Default()))
		apiGroup.Handle("POST /message", http.HandlerFunc(s.saveMessageCtrl))
		apiGroup.Handle("GET /message/{key}/{pin}", http.HandlerFunc(s.getMessageCtrl))
		apiGroup.Handle("GET /params", http.HandlerFunc(s.getParamsCtrl))
	})

	// web routes
	router.Group().Route(func(webGroup *routegroup.Bundle) {
		webGroup.Use(Logger(log.Default()), StripSlashes)
		webGroup.Handle("POST /generate-link", http.HandlerFunc(s.generateLinkCtrl))
		webGroup.Handle("GET /message/{key}", http.HandlerFunc(s.showMessageViewCtrl))
		webGroup.Handle("POST /load-message", http.HandlerFunc(s.loadMessageCtrl))
		webGroup.Handle("POST /theme", http.HandlerFunc(s.themeToggleCtrl))
		webGroup.Handle("POST /copy-feedback", http.HandlerFunc(s.copyFeedbackCtrl))
		webGroup.Handle("GET /close-popup", http.HandlerFunc(s.closePopupCtrl))
		webGroup.Handle("GET /about", http.HandlerFunc(s.aboutViewCtrl))
		webGroup.Handle("GET /{$}", http.HandlerFunc(s.indexCtrl)) // exact match for root only
	})

	// special routes without groups
	router.Handle("GET /robots.txt", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /api/\nDisallow: /show/\n"))
	}))

	// custom 404 handler
	router.NotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1") {
			rest.SendErrorJSON(w, r, log.Default(), http.StatusNotFound, errors.New("not found"), "endpoint not found")
			return
		}
		s.render(w, http.StatusNotFound, "404.tmpl.html", baseTmpl, s.newTemplateData(r, nil))
	})

	// static file handling
	if _, err := os.Stat(s.cfg.WebRoot); os.IsNotExist(err) || s.cfg.WebRoot == "" {
		// use embedded file system
		staticFS, err := fs.Sub(ui.Files, "static")
		if err != nil {
			log.Fatalf("[ERROR] can't create embedded file server %v", err)
		}
		router.HandleFiles("/static", http.FS(staticFS))
	} else {
		// use local file system
		router.HandleFiles("/static", http.Dir(s.cfg.WebRoot))
	}

	return mux
}

func (s Server) saveMessageCtrl(w http.ResponseWriter, r *http.Request) {
	request := struct {
		Message string
		Exp     int
		Pin     string
	}{}

	if err := rest.DecodeJSON(r, &request); err != nil {
		log.Printf("[WARN] can't bind request %v", request)
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "can't decode request")
		return
	}

	if len(request.Pin) != s.cfg.PinSize {
		log.Printf("[WARN] incorrect pin size %d", len(request.Pin))
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("incorrect pin size"), "incorrect pin size")
		return
	}

	msg, err := s.messager.MakeMessage(time.Second*time.Duration(request.Exp), request.Message, request.Pin)
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "can't create message")
		return
	}
	w.WriteHeader(http.StatusCreated)
	rest.RenderJSON(w, rest.JSON{"key": msg.Key, "exp": msg.Exp})
	log.Printf("[INFO] created message %s exp %s", msg.Key, msg.Exp.Format(time.RFC3339))
}

// GET /v1/message/{key}/{pin}
func (s Server) getMessageCtrl(w http.ResponseWriter, r *http.Request) {
	key, pin := r.PathValue("key"), r.PathValue("pin")
	if key == "" || pin == "" || len(pin) != s.cfg.PinSize {
		log.Print("[WARN] no valid key or pin in get request")
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("no key or pin passed"), "invalid request")
		return
	}

	serveRequest := func() (status int, res rest.JSON) {
		msg, err := s.messager.LoadMessage(key, pin)
		if err != nil {
			log.Printf("[WARN] failed to load key %v", key)
			if errors.Is(err, messager.ErrBadPinAttempt) {
				return http.StatusExpectationFailed, rest.JSON{"error": err.Error()}
			}
			return http.StatusBadRequest, rest.JSON{"error": err.Error()}
		}
		return http.StatusOK, rest.JSON{"key": msg.Key, "message": string(msg.Data)}
	}

	// make sure serveRequest works constant time on any branch
	st := time.Now()
	status, res := serveRequest()
	time.Sleep(time.Millisecond*100 - time.Since(st))
	w.WriteHeader(status)
	rest.RenderJSON(w, res)

	var statusText string
	switch status {
	case 200:
		statusText = "success"
	case 417:
		statusText = "wrong pin"
	default:
		statusText = "error"
	}
	log.Printf("[INFO] accessed message %s, status %d (%s)", key, status, statusText)
}

// GET /params
func (s Server) getParamsCtrl(w http.ResponseWriter, _ *http.Request) {
	params := struct {
		PinSize        int `json:"pin_size"`
		MaxPinAttempts int `json:"max_pin_attempts"`
		MaxExpSecs     int `json:"max_exp_sec"`
	}{
		PinSize:        s.cfg.PinSize,
		MaxPinAttempts: s.cfg.MaxPinAttempts,
		MaxExpSecs:     int(s.cfg.MaxExpire.Seconds()),
	}
	rest.RenderJSON(w, params)
}
