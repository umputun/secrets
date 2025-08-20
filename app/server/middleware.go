package server

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
)

// Logger middleware with security masking for sensitive paths
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

			// get IP - use net.SplitHostPort which handles both IPv4 and IPv6
			remoteIP := "-"
			if r.RemoteAddr != "" {
				if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" {
					remoteIP = host
				}
			}

			l.Logf("[DEBUG] %s - %s - %s - %d - %v", r.Method, q, remoteIP, ww.status, duration)
		}
		return http.HandlerFunc(fn)
	}
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
