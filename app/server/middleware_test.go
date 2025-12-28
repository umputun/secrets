package server

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	req, err := http.NewRequest("GET", "/params", http.NoBody)
	require.NoError(t, err)
	req.RemoteAddr = "127.0.0.1:12345"

	rr := httptest.NewRecorder()
	out := bytes.Buffer{}
	l := lgr.New(lgr.Out(&out), lgr.Debug)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// chain HashedIP middleware before Logger (as in production)
	handler := HashedIP("test-secret")(Logger(l)(testHandler))
	handler.ServeHTTP(rr, req)
	t.Log(out.String())
	// IP is hashed, verify it's an 8-char hex string
	assert.Contains(t, out.String(), "DEBUG GET - /params -")
	assert.Contains(t, out.String(), "- 200")
	assert.NotContains(t, out.String(), "127.0.0.1") // IP should be hashed
}

func TestLoggerMasking(t *testing.T) {
	req, err := http.NewRequest("GET",
		"/message/5e4e1633-24b01ef6-49d6-4c8a-acf9-9dac0aa0eff9/1234567890", http.NoBody)
	require.NoError(t, err)
	req.RemoteAddr = "192.168.1.1:54321"

	rr := httptest.NewRecorder()
	out := bytes.Buffer{}
	l := lgr.New(lgr.Out(&out), lgr.Debug)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// chain HashedIP middleware before Logger (as in production)
	handler := HashedIP("test-secret")(Logger(l)(testHandler))
	handler.ServeHTTP(rr, req)
	t.Log(out.String())
	assert.Contains(t, out.String(), "DEBUG GET - /message/5e4e1633-24b01ef6/*****")
	assert.Contains(t, out.String(), "- 200")
	assert.NotContains(t, out.String(), "192.168.1.1") // IP should be hashed
}

func TestHashIP(t *testing.T) {
	// same IP + secret should produce same hash
	h1 := hashIP("192.168.1.1", "secret")
	h2 := hashIP("192.168.1.1", "secret")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 8)

	// different IPs should produce different hashes
	h3 := hashIP("192.168.1.2", "secret")
	assert.NotEqual(t, h1, h3)

	// different secrets should produce different hashes
	h4 := hashIP("192.168.1.1", "other-secret")
	assert.NotEqual(t, h1, h4)
}

func TestStripSlashes(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"root path unchanged", "/", "/"},
		{"path without slash unchanged", "/api/test", "/api/test"},
		{"trailing slash removed", "/api/test/", "/api/test"},
		{"multiple trailing slashes", "/api/test///", "/api/test//"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.path, http.NoBody)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			actualPath := ""
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				actualPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
			})

			handler := StripSlashes(testHandler)
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expected, actualPath)
		})
	}
}

func TestHashedIPMiddleware(t *testing.T) {
	t.Run("adds hashed IP to context", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", http.NoBody)
		require.NoError(t, err)
		req.RemoteAddr = "192.168.1.100:12345"

		rr := httptest.NewRecorder()
		var capturedIP string
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedIP = GetHashedIP(r)
			w.WriteHeader(http.StatusOK)
		})

		handler := HashedIP("test-secret")(testHandler)
		handler.ServeHTTP(rr, req)

		assert.Len(t, capturedIP, 8)
		assert.NotEqual(t, "-", capturedIP)
		assert.NotContains(t, capturedIP, "192.168.1.100")
	})

	t.Run("same IP produces same hash", func(t *testing.T) {
		var hash1, hash2 string

		for _, hash := range []*string{&hash1, &hash2} {
			req, err := http.NewRequest("GET", "/test", http.NoBody)
			require.NoError(t, err)
			req.RemoteAddr = "10.0.0.1:9999"

			rr := httptest.NewRecorder()
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				*hash = GetHashedIP(r)
				w.WriteHeader(http.StatusOK)
			})

			handler := HashedIP("consistent-secret")(testHandler)
			handler.ServeHTTP(rr, req)
		}

		assert.Equal(t, hash1, hash2)
	})

	t.Run("different IPs produce different hashes", func(t *testing.T) {
		var hash1, hash2 string

		for _, tc := range []struct {
			ip   string
			hash *string
		}{
			{"10.0.0.1:1234", &hash1},
			{"10.0.0.2:1234", &hash2},
		} {
			req, err := http.NewRequest("GET", "/test", http.NoBody)
			require.NoError(t, err)
			req.RemoteAddr = tc.ip

			rr := httptest.NewRecorder()
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				*tc.hash = GetHashedIP(r)
				w.WriteHeader(http.StatusOK)
			})

			handler := HashedIP("test-secret")(testHandler)
			handler.ServeHTTP(rr, req)
		}

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("empty remote addr returns dash", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", http.NoBody)
		require.NoError(t, err)
		req.RemoteAddr = ""

		rr := httptest.NewRecorder()
		var capturedIP string
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedIP = GetHashedIP(r)
			w.WriteHeader(http.StatusOK)
		})

		handler := HashedIP("test-secret")(testHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, "-", capturedIP)
	})

	t.Run("ip without port (behind proxy)", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", http.NoBody)
		require.NoError(t, err)
		req.RemoteAddr = "10.0.0.50" // no port - this is what RealIP sets behind proxy

		rr := httptest.NewRecorder()
		var capturedIP string
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedIP = GetHashedIP(r)
			w.WriteHeader(http.StatusOK)
		})

		handler := HashedIP("test-secret")(testHandler)
		handler.ServeHTTP(rr, req)

		assert.Len(t, capturedIP, 8)
		assert.NotEqual(t, "-", capturedIP)
		assert.NotContains(t, capturedIP, "10.0.0.50")
	})
}

func TestGetHashedIP(t *testing.T) {
	t.Run("returns dash when not in context", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", http.NoBody)
		require.NoError(t, err)

		// no middleware applied, context doesn't have hashed IP
		ip := GetHashedIP(req)
		assert.Equal(t, "-", ip)
	})
}

func TestTimeout(t *testing.T) {
	t.Run("request completes before timeout", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		})

		handler := Timeout(100 * time.Millisecond)(testHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "success", rr.Body.String())
	})

	t.Run("request times out", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		handler := Timeout(50 * time.Millisecond)(testHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
		assert.Contains(t, rr.Body.String(), "Request timeout")
	})
}

func TestRequireHTMX_WithHeader(t *testing.T) {
	req, err := http.NewRequest("POST", "/generate-link", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireHTMX(testHandler)
	handler.ServeHTTP(rr, req)

	assert.True(t, handlerCalled, "handler should be called when HX-Request header present")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireHTMX_WithoutHeader(t *testing.T) {
	req, err := http.NewRequest("POST", "/generate-link", http.NoBody)
	require.NoError(t, err)
	// no HX-Request header

	rr := httptest.NewRecorder()
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireHTMX(testHandler)
	handler.ServeHTTP(rr, req)

	assert.False(t, handlerCalled, "handler should not be called without HX-Request header")
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "JavaScript is required")
}

func TestLoggerMaskingEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		shouldMask   bool
		expectedPart string // partial string that should appear in log
	}{
		{"normal message path", "/message/5e4e1633-24b01ef6-49d6-4c8a-acf9-9dac0aa0eff9/12345", true, "/message/5e4e1633-24b01ef6/*****"},
		{"short key not masked", "/message/short/12345", false, "/message/short/12345"},
		{"no pin segment", "/message/5e4e1633-24b01ef6-49d6-4c8a-acf9", false, "/message/5e4e1633-24b01ef6-49d6-4c8a-acf9"},
		{"message at end", "/api/message", false, "/api/message"},
		{"nested message path", "/v1/api/message/5e4e1633-24b01ef6-49d6-4c8a-acf9-9dac0aa0eff9/pin123", true, "/v1/api/message/5e4e1633-24b01ef6/*****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.path, http.NoBody)
			require.NoError(t, err)
			req.RemoteAddr = "10.0.0.1:1234"

			rr := httptest.NewRecorder()
			out := bytes.Buffer{}
			l := lgr.New(lgr.Out(&out), lgr.Debug)
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := HashedIP("test-secret")(Logger(l)(testHandler))
			handler.ServeHTTP(rr, req)

			logOutput := out.String()
			assert.Contains(t, logOutput, tt.expectedPart, "log should contain expected path")
			if tt.shouldMask {
				assert.NotContains(t, logOutput, "12345", "pin should be masked")
				assert.NotContains(t, logOutput, "pin123", "pin should be masked")
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Run("sets all security headers for https", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := SecurityHeaders("https")(testHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
		assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "strict-origin-when-cross-origin", rr.Header().Get("Referrer-Policy"))
		assert.Equal(t, "max-age=31536000; includeSubDomains", rr.Header().Get("Strict-Transport-Security"))

		csp := rr.Header().Get("Content-Security-Policy")
		assert.Contains(t, csp, "default-src 'self'")
		assert.Contains(t, csp, "script-src 'self'")
		assert.Contains(t, csp, "style-src 'self' https://fonts.googleapis.com")
		assert.Contains(t, csp, "font-src 'self' https://fonts.gstatic.com")
		assert.Contains(t, csp, "form-action 'self'")
		assert.Contains(t, csp, "frame-ancestors 'none'")
	})

	t.Run("skips HSTS for http protocol", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := SecurityHeaders("http")(testHandler)
		handler.ServeHTTP(rr, req)

		// other headers still set
		assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
		assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "strict-origin-when-cross-origin", rr.Header().Get("Referrer-Policy"))

		// HSTS not set for HTTP
		assert.Empty(t, rr.Header().Get("Strict-Transport-Security"))

		// CSP still set
		csp := rr.Header().Get("Content-Security-Policy")
		assert.Contains(t, csp, "default-src 'self'")
	})

	t.Run("sets no-cache for dynamic content", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/message", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := SecurityHeaders("https")(testHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, "no-cache, no-store, must-revalidate", rr.Header().Get("Cache-Control"))
	})

	t.Run("sets long cache for static assets", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/static/js/htmx.min.js", http.NoBody)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := SecurityHeaders("https")(testHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, "public, max-age=31536000, immutable", rr.Header().Get("Cache-Control"))
	})
}

func TestSendErrorJSON(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/test", http.NoBody)
	require.NoError(t, err)
	req.RemoteAddr = "192.168.1.100:12345"

	// apply HashedIP middleware to set hashed IP in context
	rr := httptest.NewRecorder()
	out := bytes.Buffer{}
	l := lgr.New(lgr.Out(&out))

	var capturedReq *http.Request
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		SendErrorJSON(w, r, l, http.StatusNotFound, errors.New("test error"), "endpoint not found")
	})

	handler := HashedIP("test-secret")(testHandler)
	handler.ServeHTTP(rr, req)

	// verify response
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), `"error":"endpoint not found"`)

	// verify log uses hashed IP, not real IP
	logOutput := out.String()
	assert.NotContains(t, logOutput, "192.168.1.100", "real IP should not appear in log")
	assert.Contains(t, logOutput, "endpoint not found")
	assert.Contains(t, logOutput, "/api/v1/test")

	// verify hashed IP was used
	hashedIP := GetHashedIP(capturedReq)
	assert.Len(t, hashedIP, 8)
	assert.Contains(t, logOutput, hashedIP)
}
