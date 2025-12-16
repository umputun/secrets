package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/secrets/app/messager"
	"github.com/umputun/secrets/app/store"
)

func TestServer_saveAndLoadMemory(t *testing.T) {
	ts, teardown := prepTestServer(t)
	defer teardown()

	// save message
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/message", strings.NewReader(`{"message": "my secret message","exp": 600,"pin": "12345"}`))
	require.NoError(t, err)
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 201, resp.StatusCode)

	respSave := struct {
		Key string
		Exp time.Time
	}{}
	err = json.NewDecoder(resp.Body).Decode(&respSave)
	require.NoError(t, err)
	t.Logf("%+v", respSave)

	// load saved message
	url := fmt.Sprintf("%s/api/v1/message/%s/12345", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 200, resp.StatusCode)

	respLoad := struct {
		Key     string
		Message string
	}{}
	err = json.NewDecoder(resp.Body).Decode(&respLoad)
	require.NoError(t, err)
	t.Logf("%+v", respLoad)
	assert.Equal(t, struct {
		Key     string
		Message string
	}{Key: respLoad.Key, Message: "my secret message"}, respLoad)

	// second load should fail
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 400, resp.StatusCode)
}

func TestServer_saveAndLoadBolt(t *testing.T) {
	eng, err := store.NewBolt("/tmp/secrets-test.bdb", 1*time.Minute)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove("/tmp/secrets-test.bdb"))
	}()
	signKey := messager.MakeSignKey("stew-pub-barcan-scatty-daimio-wicker-yakona", 5)
	srv, err := New(
		messager.New(eng, messager.Crypt{Key: signKey}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:         []string{"example.com"},
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Safe Secrets",
		})

	require.NoError(t, err)

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	// save message
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/message", strings.NewReader(`{"message": "my secret message","exp": 600,"pin": "12345"}`))
	require.NoError(t, err)
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 201, resp.StatusCode)

	respSave := struct {
		Key string
		Exp time.Time
	}{}
	err = json.NewDecoder(resp.Body).Decode(&respSave)
	require.NoError(t, err)
	t.Logf("%+v", respSave)

	// load saved message
	url := fmt.Sprintf("%s/api/v1/message/%s/12345", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 200, resp.StatusCode)

	respLoad := struct {
		Key     string
		Message string
	}{}
	err = json.NewDecoder(resp.Body).Decode(&respLoad)
	require.NoError(t, err)
	t.Logf("%+v", respLoad)
	assert.Equal(t, struct {
		Key     string
		Message string
	}{Key: respLoad.Key, Message: "my secret message"}, respLoad)

	// second load should fail
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 400, resp.StatusCode)
}

func TestServer_saveAndManyPinAttempt(t *testing.T) {
	ts, teardown := prepTestServer(t)
	defer teardown()

	// save message
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/message", strings.NewReader(`{"message": "my secret message","exp": 600,"pin": "12345"}`))
	require.NoError(t, err)
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 201, resp.StatusCode)

	respSave := struct {
		Key string
		Exp time.Time
	}{}
	err = json.NewDecoder(resp.Body).Decode(&respSave)
	require.NoError(t, err)
	t.Logf("%+v", respSave)

	// load saved message
	url := fmt.Sprintf("%s/api/v1/message/%s/00000", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 417, resp.StatusCode)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 417, resp.StatusCode)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 400, resp.StatusCode)

	// try with a valid pin will fail, too many attempt
	url = fmt.Sprintf("%s/api/v1/message/%s/12345", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 400, resp.StatusCode)
}

func TestServer_saveAndGoodPinAttempt(t *testing.T) {
	ts, teardown := prepTestServer(t)
	defer teardown()

	// save message
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/message", strings.NewReader(`{"message": "my secret message","exp": 600,"pin": "12345"}`))
	require.NoError(t, err)
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 201, resp.StatusCode)

	respSave := struct {
		Key string
		Exp time.Time
	}{}
	err = json.NewDecoder(resp.Body).Decode(&respSave)
	require.NoError(t, err)
	t.Logf("%+v", respSave)

	// load saved message
	url := fmt.Sprintf("%s/api/v1/message/%s/00000", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 417, resp.StatusCode)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 417, resp.StatusCode)

	// try with a valid pin will pass, not too many attempt
	url = fmt.Sprintf("%s/api/v1/message/%s/12345", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 200, resp.StatusCode)
}

func TestServer_getParams(t *testing.T) {
	ts, teardown := prepTestServer(t)
	defer teardown()

	client := http.Client{Timeout: time.Second}
	url := fmt.Sprintf("%s/api/v1/params", ts.URL)
	req, err := http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() // nolint
	assert.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"pin_size":5,"max_pin_attempts":3,"max_exp_sec":36000}`+"\n", string(body))
}

func TestServer_saveMessageCtrl(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:         []string{"example.com"},
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Safe Secrets",
		})
	require.NoError(t, err)

	tests := []struct {
		name               string
		body               string
		expectedStatusCode int
		checkResponse      func(t *testing.T, body []byte)
	}{
		{name: "valid message", body: `{"message": "secret", "exp": 600, "pin": "12345"}`, expectedStatusCode: 201, checkResponse: func(t *testing.T, body []byte) {
			var resp map[string]any
			err := json.Unmarshal(body, &resp)
			require.NoError(t, err)
			assert.NotEmpty(t, resp["key"])
			assert.NotEmpty(t, resp["exp"])
		}},
		{name: "empty message is allowed", body: `{"message": "", "exp": 600, "pin": "12345"}`, expectedStatusCode: 201},
		{name: "exp exceeds max", body: `{"message": "secret", "exp": 999999, "pin": "12345"}`, expectedStatusCode: 400},
		{name: "invalid json", body: `{invalid json}`, expectedStatusCode: 400},
		{name: "missing pin", body: `{"message": "secret", "exp": 600}`, expectedStatusCode: 400},
		{name: "wrong pin length", body: `{"message": "secret", "exp": 600, "pin": "123"}`, expectedStatusCode: 400},
		{name: "non-numeric pin is allowed", body: `{"message": "secret", "exp": 600, "pin": "abcde"}`, expectedStatusCode: 201},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/message", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			srv.saveMessageCtrl(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rr.Body.Bytes())
			}
		})
	}
}

func TestServer_getMessageCtrl(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:         []string{"example.com"},
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Safe Secrets",
		})
	require.NoError(t, err)

	// create test server with actual routes
	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	// save a message first
	msg, err := srv.messager.MakeMessage(time.Hour, "test secret", "12345")
	require.NoError(t, err)

	tests := []struct {
		name               string
		key                string
		pin                string
		expectedStatusCode int
		checkResponse      func(t *testing.T, body []byte)
	}{
		{name: "valid key and pin", key: msg.Key, pin: "12345", expectedStatusCode: 200, checkResponse: func(t *testing.T, body []byte) {
			var resp map[string]any
			err := json.Unmarshal(body, &resp)
			require.NoError(t, err)
			assert.Equal(t, "test secret", resp["message"])
		}},
		{name: "invalid pin returns 400 when key not found", key: msg.Key, pin: "99999", expectedStatusCode: 400},
		{name: "non-existent key", key: "badkey", pin: "12345", expectedStatusCode: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/api/v1/message/%s/%s", ts.URL, tt.key, tt.pin)

			resp, err := http.Get(url) // #nosec G107 - URL is constructed from test data
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatusCode, resp.StatusCode)

			if tt.checkResponse != nil {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				tt.checkResponse(t, body)
			}
		})
	}
}

func TestServer_Run(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:         []string{"example.com"},
			Port:           ":0", // use random available port
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Safe Secrets",
		})
	require.NoError(t, err)

	// test that server can start and stop cleanly
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx)
	}()

	// give server time to start
	time.Sleep(50 * time.Millisecond)

	// cancel context to stop server
	cancel()

	// wait for server to stop
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("server didn't stop in time")
	}
}

func TestServer_EmbeddedFiles(t *testing.T) {
	// test with non-existent WebRoot to trigger embedded files usage
	eng := store.NewInMemory(time.Second * 5)
	msg := messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
		MaxDuration:    10 * time.Hour,
		MaxPinAttempts: 3,
	})
	srv, err := New(msg, "test", Config{
		Domain:         []string{"example.com"},
		WebRoot:        "/non/existent/path", // non-existent path to trigger embedded files
		PinSize:        5,
		MaxPinAttempts: 3,
		MaxExpire:      10 * time.Hour,
		Branding:       "Safe Secrets",
	})
	require.NoError(t, err)

	router := srv.routes()
	ts := httptest.NewServer(router)
	defer ts.Close()

	// test that static files are served from embedded FS
	resp, err := http.Get(ts.URL + "/static/css/main.css")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/css")
}

func TestServer_LocalFiles(t *testing.T) {
	// test with valid WebRoot
	tmpDir, err := os.MkdirTemp("", "secrets-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// create a test css file
	cssDir := tmpDir + "/css"
	err = os.MkdirAll(cssDir, 0o750)
	require.NoError(t, err)
	err = os.WriteFile(cssDir+"/test.css", []byte("body { color: red; }"), 0o600)
	require.NoError(t, err)

	eng := store.NewInMemory(time.Second * 5)
	msg := messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
		MaxDuration:    10 * time.Hour,
		MaxPinAttempts: 3,
	})
	srv, err := New(msg, "test", Config{
		Domain:         []string{"example.com"},
		WebRoot:        tmpDir,
		PinSize:        5,
		MaxPinAttempts: 3,
		MaxExpire:      10 * time.Hour,
		Branding:       "Safe Secrets",
	})
	require.NoError(t, err)

	router := srv.routes()
	ts := httptest.NewServer(router)
	defer ts.Close()

	// test that static files are served from local FS
	resp, err := http.Get(ts.URL + "/static/css/test.css")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "body { color: red; }", string(body))
}

func TestServer_ThemeToggle(t *testing.T) {
	eng := store.NewInMemory(time.Second * 5)
	msg := messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
		MaxDuration:    10 * time.Hour,
		MaxPinAttempts: 3,
	})
	srv, err := New(msg, "test", Config{
		Domain:         []string{"example.com"},
		WebRoot:        "",
		PinSize:        5,
		MaxPinAttempts: 3,
		MaxExpire:      10 * time.Hour,
		Branding:       "Safe Secrets",
	})
	require.NoError(t, err)

	router := srv.routes()
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := &http.Client{}

	// test theme toggle from auto (default) to light
	req, err := http.NewRequest("POST", ts.URL+"/theme", http.NoBody)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "theme", Value: "auto"})
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "true", resp.Header.Get("HX-Refresh"))
	// verify cookie was set to light
	cookies := resp.Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "theme", cookies[0].Name)
	assert.Equal(t, "light", cookies[0].Value)

	// test theme toggle from light to dark
	req, err = http.NewRequest("POST", ts.URL+"/theme", http.NoBody)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "theme", Value: "light"})
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	// verify cookie was set to dark
	cookies = resp.Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "dark", cookies[0].Value)

	// test theme toggle from dark to light
	req, err = http.NewRequest("POST", ts.URL+"/theme", http.NoBody)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	// verify cookie was set to light
	cookies = resp.Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "light", cookies[0].Value)
}

func TestServer_ClosePopup(t *testing.T) {
	eng := store.NewInMemory(time.Second * 5)
	msg := messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
		MaxDuration:    10 * time.Hour,
		MaxPinAttempts: 3,
	})
	srv, err := New(msg, "test", Config{
		Domain:         []string{"example.com"},
		WebRoot:        "",
		PinSize:        5,
		MaxPinAttempts: 3,
		MaxExpire:      10 * time.Hour,
		Branding:       "Safe Secrets",
	})
	require.NoError(t, err)

	router := srv.routes()
	ts := httptest.NewServer(router)
	defer ts.Close()

	// test close popup
	resp, err := http.Get(ts.URL + "/close-popup")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "\n<div id=\"popup\" class=\"popup\"></div>\n", string(body))
}

func TestServer_CopyFeedback(t *testing.T) {
	eng := store.NewInMemory(time.Second * 5)
	msg := messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
		MaxDuration:    10 * time.Hour,
		MaxPinAttempts: 3,
	})
	srv, err := New(msg, "test", Config{
		Domain:         []string{"example.com"},
		WebRoot:        "",
		PinSize:        5,
		MaxPinAttempts: 3,
		MaxExpire:      10 * time.Hour,
		Branding:       "Safe Secrets",
	})
	require.NoError(t, err)

	router := srv.routes()
	ts := httptest.NewServer(router)
	defer ts.Close()

	// test copy feedback for Link type
	req, err := http.NewRequest("POST", ts.URL+"/copy-feedback", strings.NewReader("type=Link"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	// header is set after render, so we can't check it here
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "<strong>Link copied!</strong>")
	assert.Contains(t, string(body), "Share this link to access your secret content")

	// test copy feedback for Message type
	req, err = http.NewRequest("POST", ts.URL+"/copy-feedback", strings.NewReader("type=Message"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "<strong>Message copied!</strong>")
	assert.NotContains(t, string(body), "Share this link")

	// test copy feedback with invalid type (should default to Content)
	req, err = http.NewRequest("POST", ts.URL+"/copy-feedback", strings.NewReader("type=Invalid"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "<strong>Content copied!</strong>")
}

func prepTestServer(t *testing.T) (ts *httptest.Server, teardown func()) {
	eng := store.NewInMemory(time.Second)

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:         []string{"example.com"},
			Protocol:       "https",
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Safe Secrets",
		})

	require.NoError(t, err)

	ts = httptest.NewServer(srv.routes())
	return ts, ts.Close
}

func TestServer_ping(t *testing.T) {
	ts, teardown := prepTestServer(t)
	defer teardown()

	client := http.Client{Timeout: time.Second}

	tests := []struct {
		name string
		path string
	}{
		{"root ping", "/ping"},
		{"api ping", "/api/v1/ping"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Get(ts.URL + tc.path)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, "pong", string(body))
		})
	}
}

func TestServer_sitemap(t *testing.T) {
	ts, teardown := prepTestServer(t)
	defer teardown()

	client := http.Client{Timeout: time.Second}

	resp, err := client.Get(ts.URL + "/sitemap.xml")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/xml; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)

	// check XML structure
	assert.Contains(t, bodyStr, `<?xml version="1.0" encoding="UTF-8"?>`)
	assert.Contains(t, bodyStr, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)

	// check home page entry
	assert.Contains(t, bodyStr, "<loc>https://example.com/</loc>")
	assert.Contains(t, bodyStr, "<changefreq>weekly</changefreq>")
	assert.Contains(t, bodyStr, "<priority>1.0</priority>")

	// check about page entry
	assert.Contains(t, bodyStr, "<loc>https://example.com/about</loc>")
	assert.Contains(t, bodyStr, "<changefreq>monthly</changefreq>")
	assert.Contains(t, bodyStr, "<priority>0.8</priority>")

	// check lastmod format (should be YYYY-MM-DD)
	assert.Regexp(t, `<lastmod>\d{4}-\d{2}-\d{2}</lastmod>`, bodyStr)

	// ensure no message URLs are included
	assert.NotContains(t, bodyStr, "/message/")
	assert.NotContains(t, bodyStr, "/api/")
}

func TestServer_robotsTxt(t *testing.T) {
	ts, teardown := prepTestServer(t)
	defer teardown()

	client := http.Client{Timeout: time.Second}

	resp, err := client.Get(ts.URL + "/robots.txt")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// check all expected directives
	assert.Contains(t, bodyStr, "User-agent: *")
	assert.Contains(t, bodyStr, "Disallow: /api/")
	assert.Contains(t, bodyStr, "Disallow: /message/")
	assert.Contains(t, bodyStr, "Sitemap: https://example.com/sitemap.xml")
}

func TestServer_NewWithNoDomains(t *testing.T) {
	eng := store.NewInMemory(time.Second)

	_, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:         []string{}, // no domains
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Safe Secrets",
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one domain must be configured")
}

func TestServer_NewWithMultipleDomains(t *testing.T) {
	eng := store.NewInMemory(time.Second)

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:         []string{"example.com", "alt.example.com", "backup.example.com"},
			Protocol:       "https",
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Safe Secrets",
		})

	require.NoError(t, err)
	assert.Len(t, srv.cfg.Domain, 3)
	assert.Equal(t, "example.com", srv.cfg.Domain[0])
	assert.Equal(t, "alt.example.com", srv.cfg.Domain[1])
	assert.Equal(t, "backup.example.com", srv.cfg.Domain[2])
}

func TestServer_MultipleDomainsLinkGeneration(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:         []string{"primary.com", "secondary.com"},
			Protocol:       "https",
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Safe Secrets",
		})
	require.NoError(t, err)

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	// save message
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/message", strings.NewReader(`{"message": "test message","exp": 600,"pin": "12345"}`))
	require.NoError(t, err)
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 201, resp.StatusCode)

	respSave := struct {
		Key string
		Exp time.Time
	}{}
	err = json.NewDecoder(resp.Body).Decode(&respSave)
	require.NoError(t, err)

	// verify messages can be retrieved with the key regardless of which domain was used for creation
	url := fmt.Sprintf("%s/api/v1/message/%s/12345", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, http.NoBody)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	respLoad := struct {
		Key     string
		Message string
	}{}
	err = json.NewDecoder(resp.Body).Decode(&respLoad)
	require.NoError(t, err)
	assert.Equal(t, "test message", respLoad.Message)
}

func TestServer_saveFileCtrl(t *testing.T) {
	ts, teardown := prepTestServerWithFiles(t)
	defer teardown()

	client := http.Client{Timeout: 5 * time.Second}

	t.Run("valid file upload", func(t *testing.T) {
		body, contentType := createMultipartFile(t, "file", "test.txt", "text/plain", []byte("secret file content"), "12345", "600")
		req, err := http.NewRequest("POST", ts.URL+"/api/v1/file", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", contentType)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 201, resp.StatusCode)
		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]any
		err = json.Unmarshal(respBody, &result)
		require.NoError(t, err)
		assert.NotEmpty(t, result["key"])
		assert.NotEmpty(t, result["exp"])
	})

	t.Run("missing file", func(t *testing.T) {
		body, contentType := createMultipartFile(t, "", "", "", nil, "12345", "600")
		req, err := http.NewRequest("POST", ts.URL+"/api/v1/file", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", contentType)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("wrong pin size", func(t *testing.T) {
		body, contentType := createMultipartFile(t, "file", "test.txt", "text/plain", []byte("content"), "123", "600")
		req, err := http.NewRequest("POST", ts.URL+"/api/v1/file", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", contentType)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("exp exceeds max", func(t *testing.T) {
		body, contentType := createMultipartFile(t, "file", "test.txt", "text/plain", []byte("content"), "12345", "999999")
		req, err := http.NewRequest("POST", ts.URL+"/api/v1/file", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", contentType)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestServer_getFileCtrl(t *testing.T) {
	ts, teardown := prepTestServerWithFiles(t)
	defer teardown()

	client := http.Client{Timeout: 5 * time.Second}

	// upload a file first
	fileContent := []byte("secret file content for download")
	body, contentType := createMultipartFile(t, "file", "download-test.txt", "text/plain", fileContent, "12345", "600")
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/file", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 201, resp.StatusCode)

	var saveResp struct {
		Key string
		Exp time.Time
	}
	err = json.NewDecoder(resp.Body).Decode(&saveResp)
	require.NoError(t, err)

	t.Run("valid download", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/file/%s/12345", ts.URL, saveResp.Key)
		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "download-test.txt")

		downloadedContent, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, fileContent, downloadedContent)
	})

	t.Run("second download fails (one-time access)", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/file/%s/12345", ts.URL, saveResp.Key)
		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("wrong pin", func(t *testing.T) {
		// upload another file
		body, contentType := createMultipartFile(t, "file", "test2.txt", "text/plain", []byte("content"), "12345", "600")
		req, err := http.NewRequest("POST", ts.URL+"/api/v1/file", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", contentType)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, 201, resp.StatusCode)

		var resp2 struct{ Key string }
		err = json.NewDecoder(resp.Body).Decode(&resp2)
		require.NoError(t, err)

		url := fmt.Sprintf("%s/api/v1/file/%s/00000", ts.URL, resp2.Key)
		getResp, err := client.Get(url)
		require.NoError(t, err)
		defer getResp.Body.Close()
		assert.Equal(t, 417, getResp.StatusCode) // wrong pin attempt
	})

	t.Run("non-existent key", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/file/badkey/12345", ts.URL)
		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestServer_fileEndpointsDisabled(t *testing.T) {
	// test that file endpoints return 404 when EnableFiles is false
	ts, teardown := prepTestServer(t)
	defer teardown()

	client := http.Client{Timeout: time.Second}

	t.Run("POST /api/v1/file disabled", func(t *testing.T) {
		body, contentType := createMultipartFile(t, "file", "test.txt", "text/plain", []byte("content"), "12345", "600")
		req, err := http.NewRequest("POST", ts.URL+"/api/v1/file", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", contentType)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 404, resp.StatusCode)
	})

	t.Run("GET /api/v1/file disabled", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/v1/file/somekey/12345")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 404, resp.StatusCode)
	})
}

func TestServer_getParamsWithFiles(t *testing.T) {
	ts, teardown := prepTestServerWithFiles(t)
	defer teardown()

	client := http.Client{Timeout: time.Second}
	resp, err := client.Get(ts.URL + "/api/v1/params")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"pin_size":5,"max_pin_attempts":3,"max_exp_sec":36000,"files_enabled":true,"max_file_size":1048576}`+"\n", string(body))
}

func prepTestServerWithFiles(t *testing.T) (ts *httptest.Server, teardown func()) {
	eng := store.NewInMemory(time.Second * 30)

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
			MaxFileSize:    1024 * 1024,
		}),
		"1",
		Config{
			Domain:         []string{"example.com"},
			Protocol:       "https",
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Safe Secrets",
			EnableFiles:    true,
			MaxFileSize:    1024 * 1024,
		})

	require.NoError(t, err)

	ts = httptest.NewServer(srv.routes())
	return ts, ts.Close
}

func createMultipartFile(t *testing.T, fieldName, fileName, contentType string, content []byte, pin, exp string) (body *strings.Reader, ct string) {
	t.Helper()
	var b strings.Builder
	boundary := "----TestBoundary7MA4YWxkTrZu0gW"

	if fieldName != "" && content != nil {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=%q; filename=%q\r\n", fieldName, fileName))
		b.WriteString(fmt.Sprintf("Content-Type: %s\r\n\r\n", contentType))
		b.Write(content)
		b.WriteString("\r\n")
	}

	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Disposition: form-data; name=\"pin\"\r\n\r\n")
	b.WriteString(pin + "\r\n")

	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Disposition: form-data; name=\"exp\"\r\n\r\n")
	b.WriteString(exp + "\r\n")

	b.WriteString("--" + boundary + "--\r\n")

	return strings.NewReader(b.String()), "multipart/form-data; boundary=" + boundary
}

func TestServer_saveFileCtrl_LargeFile(t *testing.T) {
	// test that files larger than 64KB (global default limit) can be uploaded when files are enabled
	ts, teardown := prepTestServerWithFiles(t)
	defer teardown()

	client := http.Client{Timeout: 10 * time.Second}

	// create a file larger than 64KB but smaller than 1MB max
	largeContent := make([]byte, 100*1024) // 100KB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	body, contentType := createMultipartFile(t, "file", "large-file.bin", "application/octet-stream", largeContent, "12345", "600")
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/file", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 201, resp.StatusCode, "large files (>64KB) should be accepted when files are enabled")
	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]any
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)
	assert.NotEmpty(t, result["key"])
}

func TestServer_getFileCtrl_SpecialFilename(t *testing.T) {
	// test that files with special characters in filename are handled correctly
	ts, teardown := prepTestServerWithFiles(t)
	defer teardown()

	client := http.Client{Timeout: 5 * time.Second}

	// upload a file with special characters in name
	fileContent := []byte("test content")
	specialName := "file with spaces & special.txt"
	body, contentType := createMultipartFile(t, "file", specialName, "text/plain", fileContent, "12345", "600")
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/file", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, 201, resp.StatusCode)

	var saveResp struct{ Key string }
	err = json.NewDecoder(resp.Body).Decode(&saveResp)
	require.NoError(t, err)

	// download and check headers
	url := fmt.Sprintf("%s/api/v1/file/%s/12345", ts.URL, saveResp.Key)
	downloadResp, err := client.Get(url)
	require.NoError(t, err)
	defer downloadResp.Body.Close()

	assert.Equal(t, 200, downloadResp.StatusCode)
	assert.Equal(t, "nosniff", downloadResp.Header.Get("X-Content-Type-Options"), "should have X-Content-Type-Options header")
	assert.Contains(t, downloadResp.Header.Get("Content-Disposition"), "attachment", "should be attachment")
	assert.Contains(t, downloadResp.Header.Get("Content-Disposition"), specialName, "should contain filename")
}

func TestServer_saveFileCtrl_InvalidExpiration(t *testing.T) {
	ts, teardown := prepTestServerWithFiles(t)
	defer teardown()

	client := http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		name   string
		exp    string
		status int
	}{
		{name: "zero expiration", exp: "0", status: 400},
		{name: "negative expiration", exp: "-100", status: 400},
		{name: "non-numeric expiration", exp: "abc", status: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType := createMultipartFile(t, "file", "test.txt", "text/plain", []byte("content"), "12345", tt.exp)
			req, err := http.NewRequest("POST", ts.URL+"/api/v1/file", body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", contentType)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}

func TestServer_getFileCtrl_TextMessageViaFileEndpoint(t *testing.T) {
	// test that accessing a text message via file endpoint returns error
	ts, teardown := prepTestServerWithFiles(t)
	defer teardown()

	client := http.Client{Timeout: 5 * time.Second}

	// create a text message (not a file)
	saveBody := `{"message":"secret text","pin":"12345","exp":600}`
	saveResp, err := client.Post(ts.URL+"/api/v1/message", "application/json", strings.NewReader(saveBody))
	require.NoError(t, err)
	defer saveResp.Body.Close()
	require.Equal(t, 201, saveResp.StatusCode)

	var result struct{ Key string }
	err = json.NewDecoder(saveResp.Body).Decode(&result)
	require.NoError(t, err)

	// try to access via file endpoint
	url := fmt.Sprintf("%s/api/v1/file/%s/12345", ts.URL, result.Key)
	getResp, err := client.Get(url)
	require.NoError(t, err)
	defer getResp.Body.Close()

	assert.Equal(t, 400, getResp.StatusCode, "should reject text message via file endpoint")
}
