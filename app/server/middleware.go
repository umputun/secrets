package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
)

type ctxKey string

const hashedIPKey ctxKey = "hashedIP"

// HashedIP middleware adds anonymized IP to request context for audit logging.
// Must run after rest.RealIP when proxy headers are trusted, it reads r.RemoteAddr.
func HashedIP(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := "-"
			if r.RemoteAddr != "" {
				// without rest.RealIP in the chain the address carries the ephemeral port,
				// which would give every connection a different hash
				addr := r.RemoteAddr
				if host, _, err := net.SplitHostPort(addr); err == nil {
					addr = host
				}
				ip = hashIP(addr, secret)
			}
			ctx := context.WithValue(r.Context(), hashedIPKey, ip)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetHashedIP retrieves hashed IP from context
func GetHashedIP(r *http.Request) string {
	if ip, ok := r.Context().Value(hashedIPKey).(string); ok {
		return ip
	}
	return "-"
}

// Logger middleware with security masking for sensitive paths and IP anonymization.
// Must run after HashedIP middleware which sets hashed IP in context.
func Logger(l log.L) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := &statusWriter{ResponseWriter: w, status: 200}
			start := time.Now()

			h.ServeHTTP(ww, r)

			duration := time.Since(start)

			q := sanitizedURL(r.URL)

			// get hashed IP from context (set by HashedIP middleware)
			remoteIP := GetHashedIP(r)

			l.Logf("[DEBUG] %s - %s - %s - %d - %v", r.Method, q, remoteIP, ww.status, duration)
		}
		return http.HandlerFunc(fn)
	}
}

func sanitizedURL(u *url.URL) string {
	elems := strings.Split(u.Path, "/")
	for i, elem := range elems {
		if elem != "message" || i+2 >= len(elems) {
			continue
		}
		key := elems[i+1]
		elems[i+1] = key[:min(17, len(key)/2)]
		prefix := &url.URL{Path: strings.Join(elems[:i+2], "/")}
		return prefix.EscapedPath() + "/*****"
	}
	if strings.TrimRight(u.Path, "/") == "/email-popup" {
		return u.EscapedPath()
	}
	return u.RequestURI()
}

// hashIP returns first 8 chars of HMAC-SHA256 hash for IP anonymization
func hashIP(ip, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(ip))
	return hex.EncodeToString(h.Sum(nil))[:8]
}

// statusWriter wraps http.ResponseWriter to capture status code
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// SendErrorJSON sends error response and logs with hashed IP (privacy-safe alternative to rest.SendErrorJSON)
func SendErrorJSON(w http.ResponseWriter, r *http.Request, l log.L, code int, err error, msg string) {
	hashedIP := GetHashedIP(r)
	l.Logf("[INFO] %s - %s - %d - %s [caused by %v]", msg, hashedIP, code, sanitizedURL(&url.URL{Path: r.URL.Path}), err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, `{"error":%q}`, msg)
}

// StripSlashes removes trailing slashes from URLs
func StripSlashes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
			r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		}
		next.ServeHTTP(w, r)
	})
}

// Timeout creates a timeout middleware
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "Request timeout")
	}
}

// SecurityHeaders adds security headers to all responses.
// Disable with --proxy-security-headers when running behind a proxy that sets these.
func SecurityHeaders(protocol string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			if protocol == "https" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// CSP: strict policy with no inline scripts or eval
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self'; "+
					"style-src 'self' https://fonts.googleapis.com 'unsafe-inline'; "+
					"font-src 'self' https://fonts.gstatic.com; "+
					"img-src 'self' data:; "+
					"connect-src 'self'; "+
					"form-action 'self'; "+
					"frame-ancestors 'none'")

			// cache-control: static assets can be cached, everything else should not
			if strings.HasPrefix(r.URL.Path, "/static/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireHTMX middleware rejects requests without HX-Request header.
// Used for web form routes that require client-side JavaScript encryption.
func RequireHTMX(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("HX-Request") != "true" {
			http.Error(w, "JavaScript is required for this service", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
