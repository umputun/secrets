// Package server provides rest-like api and serves static assets as well
package server

import (
	"context"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/pkg/errors"

	log "github.com/go-pkgz/lgr"
	um "github.com/go-pkgz/rest"

	"github.com/umputun/secrets/app/messager"
	"github.com/umputun/secrets/app/store"
)

// Config is a configuration for the server
type Config struct {
	Domain  string
	WebRoot string
	// Validation parameters
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
		return Server{}, errors.Wrap(err, "can't create template cache")
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

// Run the lister and request's router, activate rest server
func (s Server) Run(ctx context.Context) error {
	log.Printf("[INFO] activate rest server")

	httpServer := &http.Server{
		Addr:              ":8080",
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
		return errors.Wrap(err, "server failed")
	}
	return nil
}

func (s Server) routes() chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID, middleware.RealIP, um.Recoverer(log.Default()))
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(um.AppInfo("secrets", "Umputun", s.version), um.Ping, um.SizeLimit(64*1024))
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(Logger(log.Default()))
		r.Post("/message", s.saveMessageCtrl)
		r.Get("/message/{key}/{pin}", s.getMessageCtrl)
		r.Get("/params", s.getParamsCtrl)
	})

	router.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, "User-agent: *\nDisallow: /api/\nDisallow: /show/\n")
	})

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1") {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, JSON{"error": "not found"})
			return
		}

		s.render(w, http.StatusNotFound, "404.tmpl.html", baseTmpl, "not found")
	})

	router.Group(func(r chi.Router) {
		r.Use(Logger(log.Default()))
		r.Use(middleware.StripSlashes)
		r.Post("/generate-link", s.generateLinkCtrl)
		r.Get("/message/{key}", s.showMessageViewCtrl)
		r.Post("/load-message", s.loadMessageCtrl)
		r.Get("/about", s.aboutViewCtrl)
		r.Get("/", s.indexCtrl)
	})

	fs, err := um.NewFileServer("/static", s.cfg.WebRoot)
	if err != nil {
		log.Fatalf("[ERROR] can't create file server %v", err)
	}

	router.Handle("/static/*", fs)

	return router
}

func (s Server) saveMessageCtrl(w http.ResponseWriter, r *http.Request) {
	request := struct {
		Message string
		Exp     int
		Pin     string
	}{}

	if err := render.DecodeJSON(r.Body, &request); err != nil {
		log.Printf("[WARN] can't bind request %v", request)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, JSON{"error": err.Error()})
		return
	}

	if len(request.Pin) != s.cfg.PinSize {
		log.Printf("[WARN] incorrect pin size %d", len(request.Pin))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, JSON{"error": "Incorrect pin size"})
		return
	}

	msg, err := s.messager.MakeMessage(time.Second*time.Duration(request.Exp), request.Message, request.Pin)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, JSON{"error": err.Error()})
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, JSON{"key": msg.Key, "exp": msg.Exp})
}

// GET /v1/message/{key}/{pin}
func (s Server) getMessageCtrl(w http.ResponseWriter, r *http.Request) {

	key, pin := chi.URLParam(r, "key"), chi.URLParam(r, "pin")
	if key == "" || pin == "" || len(pin) != s.cfg.PinSize {
		log.Print("[WARN] no valid key or pin in get request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, JSON{"error": "no key or pin passed"})
		return
	}

	serveRequest := func() (status int, res JSON) {
		msg, err := s.messager.LoadMessage(key, pin)
		if err != nil {
			log.Printf("[WARN] failed to load key %v", key)
			if err == messager.ErrBadPinAttempt {
				return http.StatusExpectationFailed, JSON{"error": err.Error()}
			}
			return http.StatusBadRequest, JSON{"error": err.Error()}
		}
		return http.StatusOK, JSON{"key": msg.Key, "message": string(msg.Data)}
	}

	// make sure serveRequest works constant time on any branch
	st := time.Now()
	status, res := serveRequest()
	time.Sleep(time.Millisecond*100 - time.Since(st))
	render.Status(r, status)
	render.JSON(w, r, res)
}

// GET /params
func (s Server) getParamsCtrl(w http.ResponseWriter, r *http.Request) {
	params := struct {
		PinSize        int `json:"pin_size"`
		MaxPinAttempts int `json:"max_pin_attempts"`
		MaxExpSecs     int `json:"max_exp_sec"`
	}{
		PinSize:        s.cfg.PinSize,
		MaxPinAttempts: s.cfg.MaxPinAttempts,
		MaxExpSecs:     int(s.cfg.MaxExpire.Seconds()),
	}
	render.JSON(w, r, params)
}
