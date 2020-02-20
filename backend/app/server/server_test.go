package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/secrets/backend/app/messager"
	"github.com/umputun/secrets/backend/app/store"
)

func TestServer_saveAndLoad(t *testing.T) {
	ts, teardown := prepTestServer()
	defer teardown()

	// save message
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/message", strings.NewReader(`{"message": "my secret message","exp": 600,"pin": "12345"}`))
	require.NoError(t, err)
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
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
	req, err = http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
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
	assert.Equal(t, 400, resp.StatusCode)
}

func TestServer_saveAndManyPinAttempt(t *testing.T) {
	ts, teardown := prepTestServer()
	defer teardown()

	// save message
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/message", strings.NewReader(`{"message": "my secret message","exp": 600,"pin": "12345"}`))
	require.NoError(t, err)
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	respSave := struct {
		Key string
		Exp time.Time
	}{}
	err = json.NewDecoder(resp.Body).Decode(&respSave)
	assert.NoError(t, err)
	t.Logf("%+v", respSave)

	// load saved message
	url := fmt.Sprintf("%s/api/v1/message/%s/xxxxxx", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	// try with valid pin will fail, too many attempt
	url = fmt.Sprintf("%s/api/v1/message/%s/12345", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, nil)
	require.NoError(t, err)
}

func TestServer_saveAndGoodPinAttempt(t *testing.T) {
	ts, teardown := prepTestServer()
	defer teardown()

	// save message
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/message", strings.NewReader(`{"message": "my secret message","exp": 600,"pin": "12345"}`))
	require.NoError(t, err)
	client := http.Client{Timeout: time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)

	respSave := struct {
		Key string
		Exp time.Time
	}{}
	err = json.NewDecoder(resp.Body).Decode(&respSave)
	assert.NoError(t, err)
	t.Logf("%+v", respSave)

	// load saved message
	url := fmt.Sprintf("%s/api/v1/message/%s/xxxxxx", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	// try with a valid pin will pass, not too many attempt
	url = fmt.Sprintf("%s/api/v1/message/%s/12345", ts.URL, respSave.Key)
	req, err = http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestServer_getParams(t *testing.T) {
	ts, teardown := prepTestServer()
	defer teardown()

	client := http.Client{Timeout: time.Second}
	url := fmt.Sprintf("%s/api/v1/params", ts.URL)
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"pin_size":5,"max_pin_attempts":3,"max_exp_sec":36000}`+"\n", string(body))
}

func prepTestServer() (*httptest.Server, func()) {
	eng := store.NewInMemory(time.Second)
	srv := Server{
		Messager: messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		PinSize:        5,
		MaxPinAttempts: 3,
		MaxExpire:      10 * time.Hour,
		Version:        "1",
	}
	ts := httptest.NewServer(srv.routes())
	return ts, ts.Close
}
