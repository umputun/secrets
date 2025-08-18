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

	"github.com/go-chi/chi/v5"
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
		})

	assert.NoError(t, err)

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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
		})
	require.NoError(t, err)

	tests := []struct {
		name               string
		body               string
		expectedStatusCode int
		checkResponse      func(t *testing.T, body []byte)
	}{
		{name: "valid message", body: `{"message": "secret", "exp": 600, "pin": "12345"}`, expectedStatusCode: 201, checkResponse: func(t *testing.T, body []byte) {
			var resp map[string]interface{}
			err := json.Unmarshal(body, &resp)
			assert.NoError(t, err)
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
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
		})
	require.NoError(t, err)

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
			var resp map[string]interface{}
			err := json.Unmarshal(body, &resp)
			assert.NoError(t, err)
			assert.Equal(t, "test secret", resp["message"])
		}},
		{name: "invalid pin returns 400 when key not found", key: msg.Key, pin: "99999", expectedStatusCode: 400},
		{name: "non-existent key", key: "badkey", pin: "12345", expectedStatusCode: 400},
		{name: "empty key", key: "", pin: "12345", expectedStatusCode: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/message/%s/%s", tt.key, tt.pin)
			if tt.key == "" {
				url = fmt.Sprintf("/api/v1/message//%s", tt.pin)
			}
			req := httptest.NewRequest(http.MethodGet, url, http.NoBody)
			rr := httptest.NewRecorder()

			// add chi context
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("key", tt.key)
			rctx.URLParams.Add("pin", tt.pin)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			srv.getMessageCtrl(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rr.Body.Bytes())
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
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
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
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("server didn't stop in time")
	}
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
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
		})

	assert.NoError(t, err)

	ts = httptest.NewServer(srv.routes())
	return ts, ts.Close
}
