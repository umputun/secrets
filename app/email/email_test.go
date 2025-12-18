package email

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/secrets/app/email/mocks"
)

func TestNewSender(t *testing.T) {
	t.Run("disabled returns nil", func(t *testing.T) {
		sndr, err := NewSender(Config{Enabled: false, Branding: "Test Brand"})
		require.NoError(t, err)
		assert.Nil(t, sndr)
	})

	t.Run("enabled without host fails", func(t *testing.T) {
		_, err := NewSender(Config{Enabled: true, From: "test@example.com", Branding: "Test Brand"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host is required")
	})

	t.Run("enabled without from fails", func(t *testing.T) {
		_, err := NewSender(Config{Enabled: true, Host: "smtp.example.com", Branding: "Test Brand"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "from address is required")
	})

	t.Run("enabled with valid config succeeds", func(t *testing.T) {
		sndr, err := NewSender(Config{
			Enabled:  true,
			Host:     "smtp.example.com",
			Port:     587,
			From:     "noreply@example.com",
			Branding: "Test Brand",
		})
		require.NoError(t, err)
		assert.NotNil(t, sndr)
	})

	t.Run("invalid template file fails", func(t *testing.T) {
		_, err := NewSender(Config{
			Enabled:  true,
			Host:     "smtp.example.com",
			From:     "noreply@example.com",
			Template: "/nonexistent/path/template.html",
			Branding: "Test Brand",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read email template")
	})
}

func TestSender_RenderBody(t *testing.T) {
	sndr, err := NewSender(Config{
		Enabled:     true,
		Host:        "smtp.example.com",
		From:        "noreply@example.com",
		Branding:    "Safe Secrets",
		BrandingURL: "https://safesecret.info",
	})
	require.NoError(t, err)

	body, err := sndr.RenderBody("https://example.com/message/abc123", "John Doe")
	require.NoError(t, err)

	assert.Contains(t, body, "https://example.com/message/abc123")
	assert.Contains(t, body, "John Doe")
	assert.Contains(t, body, "Safe Secrets")
	assert.Contains(t, body, "https://safesecret.info")
	assert.Contains(t, body, "View Secure Message")
}

func TestSender_extractEmail(t *testing.T) {
	sndr := &Sender{}

	tests := []struct {
		name     string
		from     string
		expected string
	}{
		{"plain email", "test@example.com", "test@example.com"},
		{"with display name", `"John Doe" <john@example.com>`, "john@example.com"},
		{"angle brackets only", "<noreply@example.com>", "noreply@example.com"},
		{"name and email", "John <john@example.com>", "john@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sndr.extractEmail(tt.from)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSender_computeDefaultFromName(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		branding string
		expected string
	}{
		{"plain email returns branding", "test@example.com", "TestBrand", "TestBrand"},
		{"with display name returns it", `"John Doe" <john@example.com>`, "TestBrand", "John Doe"},
		{"name without quotes returns it", "John Doe <john@example.com>", "TestBrand", "John Doe"},
		{"angle brackets only returns branding", "<noreply@example.com>", "TestBrand", "TestBrand"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sndr := &Sender{cfg: Config{From: tt.from, Branding: tt.branding}}
			result := sndr.computeDefaultFromName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSender_buildFromAddress(t *testing.T) {
	tests := []struct {
		name        string
		configFrom  string
		displayName string
		expected    string
	}{
		{"empty display name uses config", "noreply@example.com", "", "noreply@example.com"},
		{"custom display name", "noreply@example.com", "My App", `"My App" <noreply@example.com>`},
		{"display name with config having display", `"Original" <noreply@example.com>`, "Custom", `"Custom" <noreply@example.com>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sndr := &Sender{cfg: Config{From: tt.configFrom}}
			result := sndr.buildFromAddress(tt.displayName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSender_buildMailtoDestination(t *testing.T) {
	sndr := &Sender{}

	t.Run("with subject and from", func(t *testing.T) {
		result := sndr.buildMailtoDestination("user@example.com", "Test Subject", `"Sender" <sender@example.com>`)
		assert.Contains(t, result, "mailto:user@example.com")
		assert.Contains(t, result, "subject=")
		assert.Contains(t, result, "from=")
	})

	t.Run("empty subject", func(t *testing.T) {
		result := sndr.buildMailtoDestination("user@example.com", "", `"Sender" <sender@example.com>`)
		assert.NotContains(t, result, "subject=")
		assert.Contains(t, result, "from=")
	})
}

func TestSender_GetDefaultFromName(t *testing.T) {
	t.Run("returns cached display name from config", func(t *testing.T) {
		sndr := &Sender{
			cfg:             Config{From: `"Safe Secrets" <noreply@example.com>`, Branding: "Fallback Brand"},
			defaultFromName: "Safe Secrets", // cached value
		}
		result := sndr.GetDefaultFromName()
		assert.Equal(t, "Safe Secrets", result)
	})

	t.Run("returns cached branding when no display name", func(t *testing.T) {
		sndr := &Sender{
			cfg:             Config{From: "noreply@example.com", Branding: "Fallback Brand"},
			defaultFromName: "Fallback Brand", // cached value
		}
		result := sndr.GetDefaultFromName()
		assert.Equal(t, "Fallback Brand", result)
	})

	t.Run("via NewSender caches correctly", func(t *testing.T) {
		sndr, err := NewSender(Config{
			Enabled:  true,
			Host:     "smtp.example.com",
			From:     `"Test App" <test@example.com>`,
			Branding: "Branding",
		})
		require.NoError(t, err)
		assert.Equal(t, "Test App", sndr.GetDefaultFromName())
	})
}

func TestDefaultEmailTemplate(t *testing.T) {
	// verify template contains expected placeholders and structure
	assert.Contains(t, defaultEmailTemplate, "{{.Link}}")
	assert.Contains(t, defaultEmailTemplate, "{{.From}}")
	assert.Contains(t, defaultEmailTemplate, "{{.Branding}}")
	assert.Contains(t, defaultEmailTemplate, "{{.BrandingURL}}")
	assert.Contains(t, defaultEmailTemplate, "View Secure Message")
	assert.Contains(t, defaultEmailTemplate, "<!DOCTYPE html>")
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"valid simple email", "user@example.com", true},
		{"valid with subdomain", "user@mail.example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"invalid no at", "userexample.com", false},
		{"invalid no domain", "user@", false},
		{"invalid no local", "@example.com", false},
		{"invalid double at", "user@@example.com", false},
		{"invalid trailing dot", "user@example.", false},
		{"invalid with display name", "John Doe <john@example.com>", false},
		{"invalid empty", "", false},
		{"invalid spaces", "user @example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEmail(tt.email)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestSender_Send(t *testing.T) {
	t.Run("success sends email with correct parameters", func(t *testing.T) {
		mockNotifier := &mocks.NotifierMock{
			SendFunc: func(_ context.Context, _, _ string) error { return nil },
		}

		sndr, err := NewSender(Config{
			Enabled:  true,
			Host:     "smtp.example.com",
			From:     "noreply@example.com",
			Branding: "Test Brand",
			Notifier: mockNotifier,
		})
		require.NoError(t, err)

		req := Request{
			To:       "recipient@example.com",
			Subject:  "Test Subject",
			FromName: "Sender Name",
			Link:     "https://example.com/message/abc123",
		}
		err = sndr.Send(context.Background(), req)
		require.NoError(t, err)

		// verify notifier was called once
		calls := mockNotifier.SendCalls()
		require.Len(t, calls, 1)

		// verify destination contains recipient email
		assert.Contains(t, calls[0].Destination, "mailto:recipient@example.com")
		assert.Contains(t, calls[0].Destination, "subject=")
		assert.Contains(t, calls[0].Destination, "from=")

		// verify body contains link and from name
		assert.Contains(t, calls[0].Text, "https://example.com/message/abc123")
		assert.Contains(t, calls[0].Text, "Sender Name")
		assert.Contains(t, calls[0].Text, "Test Brand")
	})

	t.Run("notifier error is propagated", func(t *testing.T) {
		mockNotifier := &mocks.NotifierMock{
			SendFunc: func(_ context.Context, _, _ string) error { return errors.New("smtp connection failed") },
		}

		sndr, err := NewSender(Config{
			Enabled:  true,
			Host:     "smtp.example.com",
			From:     "noreply@example.com",
			Branding: "Test Brand",
			Notifier: mockNotifier,
		})
		require.NoError(t, err)

		req := Request{To: "recipient@example.com", Subject: "Test", FromName: "Test", Link: "https://example.com/message/abc"}
		err = sndr.Send(context.Background(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send email")
		assert.Contains(t, err.Error(), "smtp connection failed")
	})

	t.Run("empty from name uses default", func(t *testing.T) {
		mockNotifier := &mocks.NotifierMock{
			SendFunc: func(_ context.Context, _, _ string) error { return nil },
		}

		sndr, err := NewSender(Config{
			Enabled:  true,
			Host:     "smtp.example.com",
			From:     `"Default Sender" <noreply@example.com>`,
			Branding: "Fallback Brand",
			Notifier: mockNotifier,
		})
		require.NoError(t, err)

		req := Request{To: "recipient@example.com", Subject: "Test", FromName: "", Link: "https://example.com/message/abc"}
		err = sndr.Send(context.Background(), req)
		require.NoError(t, err)

		calls := mockNotifier.SendCalls()
		require.Len(t, calls, 1)
		// when FromName is empty, buildFromAddress uses original From config
		assert.Contains(t, calls[0].Destination, "from=")
	})
}
