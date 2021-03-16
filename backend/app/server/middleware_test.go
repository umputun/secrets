package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/params", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	out := bytes.Buffer{}
	l := lgr.New(lgr.Out(&out))
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%v", r)
	})

	handler := Logger(l)(testHandler)
	handler.ServeHTTP(rr, req)
	t.Log(out.String())
	assert.Contains(t, out.String(), "INFO  REST GET - /api/v1/params")
}

func TestLoggerMasking(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/message/5e4e1633-24b01ef6-49d6-4c8a-acf9-9dac0aa0eff9/1234567890", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	out := bytes.Buffer{}
	l := lgr.New(lgr.Out(&out))
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%v", r)
	})

	handler := Logger(l)(testHandler)
	handler.ServeHTTP(rr, req)
	t.Log(out.String())
	assert.Contains(t, out.String(), "INFO  REST GET - /api/v1/message/5e4e1633-24b01ef6/***** ")
}
