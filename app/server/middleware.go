package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
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

// HashedIP middleware adds anonymized IP to request context for audit logging
func HashedIP(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := "-"
			if host := extractIP(r.RemoteAddr); host != "" {
				ip = hashIP(host, secret)
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

// Logger middleware with security masking for sensitive paths and IP anonymization
func Logger(l log.L, secret string) func(http.Handler) http.Handler {
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

			// get IP and hash it for privacy
			remoteIP := "-"
			if host := extractIP(r.RemoteAddr); host != "" {
				remoteIP = hashIP(host, secret)
			}

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

// extractIP extracts IP address from RemoteAddr which may be "ip:port" or just "ip".
// rest.RealIP middleware sets RemoteAddr without port when behind reverse proxy.
func extractIP(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	// try splitting host:port first
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	// no port present - check if it's a valid IP directly
	if ip := net.ParseIP(remoteAddr); ip != nil {
		return remoteAddr
	}
	return ""
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
