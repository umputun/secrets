// Package server provides rest-like api and serves static assets as well
package server

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/umputun/secrets/backend/app/store"

	log "github.com/go-pkgz/lgr"
	um "github.com/go-pkgz/rest"

	"github.com/umputun/secrets/backend/app/messager"
)

// Server is a rest with store
type Server struct {
	Messager       Messager
	PinSize        int
	MaxPinAttempts int
	MaxExpire      time.Duration
	Version        string
}

// Messager interface making and loading messages
type Messager interface {
	MakeMessage(duration time.Duration, msg, pin string) (result *store.Message, err error)
	LoadMessage(key, pin string) (msg *store.Message, err error)
}

// Run the lister and request's router, activate rest server
func (s Server) Run() {
	log.Printf("[INFO] activate rest server")

	router := chi.NewRouter()

	router.Use(middleware.RequestID, middleware.RealIP, um.Recoverer(log.Default()))
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(um.AppInfo("secrets", "Umputun", s.Version), um.Ping)
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(10, nil)))
	router.Use(Rewrite("/show/(.*)", "/show/?$1"))

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(Logger())
		r.Post("/message", s.saveMessageCtrl)
		r.Get("/message/{key}/{pin}", s.getMessageCtrl)
		r.Get("/params", s.getParamsCtrl)
	})

	router.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, "User-agent: *\nDisallow: /api/\nDisallow: /show/\n")
	})

	s.fileServer(router, "/", http.Dir(filepath.Join(".", "docroot")))

	log.Fatalf("server failed, %v", http.ListenAndServe(":8080", router))
}

// POST /v1/message
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

	if len(request.Pin) != s.PinSize {
		log.Printf("[WARN] incorrect pin size %d", len(request.Pin))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, JSON{"error": "Incorrect pin size"})
		return
	}

	msg, err := s.Messager.MakeMessage(time.Second*time.Duration(request.Exp), request.Message, request.Pin)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, JSON{"error": err.Error()})
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, JSON{"key": msg.Key, "exp": msg.Exp})
}

// GET /v1/message/:key/:pin
func (s Server) getMessageCtrl(w http.ResponseWriter, r *http.Request) {

	key, pin := chi.URLParam(r, "key"), chi.URLParam(r, "pin")
	if key == "" || pin == "" || len(pin) != s.PinSize {
		log.Print("[WARN] no valid key or pin in get request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, JSON{"error": "no key or pin passed"})
		return
	}

	serveRequest := func() (status int, res JSON) {
		msg, err := s.Messager.LoadMessage(key, pin)
		if err != nil {
			log.Printf("[WARN] failed to load key %v", key)
			if err == messager.ErrBadPinAttempt {
				return http.StatusExpectationFailed, JSON{"error": err.Error()}
			}
			return http.StatusBadRequest, JSON{"error": err.Error()}
		}
		return http.StatusOK, JSON{"key": msg.Key, "message": msg.Data}
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
		PinSize:        s.PinSize,
		MaxPinAttempts: s.MaxPinAttempts,
		MaxExpSecs:     int(s.MaxExpire.Seconds()),
	}
	render.JSON(w, r, params)
}

// serves static files from ./docroot
func (s Server) fileServer(r chi.Router, path string, root http.FileSystem) {
	log.Printf("[INFO] run file server for %s", root)
	fs := http.StripPrefix(path, http.FileServer(root))
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		// don't show dirs, just serve files
		if strings.HasSuffix(r.URL.Path, "/") && len(r.URL.Path) > 1 && r.URL.Path != "/show/" {
			http.NotFound(w, r)
			return
		}
		fs.ServeHTTP(w, r)
	})
}
