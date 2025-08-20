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

	handler := Logger(l)(testHandler)
	handler.ServeHTTP(rr, req)
	t.Log(out.String())
	assert.Contains(t, out.String(), "DEBUG GET - /params - 127.0.0.1 - 200")
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

	handler := Logger(l)(testHandler)
	handler.ServeHTTP(rr, req)
	t.Log(out.String())
	assert.Contains(t, out.String(), "DEBUG GET - /message/5e4e1633-24b01ef6/***** - 192.168.1.1 - 200")
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
