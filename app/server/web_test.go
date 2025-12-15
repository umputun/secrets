package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/secrets/app/messager"
	"github.com/umputun/secrets/app/store"
)

func TestTemplates_Duration(t *testing.T) {
	tests := []struct {
		name string
		unit string
		v    int
		want time.Duration
	}{
		{name: "minutes", unit: "m", v: 5, want: time.Duration(5) * time.Minute},
		{name: "hours", unit: "h", v: 5, want: time.Duration(5) * time.Hour},
		{name: "days", unit: "d", v: 5, want: time.Duration(5*24) * time.Hour},
		{name: "bad unit", unit: "x", v: 5, want: time.Duration(0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := duration(tt.v, tt.unit)
			t.Logf("%+v", r)

			assert.Equal(t, tt.want, r)
		})
	}
}

func TestTemplates_HumanDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{name: "seconds", d: time.Duration(5) * time.Second, want: "5 seconds"},
		{name: "minutes", d: time.Duration(5) * time.Minute, want: "5 minutes"},
		{name: "hours", d: time.Duration(5) * time.Hour, want: "5 hours"},
		{name: "days", d: time.Duration(5*24) * time.Hour, want: "5 days"},
		{name: "1 second", d: time.Duration(1) * time.Second, want: "1 second"},
		{name: "1 minute", d: time.Duration(1) * time.Minute, want: "1 minute"},
		{name: "1 hour", d: time.Duration(1) * time.Hour, want: "1 hour"},
		{name: "1 day", d: time.Duration(1*24) * time.Hour, want: "1 day"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := humanDuration(tt.d)
			t.Logf("%+v", r)

			assert.Equal(t, tt.want, r)
		})
	}
}

func TestTemplates_NewTemplateCache(t *testing.T) {
	cache, err := newTemplateCache()

	require.NoError(t, err)

	assert.Len(t, cache, 9)
	assert.NotNil(t, cache["404.tmpl.html"])
	assert.NotNil(t, cache["about.tmpl.html"])
	assert.NotNil(t, cache["home.tmpl.html"])
	assert.NotNil(t, cache["show-message.tmpl.html"])
	assert.NotNil(t, cache["decoded-message.tmpl.html"])
	assert.NotNil(t, cache["error.tmpl.html"])
	assert.NotNil(t, cache["secure-link.tmpl.html"])
	assert.NotNil(t, cache["popup.tmpl.html"])
	assert.NotNil(t, cache["copy-button.tmpl.html"])
}

func TestServer_indexCtrl(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	srv.indexCtrl(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Safe Secrets")
	assert.Contains(t, rr.Body.String(), "Generate Secure Link")
}

func TestServer_aboutViewCtrl(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/about", http.NoBody)
	rr := httptest.NewRecorder()

	srv.aboutViewCtrl(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "How it works")
}

func TestServer_showMessageViewCtrl(t *testing.T) {
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

	resp, err := http.Get(ts.URL + "/message/testkey123")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := make([]byte, 1024*10) // 10KB buffer
	n, _ := resp.Body.Read(body)
	responseBody := string(body[:n])
	assert.Contains(t, responseBody, "testkey123")
}

func TestServer_generateLinkCtrl(t *testing.T) {
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
			Protocol:       "https",
			Domain:         []string{"example.com", "alt.example.com"},
		})
	require.NoError(t, err)

	tests := []struct {
		name           string
		formData       url.Values
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name: "valid request",
			formData: url.Values{
				"message": {"secret message"},
				"exp":     {"15"},
				"expUnit": {"m"},
				"pin":     {"1", "2", "3", "4", "5"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "https://example.com/message/")
			},
		},
		{
			name: "empty message",
			formData: url.Values{
				"message": {""},
				"exp":     {"15"},
				"expUnit": {"m"},
				"pin":     {"1", "2", "3", "4", "5"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				// form validation error will be shown in the form, look for the error class/input
				assert.Contains(t, body, "value=\"15\"") // verify it returns the form
			},
		},
		{
			name: "invalid pin",
			formData: url.Values{
				"message": {"secret message"},
				"exp":     {"15"},
				"expUnit": {"m"},
				"pin":     {"1", "2", "", "4", "5"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Pin must be 5 digits long without empty values")
			},
		},
		{
			name: "exceed max duration",
			formData: url.Values{
				"message": {"secret message"},
				"exp":     {"100"},
				"expUnit": {"d"},
				"pin":     {"1", "2", "3", "4", "5"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Expire must be less than")
			},
		},
		{
			name: "invalid exp unit",
			formData: url.Values{
				"message": {"secret message"},
				"exp":     {"15"},
				"expUnit": {"x"},
				"pin":     {"1", "2", "3", "4", "5"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Only Minutes, Hours and Days are allowed")
			},
		},
		{
			name: "non-numeric exp",
			formData: url.Values{
				"message": {"secret message"},
				"exp":     {"abc"},
				"expUnit": {"m"},
				"pin":     {"1", "2", "3", "4", "5"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Expire must be a number")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/generate-link", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()

			srv.generateLinkCtrl(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rr.Body.String())
			}
		})
	}
}

func TestServer_generateLinkCtrl_HTMX(t *testing.T) {
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
			Protocol:       "https",
			Domain:         []string{"example.com"},
		})
	require.NoError(t, err)

	t.Run("htmx request with validation error returns 400", func(t *testing.T) {
		formData := url.Values{
			"message": {""},
			"exp":     {"15"},
			"expUnit": {"m"},
			"pin":     {"12345"},
		}

		req := httptest.NewRequest(http.MethodPost, "/generate-link", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true") // HTMX request
		rr := httptest.NewRecorder()

		srv.generateLinkCtrl(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)                 // should return 400 for HTMX
		assert.Contains(t, rr.Body.String(), "Create a Secure Message") // returns the form
		assert.Contains(t, rr.Body.String(), "value=\"15\"")            // with preserved values
	})

	t.Run("regular request with validation error returns 200", func(t *testing.T) {
		formData := url.Values{
			"message": {""},
			"exp":     {"15"},
			"expUnit": {"m"},
			"pin":     {"12345"},
		}

		req := httptest.NewRequest(http.MethodPost, "/generate-link", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// no HX-Request header
		rr := httptest.NewRecorder()

		srv.generateLinkCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)                         // should return 200 for regular request
		assert.Contains(t, rr.Body.String(), "Create a Secure Message") // returns the form
		assert.Contains(t, rr.Body.String(), "value=\"15\"")            // with preserved values
	})

	t.Run("htmx request with valid data returns partial", func(t *testing.T) {
		formData := url.Values{
			"message": {"secret message"},
			"exp":     {"15"},
			"expUnit": {"m"},
			"pin":     {"12345"},
		}

		req := httptest.NewRequest(http.MethodPost, "/generate-link", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		rr := httptest.NewRecorder()

		srv.generateLinkCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Secure Link Generated")
		assert.Contains(t, rr.Body.String(), "https://example.com/message/")
		assert.Contains(t, rr.Body.String(), "id=\"msg-link\"") // verify it's the partial template
	})
}

func TestServer_loadMessageCtrl(t *testing.T) {
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

	// first save a message
	msg, err := srv.messager.MakeMessage(time.Hour, "test secret", "12345")
	require.NoError(t, err)

	tests := []struct {
		name           string
		formData       url.Values
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name: "valid pin",
			formData: url.Values{
				"key": {msg.Key},
				"pin": {"1", "2", "3", "4", "5"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "test secret")
			},
		},
		{
			name: "invalid pin",
			formData: url.Values{
				"key": {msg.Key},
				"pin": {"9", "9", "9", "9", "9"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "error")
			},
		},
		{
			name: "non-existent key",
			formData: url.Values{
				"key": {"nonexistent"},
				"pin": {"1", "2", "3", "4", "5"},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "error")
			},
		},
		{
			name: "empty pin",
			formData: url.Values{
				"key": {msg.Key},
				"pin": {"", "", "", "", ""},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/load-message", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rr := httptest.NewRecorder()

			srv.loadMessageCtrl(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rr.Body.String())
			}
		})
	}
}

func TestServer_render(t *testing.T) {
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

	t.Run("existing template", func(t *testing.T) {
		rr := httptest.NewRecorder()
		data := templateData{
			Form:     createMsgForm{Exp: 15, MaxExp: "10 hours"},
			PinSize:  5,
			Branding: "Safe Secrets",
		}
		srv.render(rr, http.StatusOK, "home.tmpl.html", baseTmpl, data)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Safe Secrets")
	})

	t.Run("non-existing template", func(t *testing.T) {
		rr := httptest.NewRecorder()
		srv.render(rr, http.StatusOK, "nonexistent.tmpl.html", baseTmpl, nil)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "Internal Server Error")
	})

	t.Run("empty template name defaults to base", func(t *testing.T) {
		rr := httptest.NewRecorder()
		data := templateData{
			Form:     createMsgForm{Exp: 15, MaxExp: "10 hours"},
			PinSize:  5,
			Branding: "Safe Secrets",
		}
		srv.render(rr, http.StatusOK, "home.tmpl.html", "", data)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Safe Secrets")
	})
}

func TestServer_until(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected []int
	}{
		{name: "zero", n: 0, expected: []int{}},
		{name: "one", n: 1, expected: []int{0}},
		{name: "five", n: 5, expected: []int{0, 1, 2, 3, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := until(tt.n)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServer_newTemplateData(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"test-v",
		Config{
			Domain:         []string{"example.com"},
			Protocol:       "https",
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Test Brand",
		})
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", http.NoBody)

	t.Run("with form data", func(t *testing.T) {
		form := createMsgForm{Exp: 15, MaxExp: "10 hours"}
		data := srv.newTemplateData(req, form)

		assert.Equal(t, form, data.Form)
		assert.Equal(t, 5, data.PinSize)
		assert.Equal(t, "Test Brand", data.Branding)
		assert.Equal(t, "auto", data.Theme) // default theme
		assert.Equal(t, time.Now().Year(), data.CurrentYear)
		// verify URL field is set correctly
		assert.Equal(t, "https://example.com/", data.URL)
		// verify BaseURL field is set correctly
		assert.Equal(t, "https://example.com", data.BaseURL)
		// verify IsMessagePage is false by default
		assert.False(t, data.IsMessagePage)
		// verify PageTitle and PageDesc are empty (set by individual handlers)
		assert.Empty(t, data.PageTitle)
		assert.Empty(t, data.PageDesc)
	})

	t.Run("with nil form", func(t *testing.T) {
		data := srv.newTemplateData(req, nil)

		assert.Nil(t, data.Form)
		assert.Equal(t, 5, data.PinSize)
		assert.Equal(t, "Test Brand", data.Branding)
		assert.Equal(t, "auto", data.Theme)
	})

	t.Run("with theme cookie", func(t *testing.T) {
		req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
		data := srv.newTemplateData(req, nil)

		assert.Equal(t, "dark", data.Theme)
	})
}

func TestServer_BrandingInTemplates(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"test-v",
		Config{
			Domain:         []string{"example.com"},
			Protocol:       "https",
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Acme Corp Secrets",
		})
	require.NoError(t, err)

	t.Run("home page contains custom branding", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", http.NoBody)
		rr := httptest.NewRecorder()

		srv.indexCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		body := rr.Body.String()
		assert.Contains(t, body, "Acme Corp Secrets")
		// the title is now SEO optimized, not using branding in title
		assert.Contains(t, body, "<title>Secure Password Sharing - Self-Destructing Messages</title>")
		assert.NotContains(t, body, "Safe Secrets")
	})

	t.Run("about page contains custom branding", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/about", http.NoBody)
		rr := httptest.NewRecorder()

		srv.aboutViewCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		body := rr.Body.String()
		assert.Contains(t, body, "Acme Corp Secrets")
		assert.NotContains(t, body, "Safe Secrets")
	})
}

func TestServer_getValidatedHost(t *testing.T) {
	tests := []struct {
		name         string
		protocol     string
		domains      []string
		host         string
		expectedHost string
	}{
		{
			name:         "allowed domain",
			protocol:     "https",
			domains:      []string{"example.com", "alt.example.com"},
			host:         "example.com",
			expectedHost: "example.com",
		},
		{
			name:         "case insensitive matching",
			protocol:     "https",
			domains:      []string{"example.com"},
			host:         "EXAMPLE.COM",
			expectedHost: "EXAMPLE.COM",
		},
		{
			name:         "https standard port stripped",
			protocol:     "https",
			domains:      []string{"example.com"},
			host:         "example.com:443",
			expectedHost: "example.com",
		},
		{
			name:         "http standard port stripped",
			protocol:     "http",
			domains:      []string{"example.com"},
			host:         "example.com:80",
			expectedHost: "example.com",
		},
		{
			name:         "https custom port kept",
			protocol:     "https",
			domains:      []string{"example.com"},
			host:         "example.com:8080",
			expectedHost: "example.com:8080",
		},
		{
			name:         "http non-standard port kept",
			protocol:     "http",
			domains:      []string{"example.com"},
			host:         "example.com:443",
			expectedHost: "example.com:443",
		},
		{
			name:         "IPv6 address supported",
			protocol:     "https",
			domains:      []string{"::1"},
			host:         "[::1]:8080",
			expectedHost: "[::1]:8080",
		},
		{
			name:         "IPv6 standard port stripped",
			protocol:     "https",
			domains:      []string{"::1"},
			host:         "[::1]:443",
			expectedHost: "::1",
		},
		{
			name:         "second allowed domain",
			protocol:     "https",
			domains:      []string{"example.com", "alt.example.com"},
			host:         "alt.example.com",
			expectedHost: "alt.example.com",
		},
		{
			name:         "disallowed domain falls back to first configured",
			protocol:     "https",
			domains:      []string{"example.com", "alt.example.com"},
			host:         "malicious.com",
			expectedHost: "example.com",
		},
		{
			name:         "empty host falls back to first configured",
			protocol:     "https",
			domains:      []string{"example.com"},
			host:         "",
			expectedHost: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := store.NewInMemory(time.Second)
			srv, err := New(
				messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
					MaxDuration:    10 * time.Hour,
					MaxPinAttempts: 3,
				}),
				"test-v",
				Config{
					Domain:         tt.domains,
					Protocol:       tt.protocol,
					PinSize:        5,
					MaxPinAttempts: 3,
					MaxExpire:      10 * time.Hour,
					Branding:       "Test Brand",
				})
			require.NoError(t, err)

			req := httptest.NewRequest("GET", "/", http.NoBody)
			req.Host = tt.host

			result := srv.getValidatedHost(req)
			assert.Equal(t, tt.expectedHost, result)
		})
	}
}

func TestServer_generateLinkCtrl_MultipleDomain(t *testing.T) {
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
			Protocol:       "https",
			Domain:         []string{"example.com", "alt.example.com"},
		})
	require.NoError(t, err)

	t.Run("uses request domain when allowed", func(t *testing.T) {
		formData := url.Values{
			"message": {"secret message"},
			"exp":     {"15"},
			"expUnit": {"m"},
			"pin":     {"1", "2", "3", "4", "5"},
		}

		req := httptest.NewRequest(http.MethodPost, "/generate-link", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Host = "alt.example.com"
		rr := httptest.NewRecorder()

		srv.generateLinkCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "https://alt.example.com/message/")
	})

	t.Run("falls back to first domain when disallowed", func(t *testing.T) {
		formData := url.Values{
			"message": {"secret message"},
			"exp":     {"15"},
			"expUnit": {"m"},
			"pin":     {"1", "2", "3", "4", "5"},
		}

		req := httptest.NewRequest(http.MethodPost, "/generate-link", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Host = "malicious.com"
		rr := httptest.NewRecorder()

		srv.generateLinkCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "https://example.com/message/")
	})
}

func TestServer_IPv6LinkGeneration(t *testing.T) {
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
			Protocol:       "https",
			Domain:         []string{"::1", "2001:db8::1"},
		})
	require.NoError(t, err)

	t.Run("IPv6 with standard port gets bracketed", func(t *testing.T) {
		formData := url.Values{
			"message": {"secret message"},
			"exp":     {"15"},
			"expUnit": {"m"},
			"pin":     {"1", "2", "3", "4", "5"},
		}

		req := httptest.NewRequest(http.MethodPost, "/generate-link", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Host = "[::1]:443" // iPv6 with standard HTTPS port
		rr := httptest.NewRecorder()

		srv.generateLinkCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "https://[::1]/message/", "IPv6 without port should be bracketed")
		assert.NotContains(t, rr.Body.String(), "https://::1/message/", "Should not have unbracketed IPv6")
	})

	t.Run("IPv6 with non-standard port stays bracketed", func(t *testing.T) {
		formData := url.Values{
			"message": {"secret message"},
			"exp":     {"15"},
			"expUnit": {"m"},
			"pin":     {"1", "2", "3", "4", "5"},
		}

		req := httptest.NewRequest(http.MethodPost, "/generate-link", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Host = "[::1]:8080" // iPv6 with non-standard port
		rr := httptest.NewRecorder()

		srv.generateLinkCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "https://[::1]:8080/message/", "IPv6 with non-standard port should keep port and brackets")
	})

	t.Run("IPv6 without port gets bracketed", func(t *testing.T) {
		formData := url.Values{
			"message": {"secret message"},
			"exp":     {"15"},
			"expUnit": {"m"},
			"pin":     {"1", "2", "3", "4", "5"},
		}

		req := httptest.NewRequest(http.MethodPost, "/generate-link", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Host = "2001:db8::1" // iPv6 without port or brackets
		rr := httptest.NewRecorder()

		srv.generateLinkCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "https://[2001:db8::1]/message/", "IPv6 without brackets should be bracketed")
	})

	t.Run("IPv6 edge case - unbracketed with port", func(t *testing.T) {
		// this tests the edge case where getValidatedHost might return IPv6:port without brackets
		// though this shouldn't normally happen with our current getValidatedHost implementation
		formData := url.Values{
			"message": {"secret message"},
			"exp":     {"15"},
			"expUnit": {"m"},
			"pin":     {"1", "2", "3", "4", "5"},
		}

		// create a custom request that simulates an edge case
		req := httptest.NewRequest(http.MethodPost, "/generate-link", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Host = "2001:db8::1:8080" // intentionally malformed: not valid IPv6 with port; correct form is [2001:db8::1]:8080
		rr := httptest.NewRecorder()

		srv.generateLinkCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		// should either be bracketed IPv6 address or fallback to first domain
		body := rr.Body.String()
		assert.Contains(t, body, "https://[",
			"Should handle IPv6 edge case properly (must be bracketed IPv6 or fallback to valid domain): %s", body)
	})
}

func TestIPv6BracketingLogic(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		description string
	}{
		{
			name:        "IPv4 unchanged",
			input:       "192.168.1.1",
			expected:    "192.168.1.1",
			description: "IPv4 addresses should not be modified",
		},
		{
			name:        "IPv4 with port unchanged",
			input:       "192.168.1.1:8080",
			expected:    "192.168.1.1:8080",
			description: "IPv4 with port should not be modified",
		},
		{
			name:        "hostname unchanged",
			input:       "example.com",
			expected:    "example.com",
			description: "Hostnames should not be modified",
		},
		{
			name:        "hostname with port unchanged",
			input:       "example.com:8080",
			expected:    "example.com:8080",
			description: "Hostnames with port should not be modified",
		},
		{
			name:        "IPv6 gets bracketed",
			input:       "::1",
			expected:    "[::1]",
			description: "Unbracketed IPv6 should get bracketed",
		},
		{
			name:        "IPv6 full address gets bracketed",
			input:       "2001:db8::1",
			expected:    "[2001:db8::1]",
			description: "Full IPv6 address should get bracketed",
		},
		{
			name:        "IPv6 already bracketed unchanged",
			input:       "[::1]",
			expected:    "[::1]",
			description: "Already bracketed IPv6 should stay unchanged",
		},
		{
			name:        "IPv6 with port stays bracketed",
			input:       "[::1]:8080",
			expected:    "[::1]:8080",
			description: "IPv6 with port and brackets should stay unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// simulate the bracketing logic from generateLinkCtrl
			validatedHost := tt.input

			// this is the same logic from the function
			host, port, err := net.SplitHostPort(validatedHost)
			if err != nil {
				// no port present, check if the whole string is an IPv6 address
				if ip := net.ParseIP(validatedHost); ip != nil && ip.To4() == nil {
					validatedHost = "[" + validatedHost + "]"
				}
			} else {
				// port present, check if host part is IPv6 and needs bracketing
				if ip := net.ParseIP(host); ip != nil && ip.To4() == nil && !strings.HasPrefix(host, "[") {
					validatedHost = "[" + host + "]:" + port
				}
			}

			assert.Equal(t, tt.expected, validatedHost, tt.description)
		})
	}
}

func TestServer_URLConstruction(t *testing.T) {
	tests := []struct {
		name        string
		messageKey  string
		expectedURL string
	}{
		{
			name:        "normal key",
			messageKey:  "68cb14f4-e2767ae5-ef7b-492f-9456-25d3e998074f",
			expectedURL: "https://example.com/message/68cb14f4-e2767ae5-ef7b-492f-9456-25d3e998074f",
		},
		{
			name:        "key with spaces (hypothetical)",
			messageKey:  "test key with spaces",
			expectedURL: "https://example.com/message/test%20key%20with%20spaces",
		},
		{
			name:        "key with special chars (hypothetical)",
			messageKey:  "test+key&with=special?chars",
			expectedURL: "https://example.com/message/test+key&with=special%3Fchars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// simulate URL construction logic
			validatedHost := "example.com"
			protocol := "https"

			msgURL := (&url.URL{
				Scheme: protocol,
				Host:   validatedHost,
				Path:   path.Join("/message", tt.messageKey),
			}).String()

			assert.Equal(t, tt.expectedURL, msgURL)
		})
	}
}

func TestServer_SEOMetaTags(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"test-v",
		Config{
			Domain:         []string{"example.com"},
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Test SEO",
			Protocol:       "https",
		})
	require.NoError(t, err)

	t.Run("home page has SEO meta tags", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", http.NoBody)
		req.Host = "example.com"
		rr := httptest.NewRecorder()

		srv.indexCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		body := rr.Body.String()

		// check optimized title
		assert.Contains(t, body, "<title>Secure Password Sharing - Self-Destructing Messages</title>")

		// check meta description
		assert.Contains(t, body, `<meta name="description" content="Share sensitive information securely with self-destructing messages protected by PIN codes. Free, open-source, and privacy-focused password sharing."`)

		// check canonical URL
		assert.Contains(t, body, `<link rel="canonical" href="https://example.com/">`)

		// check Open Graph tags
		assert.Contains(t, body, `<meta property="og:type" content="website">`)
		assert.Contains(t, body, `<meta property="og:url" content="https://example.com/">`)
		assert.Contains(t, body, `<meta property="og:title" content="Secure Password Sharing - Self-Destructing Messages">`)
		assert.Contains(t, body, `<meta property="og:site_name" content="Test SEO">`)

		// check Twitter Card tags
		assert.Contains(t, body, `<meta name="twitter:card" content="summary_large_image">`)
		assert.Contains(t, body, `<meta name="twitter:url" content="https://example.com/">`)
		assert.Contains(t, body, `<meta name="twitter:title" content="Secure Password Sharing - Self-Destructing Messages">`)

		// check structured data JSON-LD
		assert.Contains(t, body, `"@type": "WebApplication"`)
		assert.Contains(t, body, `"applicationCategory": "SecurityApplication"`)
		assert.Contains(t, body, `"name": "Test SEO"`)

		// check other meta tags
		assert.Contains(t, body, `<meta name="keywords" content="password sharing, secure messaging`)
		assert.Contains(t, body, `<meta name="author" content="Umputun">`)
		assert.Contains(t, body, `<meta name="robots" content="index, follow">`)
	})

	t.Run("about page has SEO meta tags", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/about", http.NoBody)
		req.Host = "example.com"
		rr := httptest.NewRecorder()

		srv.aboutViewCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		body := rr.Body.String()

		// check optimized title
		assert.Contains(t, body, "<title>How It Works - Encrypted Message Sharing</title>")

		// check meta description
		assert.Contains(t, body, `<meta name="description" content="Learn how SafeSecret protects your sensitive information with PIN-protected encryption, self-destructing messages, and zero-knowledge architecture."`)

		// check canonical URL
		assert.Contains(t, body, `<link rel="canonical" href="https://example.com/about">`)

		// check Open Graph tags
		assert.Contains(t, body, `<meta property="og:url" content="https://example.com/about">`)
		assert.Contains(t, body, `<meta property="og:title" content="How It Works - Encrypted Message Sharing">`)
	})

	t.Run("message page has noindex meta tag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/message/test-key-123", http.NoBody)
		req.Host = "example.com"
		rr := httptest.NewRecorder()

		srv.showMessageViewCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		body := rr.Body.String()

		// check that message pages have noindex, nofollow
		assert.Contains(t, body, `<meta name="robots" content="noindex, nofollow">`)
		// verify canonical URL is still set
		assert.Contains(t, body, `<link rel="canonical" href="https://example.com/message/test-key-123">`)
		// check X-Robots-Tag header for defense-in-depth
		assert.Equal(t, "noindex, nofollow, noarchive", rr.Header().Get("X-Robots-Tag"))
	})
}

func TestServer_CanonicalURL(t *testing.T) {
	eng := store.NewInMemory(time.Second)
	srv, err := New(
		messager.New(eng, messager.Crypt{Key: "123456789012345678901234567"}, messager.Params{
			MaxDuration:    10 * time.Hour,
			MaxPinAttempts: 3,
		}),
		"test-v",
		Config{
			Domain:         []string{"example.com"},
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
			Branding:       "Test SEO",
			Protocol:       "https",
		})
	require.NoError(t, err)

	t.Run("generates correct canonical URL", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", http.NoBody)
		req.Host = "example.com"
		rr := httptest.NewRecorder()

		srv.indexCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		body := rr.Body.String()

		// should use configured domain for canonical
		assert.Contains(t, body, `<link rel="canonical" href="https://example.com/">`)
		assert.Contains(t, body, `<meta property="og:url" content="https://example.com/">`)
	})

	t.Run("canonical URL for about page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/about", http.NoBody)
		req.Host = "example.com"
		rr := httptest.NewRecorder()

		srv.aboutViewCtrl(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		body := rr.Body.String()

		// should include path in canonical URL
		assert.Contains(t, body, `<link rel="canonical" href="https://example.com/about">`)
		assert.Contains(t, body, `<meta property="og:url" content="https://example.com/about">`)
	})
}
