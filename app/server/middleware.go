package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
)

type ctxKey string

const hashedIPKey ctxKey = "hashedIP"

// HashedIP middleware adds anonymized IP to request context for audit logging.
// Must run after rest.RealIP middleware which sets r.RemoteAddr to the client IP.
func HashedIP(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := "-"
			if r.RemoteAddr != "" {
				ip = hashIP(r.RemoteAddr, secret)
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

			// get URL and mask sensitive parts
			q := r.URL.String()
			if qun, err := url.QueryUnescape(q); err == nil {
				q = qun
			}

			// hide key and pin in message paths
			if strings.Contains(q, "/message/") {
				elems := strings.Split(q, "/")
				for i, elem := range elems {
					if elem == "message" && i+2 < len(elems) && len(elems[i+1]) >= 18 {
						// show partial key, hide pin
						prefix := strings.Join(elems[:i+1], "/")
						q = fmt.Sprintf("%s/%s/*****", prefix, elems[i+1][:17])
						break
					}
				}
			}

			// get hashed IP from context (set by HashedIP middleware)
			remoteIP := GetHashedIP(r)

			l.Logf("[DEBUG] %s - %s - %s - %d - %v", r.Method, q, remoteIP, ww.status, duration)
		}
		return http.HandlerFunc(fn)
	}
}

// hashIP returns first 12 chars of HMAC-SHA1 hash for IP anonymization
func hashIP(ip, secret string) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(ip))
	return hex.EncodeToString(h.Sum(nil))[:12]
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
