// Package server provides rest-like api and serves static assets as well
package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/umputun/secrets/v2/app/email"
	"github.com/umputun/secrets/v2/app/messager"
	"github.com/umputun/secrets/v2/app/server/assets"
	"github.com/umputun/secrets/v2/app/store"
)

// Config is a configuration for the server
type Config struct {
	Domain   []string // allowed domains list
	WebRoot  string
	Protocol string
	Branding string
	Listen   string // server listen address (ip:port or :port), defaults to :8080
	SignKey  string // sign key (will be hashed before use for IP anonymization)

	// validation parameters
	PinSize        int
	MaxPinAttempts int
	MaxExpire      time.Duration

	// file upload settings
	EnableFiles bool
	MaxFileSize int64 // bytes, 0 means use default (1MB)

	// authentication (optional)
	AuthHash   string        // bcrypt hash of password, empty disables auth
	SessionTTL time.Duration // session lifetime, defaults to 168h (7 days)

	EmailEnabled           bool // email sharing (optional)
	Paranoid               bool // paranoid mode - client-side encryption only
	DisableSecurityHeaders bool // skip security headers when proxy handles them
}

//go:generate moq -out mocks/email_sender_mock.go -pkg mocks -skip-ensure -fmt goimports . EmailSender

// EmailSender defines the interface for sending emails (consumer-side interface)
type EmailSender interface {
	Send(ctx context.Context, req email.Request) error
	GetDefaultFromName() string
}

// Server is a rest with store
type Server struct {
	messager      Messager
	emailSender   EmailSender
	cfg           Config
	version       string
	templateCache map[string]*template.Template
	logSecret     string // derived from SignKey for IP anonymization in logs
}

// New creates a new server with template cache
func New(m Messager, version string, cfg Config) (Server, error) {
	if len(cfg.Domain) == 0 {
		return Server{}, errors.New("at least one domain must be configured")
	}

	cache, err := newTemplateCache()
	if err != nil {
		return Server{}, fmt.Errorf("can't create template cache: %w", err)
	}

	// derive log secret from sign key for IP anonymization (never use raw SignKey for logging)
	h := sha256.Sum256([]byte(cfg.SignKey + ":log"))
	logSecret := hex.EncodeToString(h[:])

	return Server{
		messager:      m,
		cfg:           cfg,
		version:       version,
		templateCache: cache,
		logSecret:     logSecret,
	}, nil
}

// WithEmail sets the email sender for the server
func (s Server) WithEmail(sender EmailSender) Server {
	s.emailSender = sender
	return s
}

// Messager interface making and loading messages
type Messager interface {
	MakeMessage(ctx context.Context, duration time.Duration, msg, pin string) (result *store.Message, err error)
	MakeFileMessage(ctx context.Context, req messager.FileRequest) (result *store.Message, err error)
	LoadMessage(ctx context.Context, key, pin string) (msg *store.Message, err error)
	IsFile(ctx context.Context, key string) bool // checks if message is a file without decrypting
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
		Paranoid:     s.cfg.Paranoid,
	}
}

// Run the lister and request's router, activate rest server
func (s Server) Run(ctx context.Context) error {
	log.Printf("[INFO] activate rest server")

	port := s.cfg.Listen
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

func (s Server) routes() http.Handler {
	router := routegroup.New(http.NewServeMux())

	// determine size limit based on mode and whether files are enabled
	sizeLimit := int64(64 * 1024) // 64KB default for text-only
	if s.cfg.Paranoid {
		// paranoid mode: base64 adds ~33% overhead, use MaxFileSize * 1.4 for all requests
		// (server can't distinguish text from files in paranoid mode)
		sizeLimit = int64(float64(s.cfg.MaxFileSize) * 1.4)
	} else if s.cfg.EnableFiles {
		sizeLimit = s.cfg.MaxFileSize + 10*1024 // file size + 10KB for form overhead
	}

	// global middleware - applied to all routes
	router.Use(
		rest.RealIP, // x-Real-IP → CF-Connecting-IP → leftmost public XFF → RemoteAddr
		HashedIP(s.logSecret),
		rest.Recoverer(log.Default()),
		rest.Throttle(1000),
		Timeout(60*time.Second),
		rest.AppInfo("secrets", "Umputun", s.version),
		rest.Ping,
		rest.SizeLimit(sizeLimit),
		tollbooth.HTTPMiddleware(tollbooth.NewLimiter(10, nil)),
	)

	// security headers - enabled by default, disabled with --proxy-security-headers
	if !s.cfg.DisableSecurityHeaders {
		router.Use(SecurityHeaders(s.cfg.Protocol))
	}

	// API routes
	router.Mount("/api/v1").Route(func(apiGroup *routegroup.Bundle) {
		apiGroup.Use(Logger(log.Default()))
		apiGroup.HandleFunc("POST /message", s.saveMessageCtrl)
		apiGroup.HandleFunc("GET /message/{key}/{pin}", s.getMessageCtrl)
		apiGroup.HandleFunc("GET /params", s.getParamsCtrl)
	})

	// auth routes (only if auth enabled)
	if s.cfg.AuthHash != "" {
		router.HandleFunc("POST /login", s.loginCtrl)
		router.HandleFunc("GET /logout", s.logoutCtrl)
		router.HandleFunc("GET /login-popup", s.loginPopupCtrl)
	}

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

		// email routes (only if email is enabled)
		if s.cfg.EmailEnabled {
			webGroup.HandleFunc("GET /email-popup", s.emailPopupCtrl)
			webGroup.HandleFunc("POST /send-email", s.sendEmailCtrl)
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
		staticFS, err := fs.Sub(assets.Files, "static")
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
	// check basic auth if auth is enabled
	if s.cfg.AuthHash != "" && !s.checkBasicAuth(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="secrets"`)
		rest.SendErrorJSON(w, r, log.Default(), http.StatusUnauthorized, errors.New("unauthorized"), "authentication required")
		return
	}

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

	msg, err := s.messager.MakeMessage(r.Context(), time.Second*time.Duration(request.Exp), request.Message, request.Pin)
	if err != nil {
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, err, "can't create message")
		return
	}
	_ = rest.EncodeJSON(w, http.StatusCreated, rest.JSON{"key": msg.Key, "exp": msg.Exp})
	log.Printf("[INFO] created message %s, type=text, size=%d, exp=%s, ip=%s",
		msg.Key, len(request.Message), msg.Exp.Format(time.RFC3339), GetHashedIP(r))
}

// GET /v1/message/{key}/{pin}
func (s Server) getMessageCtrl(w http.ResponseWriter, r *http.Request) {
	key, pin := r.PathValue("key"), r.PathValue("pin")
	if key == "" || pin == "" || len(pin) != s.cfg.PinSize {
		log.Print("[WARN] no valid key or pin in get request")
		rest.SendErrorJSON(w, r, log.Default(), http.StatusBadRequest, errors.New("no key or pin passed"), "invalid request")
		return
	}

	msgType := "unknown"
	serveRequest := func() (status int, res rest.JSON) {
		msg, err := s.messager.LoadMessage(r.Context(), key, pin)
		if err != nil {
			log.Printf("[WARN] failed to load key %v", key)
			if errors.Is(err, messager.ErrBadPinAttempt) {
				return http.StatusExpectationFailed, rest.JSON{"error": err.Error()}
			}
			return http.StatusBadRequest, rest.JSON{"error": err.Error()}
		}
		// determine message type for logging
		if messager.IsFileMessage(msg.Data) {
			msgType = "file"
			// reject file messages when files are disabled
			if !s.cfg.EnableFiles {
				log.Printf("[WARN] file download rejected for %s, files disabled", key)
				return http.StatusForbidden, rest.JSON{"error": "file downloads disabled"}
			}
		} else {
			msgType = "text"
		}
		return http.StatusOK, rest.JSON{"key": msg.Key, "message": string(msg.Data)}
	}

	// make sure serveRequest takes constant time on any branch to prevent timing attacks
	st := time.Now()
	status, res := serveRequest()
	if elapsed := time.Since(st); elapsed < 100*time.Millisecond {
		time.Sleep(100*time.Millisecond - elapsed)
	}
	_ = rest.EncodeJSON(w, status, res)

	var statusText string
	switch status {
	case 200:
		statusText = "success"
	case 417:
		statusText = "wrong pin"
	default:
		statusText = "error"
	}
	log.Printf("[INFO] accessed message %s, type=%s, status=%d (%s), ip=%s",
		key, msgType, status, statusText, GetHashedIP(r))
}

// GET /params
func (s Server) getParamsCtrl(w http.ResponseWriter, _ *http.Request) {
	params := struct {
		PinSize        int   `json:"pin_size"`
		MaxPinAttempts int   `json:"max_pin_attempts"`
		MaxExpSecs     int   `json:"max_exp_sec"`
		FilesEnabled   bool  `json:"files_enabled"`
		MaxFileSize    int64 `json:"max_file_size"`
		Paranoid       bool  `json:"paranoid"`
	}{
		PinSize:        s.cfg.PinSize,
		MaxPinAttempts: s.cfg.MaxPinAttempts,
		MaxExpSecs:     int(s.cfg.MaxExpire.Seconds()),
		FilesEnabled:   s.cfg.EnableFiles,
		MaxFileSize:    s.cfg.MaxFileSize,
		Paranoid:       s.cfg.Paranoid,
	}
	rest.RenderJSON(w, params)
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
