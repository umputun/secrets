package server

import (
	"bytes"
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

	handler := Logger(l, "test-secret")(testHandler)
	handler.ServeHTTP(rr, req)
	t.Log(out.String())
	// IP is hashed, verify it's a 12-char hex string
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

	handler := Logger(l, "test-secret")(testHandler)
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
	assert.Len(t, h1, 12)

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

		assert.Len(t, capturedIP, 12)
		assert.NotEqual(t, "-", capturedIP)
		assert.NotContains(t, capturedIP, "192.168.1.100")
	})

	t.Run("same IP produces same hash", func(t *testing.T) {
		var hash1, hash2 string

		for i, hash := range []*string{&hash1, &hash2} {
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
			_ = i
		}

		assert.Equal(t, hash1, hash2)
	})

	t.Run("different IPs produce different hashes", func(t *testing.T) {
		var hash1, hash2 string

		for i, tc := range []struct {
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
			_ = i
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
