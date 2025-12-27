package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/umputun/secrets/v2/app/email"
	"github.com/umputun/secrets/v2/app/messager"
	"github.com/umputun/secrets/v2/app/server/mocks"
	"github.com/umputun/secrets/v2/app/store"
)

// helper to generate bcrypt hash for testing
func testBcryptHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	return string(hash)
}

func TestServer_isAuthenticated(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	hash := testBcryptHash(t, "secret123")

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:     []string{"example.com"},
			PinSize:    5,
			MaxExpire:  10 * time.Hour,
			AuthHash:   hash,
			SessionTTL: time.Hour,
		},
	)
	require.NoError(t, err)

	t.Run("no cookie returns false", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		assert.False(t, srv.isAuthenticated(req))
	})

	t.Run("invalid cookie returns false", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.AddCookie(&http.Cookie{Name: authCookieName, Value: "invalid-token"})
		assert.False(t, srv.isAuthenticated(req))
	})

	t.Run("valid cookie returns true", func(t *testing.T) {
		token := srv.generateSessionToken()
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
		assert.True(t, srv.isAuthenticated(req))
	})

	t.Run("expired token returns false", func(t *testing.T) {
		// create server with very short TTL
		shortSrv, err := New(
			messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
				MaxDuration:    10 * time.Hour,
				MaxPinAttempts: 3,
			}),
			"1",
			Config{
				Domain:     []string{"example.com"},
				PinSize:    5,
				MaxExpire:  10 * time.Hour,
				AuthHash:   hash,
				SessionTTL: time.Millisecond,
			},
		)
		require.NoError(t, err)

		token := shortSrv.generateSessionToken()
		time.Sleep(10 * time.Millisecond) // wait for expiration

		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
		assert.False(t, shortSrv.isAuthenticated(req))
	})
}

func TestServer_checkBasicAuth(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	hash := testBcryptHash(t, "secret123")

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:     []string{"example.com"},
			PinSize:    5,
			MaxExpire:  10 * time.Hour,
			AuthHash:   hash,
			SessionTTL: time.Hour,
		},
	)
	require.NoError(t, err)

	t.Run("no auth header returns false", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		assert.False(t, srv.checkBasicAuth(req))
	})

	t.Run("wrong username returns false", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.SetBasicAuth("wronguser", "secret123")
		assert.False(t, srv.checkBasicAuth(req))
	})

	t.Run("wrong password returns false", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.SetBasicAuth("secrets", "wrongpassword")
		assert.False(t, srv.checkBasicAuth(req))
	})

	t.Run("correct credentials return true", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.SetBasicAuth("secrets", "secret123")
		assert.True(t, srv.checkBasicAuth(req))
	})
}

func TestServer_loginCtrl(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	hash := testBcryptHash(t, "secret123")

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:     []string{"example.com"},
			PinSize:    5,
			MaxExpire:  10 * time.Hour,
			AuthHash:   hash,
			SessionTTL: time.Hour,
			Protocol:   "https",
		},
	)
	require.NoError(t, err)

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	t.Run("valid password sets cookie and triggers form resubmit", func(t *testing.T) {
		form := url.Values{}
		form.Set("password", "secret123")

		req, err := http.NewRequest(http.MethodPost, ts.URL+"/login", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // don't follow redirects
		}}

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "submitSecretForm", resp.Header.Get("HX-Trigger"))

		// check cookie is set
		var authCookie *http.Cookie
		for _, c := range resp.Cookies() {
			if c.Name == authCookieName {
				authCookie = c
				break
			}
		}
		require.NotNil(t, authCookie)
		assert.True(t, authCookie.HttpOnly)
	})

	t.Run("invalid password returns error popup", func(t *testing.T) {
		form := url.Values{}
		form.Set("password", "wrongpassword")

		req, err := http.NewRequest(http.MethodPost, ts.URL+"/login", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		// should NOT have HX-Trigger header (form resubmit)
		assert.Empty(t, resp.Header.Get("HX-Trigger"))
	})
}

func TestServer_logoutCtrl(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	hash := testBcryptHash(t, "secret123")

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:     []string{"example.com"},
			PinSize:    5,
			MaxExpire:  10 * time.Hour,
			AuthHash:   hash,
			SessionTTL: time.Hour,
		},
	)
	require.NoError(t, err)

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // don't follow redirects
	}}

	resp, err := client.Get(ts.URL + "/logout")
	require.NoError(t, err)
	defer resp.Body.Close()

	// should redirect to home
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Equal(t, "/", resp.Header.Get("Location"))

	// should clear cookie (MaxAge=-1)
	var authCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == authCookieName {
			authCookie = c
			break
		}
	}
	require.NotNil(t, authCookie)
	assert.Equal(t, -1, authCookie.MaxAge)
}

func TestServer_generateLinkCtrl_WithAuth(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	hash := testBcryptHash(t, "secret123")

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:     []string{"example.com"},
			PinSize:    5,
			MaxExpire:  10 * time.Hour,
			AuthHash:   hash,
			SessionTTL: time.Hour,
		},
	)
	require.NoError(t, err)

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	t.Run("unauthenticated request returns 401 with login popup", func(t *testing.T) {
		form := url.Values{}
		form.Set("message", "test message")
		form.Set("pin", "12345")
		form.Set("exp", "10")
		form.Set("expUnit", "m")

		req, err := http.NewRequest(http.MethodPost, ts.URL+"/generate-link", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("authenticated request succeeds", func(t *testing.T) {
		// first login to get session cookie
		loginForm := url.Values{}
		loginForm.Set("password", "secret123")

		loginReq, err := http.NewRequest(http.MethodPost, ts.URL+"/login", strings.NewReader(loginForm.Encode()))
		require.NoError(t, err)
		loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}
		loginResp, err := client.Do(loginReq)
		require.NoError(t, err)
		defer loginResp.Body.Close()

		// get the session cookie
		var sessionCookie *http.Cookie
		for _, c := range loginResp.Cookies() {
			if c.Name == authCookieName {
				sessionCookie = c
				break
			}
		}
		require.NotNil(t, sessionCookie)

		// now make authenticated request
		form := url.Values{}
		form.Set("message", "test message")
		form.Set("pin", "12345")
		form.Set("exp", "10")
		form.Set("expUnit", "m")

		req, err := http.NewRequest(http.MethodPost, ts.URL+"/generate-link", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(sessionCookie)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestServer_saveMessageCtrl_WithAuth(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	hash := testBcryptHash(t, "secret123")

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:     []string{"example.com"},
			PinSize:    5,
			MaxExpire:  10 * time.Hour,
			AuthHash:   hash,
			SessionTTL: time.Hour,
		},
	)
	require.NoError(t, err)

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	t.Run("API without basic auth returns 401", func(t *testing.T) {
		body := `{"Message": "test", "Exp": 600, "Pin": "12345"}`
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/message", strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("WWW-Authenticate"), `Basic realm="secrets"`)
	})

	t.Run("API with wrong basic auth returns 401", func(t *testing.T) {
		body := `{"Message": "test", "Exp": 600, "Pin": "12345"}`
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/message", strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth("secrets", "wrongpassword")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("API with correct basic auth succeeds", func(t *testing.T) {
		body := `{"Message": "test", "Exp": 600, "Pin": "12345"}`
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/message", strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth("secrets", "secret123")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}

func TestServer_NoAuthWhenDisabled(t *testing.T) {
	eng := store.NewInMemory(time.Second)

	// no AuthHash = auth disabled
	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:    []string{"example.com"},
			PinSize:   5,
			MaxExpire: 10 * time.Hour,
			AuthHash:  "", // auth disabled
		},
	)
	require.NoError(t, err)

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	t.Run("generate-link works without auth when disabled", func(t *testing.T) {
		form := url.Values{}
		form.Set("message", "test message")
		form.Set("pin", "12345")
		form.Set("exp", "10")
		form.Set("expUnit", "m")

		req, err := http.NewRequest(http.MethodPost, ts.URL+"/generate-link", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("API works without auth when disabled", func(t *testing.T) {
		body := `{"Message": "test", "Exp": 600, "Pin": "12345"}`
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/message", strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("auth routes not registered when disabled", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/login-popup")
		require.NoError(t, err)
		defer resp.Body.Close()

		// should be 404 since route is not registered
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestServer_generateSessionToken(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	hash := testBcryptHash(t, "secret123")

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:     []string{"example.com"},
			PinSize:    5,
			MaxExpire:  10 * time.Hour,
			AuthHash:   hash,
			SessionTTL: time.Hour,
		},
	)
	require.NoError(t, err)

	t.Run("generates valid token format", func(t *testing.T) {
		token := srv.generateSessionToken()
		parts := strings.Split(token, ".")
		assert.Len(t, parts, 3, "token should have 3 parts: uuid.timestamp.signature")
	})

	t.Run("tokens are unique", func(t *testing.T) {
		token1 := srv.generateSessionToken()
		token2 := srv.generateSessionToken()
		assert.NotEqual(t, token1, token2)
	})
}

func TestServer_validateSessionToken(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	hash := testBcryptHash(t, "secret123")

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:     []string{"example.com"},
			PinSize:    5,
			MaxExpire:  10 * time.Hour,
			AuthHash:   hash,
			SessionTTL: time.Hour,
		},
	)
	require.NoError(t, err)

	t.Run("valid token is accepted", func(t *testing.T) {
		token := srv.generateSessionToken()
		assert.True(t, srv.validateSessionToken(token))
	})

	t.Run("empty token is rejected", func(t *testing.T) {
		assert.False(t, srv.validateSessionToken(""))
	})

	t.Run("malformed token is rejected", func(t *testing.T) {
		assert.False(t, srv.validateSessionToken("not-a-valid-token"))
		assert.False(t, srv.validateSessionToken("only.two"))
		assert.False(t, srv.validateSessionToken("uuid.invalid-timestamp.sig"))
		assert.False(t, srv.validateSessionToken("uuid.123.!!!invalid-base64!!!"))
	})

	t.Run("tampered token is rejected", func(t *testing.T) {
		token := srv.generateSessionToken()
		// tamper with the signature
		parts := strings.Split(token, ".")
		parts[2] = "tamperedsignature"
		tamperedToken := strings.Join(parts, ".")
		assert.False(t, srv.validateSessionToken(tamperedToken))
	})

	t.Run("token from different secret is rejected", func(t *testing.T) {
		differentSrv, err := New(
			messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
				MaxDuration:    10 * time.Hour,
				MaxPinAttempts: 3,
			}),
			"1",
			Config{
				Domain:     []string{"example.com"},
				PinSize:    5,
				MaxExpire:  10 * time.Hour,
				AuthHash:   testBcryptHash(t, "different-password"), // different secret
				SessionTTL: time.Hour,
			},
		)
		require.NoError(t, err)

		token := differentSrv.generateSessionToken()
		assert.False(t, srv.validateSessionToken(token))
	})
}

func TestServer_loginPopupCtrl(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	hash := testBcryptHash(t, "secret123")

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:     []string{"example.com"},
			PinSize:    5,
			MaxExpire:  10 * time.Hour,
			AuthHash:   hash,
			SessionTTL: time.Hour,
		},
	)
	require.NoError(t, err)

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	t.Run("renders login popup", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/login-popup")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
	})
}

func TestServer_EmailEndpointsRequireAuth(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	hash := testBcryptHash(t, "secret123")

	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"1",
		Config{
			Domain:       []string{"example.com"},
			Protocol:     "https",
			PinSize:      5,
			MaxExpire:    10 * time.Hour,
			AuthHash:     hash,
			SessionTTL:   time.Hour,
			EmailEnabled: true,
		},
	)
	require.NoError(t, err)

	// add mock email sender
	mock := &mocks.EmailSenderMock{
		GetDefaultFromNameFunc: func() string { return "Test Sender" },
		SendFunc:               func(ctx context.Context, req email.Request) error { return nil },
	}
	srv = srv.WithEmail(mock)

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	t.Run("unauthenticated request to email-popup returns 401", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/email-popup?link=https://example.com/message/123", http.NoBody)
		require.NoError(t, err)
		req.Header.Set("HX-Request", "true")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("unauthenticated request to send-email returns 401", func(t *testing.T) {
		form := url.Values{}
		form.Set("link", "https://example.com/message/123")
		form.Set("to", "test@example.com")
		form.Set("subject", "Test Subject")

		req, err := http.NewRequest(http.MethodPost, ts.URL+"/send-email", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("authenticated request to email-popup succeeds", func(t *testing.T) {
		// login first
		loginForm := url.Values{}
		loginForm.Set("password", "secret123")
		loginReq, err := http.NewRequest(http.MethodPost, ts.URL+"/login", strings.NewReader(loginForm.Encode()))
		require.NoError(t, err)
		loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}
		loginResp, err := client.Do(loginReq)
		require.NoError(t, err)
		defer loginResp.Body.Close()

		// get the session cookie
		var sessionCookie *http.Cookie
		for _, c := range loginResp.Cookies() {
			if c.Name == authCookieName {
				sessionCookie = c
				break
			}
		}
		require.NotNil(t, sessionCookie, "session cookie should be set after login")

		// now request email popup with auth cookie
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/email-popup?link=https://example.com/message/123", http.NoBody)
		require.NoError(t, err)
		req.AddCookie(sessionCookie)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("authenticated request to send-email succeeds", func(t *testing.T) {
		// login first
		loginForm := url.Values{}
		loginForm.Set("password", "secret123")
		loginReq, err := http.NewRequest(http.MethodPost, ts.URL+"/login", strings.NewReader(loginForm.Encode()))
		require.NoError(t, err)
		loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}
		loginResp, err := client.Do(loginReq)
		require.NoError(t, err)
		defer loginResp.Body.Close()

		// get the session cookie
		var sessionCookie *http.Cookie
		for _, c := range loginResp.Cookies() {
			if c.Name == authCookieName {
				sessionCookie = c
				break
			}
		}
		require.NotNil(t, sessionCookie, "session cookie should be set after login")

		// send email with auth cookie
		form := url.Values{}
		form.Set("link", "https://example.com/message/123")
		form.Set("to", "test@example.com")
		form.Set("subject", "Test Subject")

		req, err := http.NewRequest(http.MethodPost, ts.URL+"/send-email", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(sessionCookie)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
