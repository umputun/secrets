package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
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

	assert.NoError(t, err)

	assert.Equal(t, 8, len(cache))
	assert.NotNil(t, cache["404.tmpl.html"])
	assert.NotNil(t, cache["about.tmpl.html"])
	assert.NotNil(t, cache["home.tmpl.html"])
	assert.NotNil(t, cache["show-message.tmpl.html"])
	assert.NotNil(t, cache["decoded-message.tmpl.html"])
	assert.NotNil(t, cache["error.tmpl.html"])
	assert.NotNil(t, cache["footer.tmpl.html"])
	assert.NotNil(t, cache["secure-link.tmpl.html"])
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
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
		})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	srv.indexCtrl(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "SafeSecret")
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
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
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
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
		})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/message/testkey123", http.NoBody)
	rr := httptest.NewRecorder()

	// add chi context with URL param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("key", "testkey123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	srv.showMessageViewCtrl(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("HX-Trigger-After-Swap"), "setUpPinInputListeners")
	assert.Contains(t, rr.Body.String(), "testkey123")
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
			Domain:         "example.com",
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
				assert.Contains(t, body, "value=15") // verify it returns the form
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

func TestServer_loadMessageCtrl(t *testing.T) {
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
			PinSize:        5,
			MaxPinAttempts: 3,
			MaxExpire:      10 * time.Hour,
		})
	require.NoError(t, err)

	t.Run("existing template", func(t *testing.T) {
		rr := httptest.NewRecorder()
		data := templateData{
			Form:    createMsgForm{Exp: 15, MaxExp: "10 hours"},
			PinSize: 5,
		}
		srv.render(rr, http.StatusOK, "home.tmpl.html", baseTmpl, data)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "SafeSecret")
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
			Form:    createMsgForm{Exp: 15, MaxExp: "10 hours"},
			PinSize: 5,
		}
		srv.render(rr, http.StatusOK, "home.tmpl.html", "", data)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "SafeSecret")
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
