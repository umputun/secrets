// Package server provides rest-like api and serves static assets as well
package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
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
	Domain   []string // allowed domains list
	WebRoot  string
	Protocol string
	Branding string
	Port     string // server port, defaults to :8080
	// validation parameters
	PinSize        int
	MaxPinAttempts int
	MaxExpire      time.Duration
	// file support
	EnableFiles bool
	MaxFileSize int64
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
	if len(cfg.Domain) == 0 {
		return Server{}, fmt.Errorf("at least one domain must be configured")
	}

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
	MakeFileMessage(req messager.FileRequest) (result *store.Message, err error)
	LoadMessage(key, pin string) (msg *store.Message, err error)
}

// newTemplateData creates a templateData with common fields populated
func (s Server) newTemplateData(r *http.Request, form any) templateData {
	// use the first configured domain for canonical URLs (SEO best practice)
	canonicalDomain := s.cfg.Domain[0]

	// construct the canonical URL
	url := fmt.Sprintf("%s://%s%s", s.cfg.Protocol, canonicalDomain, r.URL.Path)
	// construct the base URL
	baseURL := fmt.Sprintf("%s://%s", s.cfg.Protocol, canonicalDomain)

	return templateData{
		Form:         form,
		PinSize:      s.cfg.PinSize,
		CurrentYear:  time.Now().Year(),
		Theme:        getTheme(r),
		Branding:     s.cfg.Branding,
		URL:          url,
		BaseURL:      baseURL,
		FilesEnabled: s.cfg.EnableFiles,
		MaxFileSize:  s.cfg.MaxFileSize,
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
			// graceful shutdown with 10 second timeout
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil {
				log.Printf("[ERROR] failed to gracefully shutdown http server: %v", shutdownErr)
				// force close if graceful shutdown fails
				if clsErr := httpServer.Close(); clsErr != nil {
					log.Printf("[ERROR] failed to close http server: %v", clsErr)
				}
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

const defaultSizeLimit = 64 * 1024 // 64KB default for non-file routes

// sizeLimitMiddleware returns a middleware that applies appropriate size limits.
// file upload routes use MaxFileSize, all other routes use defaultSizeLimit.
func (s Server) sizeLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := int64(defaultSizeLimit)

			// use larger limit for file upload routes
			if s.cfg.EnableFiles && r.Method == http.MethodPost {
				path := r.URL.Path
				if path == "/api/v1/file" || path == "/generate-file-link" {
					limit = s.cfg.MaxFileSize + multipartFormOverhead
				}
			}

			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

const multipartFormOverhead = 4096 // extra bytes for form fields in multipart requests

func (s Server) routes() http.Handler {
	router := routegroup.New(http.NewServeMux())

	// global middleware - applied to all routes
	router.Use(
		rest.RealIP,
		rest.Recoverer(log.Default()),
		rest.Throttle(1000),
		Timeout(60*time.Second),
		rest.AppInfo("secrets", "Umputun", s.version),
		rest.Ping,
		s.sizeLimitMiddleware(),
		tollbooth.HTTPMiddleware(tollbooth.NewLimiter(10, nil)),
	)

	// API routes
	router.Mount("/api/v1").Route(func(apiGroup *routegroup.Bundle) {
		apiGroup.Use(Logger(log.Default()))
		apiGroup.HandleFunc("POST /message", s.saveMessageCtrl)
		apiGroup.HandleFunc("GET /message/{key}/{pin}", s.getMessageCtrl)
		apiGroup.HandleFunc("GET /params", s.getParamsCtrl)
		if s.cfg.EnableFiles {
			apiGroup.HandleFunc("POST /file", s.saveFileCtrl)
			apiGroup.HandleFunc("GET /file/{key}/{pin}", s.getFileCtrl)
		}
	})

	// web routes
	router.Group().Route(func(webGroup *routegroup.Bundle) {
		webGroup.Use(Logger(log.Default()), StripSlashes)
		webGroup.HandleFunc("POST /generate-link", s.generateLinkCtrl)
		webGroup.HandleFunc("GET /message/{key}", s.showMessageViewCtrl)
		webGroup.HandleFunc("POST /load-message", s.loadMessageCtrl)
		webGroup.HandleFunc("POST /theme", s.themeToggleCtrl)
		webGroup.HandleFunc("POST /copy-feedback", s.copyFeedbackCtrl)
		webGroup.HandleFunc("GET /close-popup", s.closePopupCtrl)
		webGroup.HandleFunc("GET /about", s.aboutViewCtrl)
		webGroup.HandleFunc("GET /{$}", s.indexCtrl) // exact match for root only
		webGroup.HandleFunc("POST /download-file", s.downloadFileCtrl)
		if s.cfg.EnableFiles {
			webGroup.HandleFunc("GET /form/text", s.textFormPartialCtrl)
			webGroup.HandleFunc("GET /form/file", s.fileFormPartialCtrl)
			webGroup.HandleFunc("POST /generate-file-link", s.generateFileLinkCtrl)
		}
	})

	// special routes without groups
	router.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		robotsContent := fmt.Sprintf("User-agent: *\nDisallow: /api/\nDisallow: /message/\nSitemap: %s://%s/sitemap.xml\n",
			s.cfg.Protocol, s.cfg.Domain[0])
		_, _ = w.Write([]byte(robotsContent))
	})

	router.HandleFunc("GET /sitemap.xml", s.sitemapCtrl)

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

	return router
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
	type paramsResp struct {
		PinSize        int   `json:"pin_size"`
		MaxPinAttempts int   `json:"max_pin_attempts"`
		MaxExpSecs     int   `json:"max_exp_sec"`
		FilesEnabled   bool  `json:"files_enabled,omitempty"`
		MaxFileSize    int64 `json:"max_file_size,omitempty"`
	}
	params := paramsResp{
		PinSize:        s.cfg.PinSize,
		MaxPinAttempts: s.cfg.MaxPinAttempts,
		MaxExpSecs:     int(s.cfg.MaxExpire.Seconds()),
	}
	if s.cfg.EnableFiles {
		params.FilesEnabled = true
		params.MaxFileSize = s.cfg.MaxFileSize
	}
	rest.RenderJSON(w, params)
}

// POST /api/v1/file - save file
func (s Server) saveFileCtrl(w http.ResponseWriter, r *http.Request) {
	// size limit is enforced by sizeLimitMiddleware
	if err := r.ParseMultipartForm(s.cfg.MaxFileSize); err != nil {
		log.Printf("[WARN] can't parse multipart form: %v", err)
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "file too large or invalid form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[WARN] no file in request: %v", err)
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "file required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("[ERROR] can't read file: %v", err)
		rest.SendErrorJSON(w, r, log.Default(), http.StatusInternalServerError, err, "failed to read file")
		return
	}

	pin := r.FormValue("pin")
	if len(pin) != s.cfg.PinSize {
		log.Printf("[WARN] incorrect pin size %d for file upload", len(pin))
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("incorrect pin size"), "incorrect pin size")
		return
	}

	expStr := r.FormValue("exp")
	exp, err := strconv.Atoi(expStr)
	if err != nil {
		log.Printf("[WARN] invalid expiration: %v", err)
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "invalid expiration")
		return
	}
	if exp <= 0 {
		log.Printf("[WARN] expiration must be positive: %d", exp)
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("expiration must be positive"), "expiration must be positive")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	msg, err := s.messager.MakeFileMessage(messager.FileRequest{
		Duration:    time.Second * time.Duration(exp),
		Data:        data,
		FileName:    header.Filename,
		ContentType: contentType,
		Pin:         pin,
	})
	if err != nil {
		log.Printf("[WARN] can't create file message: %v", err)
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "can't create file message")
		return
	}

	w.WriteHeader(http.StatusCreated)
	rest.RenderJSON(w, rest.JSON{"key": msg.Key, "exp": msg.Exp})
	log.Printf("[INFO] created file message %s, file=%s, size=%d, exp=%s", msg.Key, header.Filename, len(data), msg.Exp.Format(time.RFC3339))
}

// GET /api/v1/file/{key}/{pin} - get file
func (s Server) getFileCtrl(w http.ResponseWriter, r *http.Request) {
	key, pin := r.PathValue("key"), r.PathValue("pin")
	if key == "" || pin == "" || len(pin) != s.cfg.PinSize {
		log.Print("[WARN] no valid key or pin in get file request")
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("no key or pin passed"), "invalid request")
		return
	}

	serveRequest := func() (status int, msg *store.Message, errMsg string) {
		msg, err := s.messager.LoadMessage(key, pin)
		if err != nil {
			log.Printf("[WARN] failed to load file key %v", key)
			if errors.Is(err, messager.ErrBadPinAttempt) {
				return http.StatusExpectationFailed, nil, err.Error()
			}
			return http.StatusBadRequest, nil, err.Error()
		}
		if !msg.IsFile {
			return http.StatusBadRequest, nil, "not a file message"
		}
		return http.StatusOK, msg, ""
	}

	// make sure serveRequest works constant time on any branch
	st := time.Now()
	status, msg, errMsg := serveRequest()
	time.Sleep(time.Millisecond*100 - time.Since(st))

	if status != http.StatusOK {
		w.WriteHeader(status)
		rest.RenderJSON(w, rest.JSON{"error": errMsg})
		log.Printf("[INFO] file access failed %s, status %d", key, status)
		return
	}

	s.serveFileDownload(w, msg)
	log.Printf("[INFO] served file %s, name=%s, size=%d", key, msg.FileName, len(msg.Data))
}

// sitemapCtrl generates an XML sitemap for SEO
// GET /sitemap.xml
func (s Server) sitemapCtrl(w http.ResponseWriter, _ *http.Request) {
	baseURL := fmt.Sprintf("%s://%s", s.cfg.Protocol, s.cfg.Domain[0])

	// use current time for lastmod
	lastMod := time.Now().Format("2006-01-02")

	sitemap := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <url>
        <loc>%s/</loc>
        <lastmod>%s</lastmod>
        <changefreq>weekly</changefreq>
        <priority>1.0</priority>
    </url>
    <url>
        <loc>%s/about</loc>
        <lastmod>%s</lastmod>
        <changefreq>monthly</changefreq>
        <priority>0.8</priority>
    </url>
</urlset>`, baseURL, lastMod, baseURL, lastMod)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(sitemap))
}
