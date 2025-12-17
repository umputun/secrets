package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailSender(t *testing.T) {
	t.Run("disabled returns nil", func(t *testing.T) {
		sender, err := NewEmailSender(EmailConfig{Enabled: false}, "Test Brand")
		require.NoError(t, err)
		assert.Nil(t, sender)
	})

	t.Run("enabled without host fails", func(t *testing.T) {
		_, err := NewEmailSender(EmailConfig{Enabled: true, From: "test@example.com"}, "Test Brand")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host is required")
	})

	t.Run("enabled without from fails", func(t *testing.T) {
		_, err := NewEmailSender(EmailConfig{Enabled: true, Host: "smtp.example.com"}, "Test Brand")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "from address is required")
	})

	t.Run("enabled with valid config succeeds", func(t *testing.T) {
		sender, err := NewEmailSender(EmailConfig{
			Enabled: true,
			Host:    "smtp.example.com",
			Port:    587,
			From:    "noreply@example.com",
		}, "Test Brand")
		require.NoError(t, err)
		assert.NotNil(t, sender)
	})

	t.Run("invalid template file fails", func(t *testing.T) {
		_, err := NewEmailSender(EmailConfig{
			Enabled:  true,
			Host:     "smtp.example.com",
			From:     "noreply@example.com",
			Template: "/nonexistent/path/template.html",
		}, "Test Brand")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read email template")
	})
}

func TestEmailSender_RenderBody(t *testing.T) {
	sender, err := NewEmailSender(EmailConfig{
		Enabled: true,
		Host:    "smtp.example.com",
		From:    "noreply@example.com",
	}, "Safe Secrets")
	require.NoError(t, err)

	body, err := sender.RenderBody("https://example.com/message/abc123", "John Doe")
	require.NoError(t, err)

	assert.Contains(t, body, "https://example.com/message/abc123")
	assert.Contains(t, body, "John Doe")
	assert.Contains(t, body, "Safe Secrets")
	assert.Contains(t, body, "View Secure Message")
}

func TestEmailSender_extractEmail(t *testing.T) {
	sender := &emailSender{}

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
			result := sender.extractEmail(tt.from)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmailSender_extractDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		expected string
	}{
		{"plain email", "test@example.com", ""},
		{"with display name", `"John Doe" <john@example.com>`, "John Doe"},
		{"name without quotes", "John Doe <john@example.com>", "John Doe"},
		{"angle brackets only", "<noreply@example.com>", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &emailSender{cfg: EmailConfig{From: tt.from}}
			result := sender.extractDisplayName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmailSender_buildFromAddress(t *testing.T) {
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
			sender := &emailSender{cfg: EmailConfig{From: tt.configFrom}}
			result := sender.buildFromAddress(tt.displayName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmailSender_buildMailtoDestination(t *testing.T) {
	sender := &emailSender{}

	t.Run("single recipient", func(t *testing.T) {
		result := sender.buildMailtoDestination(
			[]string{"user@example.com"},
			"Test Subject",
			`"Sender" <sender@example.com>`,
		)
		assert.Contains(t, result, "mailto:")
		assert.Contains(t, result, "user@example.com")
		assert.Contains(t, result, "subject=")
		assert.Contains(t, result, "from=")
	})

	t.Run("multiple recipients", func(t *testing.T) {
		result := sender.buildMailtoDestination(
			[]string{"user1@example.com", "user2@example.com"},
			"Test Subject",
			`"Sender" <sender@example.com>`,
		)
		assert.Contains(t, result, "user1@example.com")
		assert.Contains(t, result, "user2@example.com")
	})

	t.Run("empty subject", func(t *testing.T) {
		result := sender.buildMailtoDestination(
			[]string{"user@example.com"},
			"",
			`"Sender" <sender@example.com>`,
		)
		assert.NotContains(t, result, "subject=")
	})
}

func TestEmailSender_GetDefaultFromName(t *testing.T) {
	t.Run("returns display name from config", func(t *testing.T) {
		sender := &emailSender{
			cfg:      EmailConfig{From: `"Safe Secrets" <noreply@example.com>`},
			branding: "Fallback Brand",
		}
		result := sender.GetDefaultFromName()
		assert.Equal(t, "Safe Secrets", result)
	})

	t.Run("returns branding when no display name", func(t *testing.T) {
		sender := &emailSender{
			cfg:      EmailConfig{From: "noreply@example.com"},
			branding: "Fallback Brand",
		}
		result := sender.GetDefaultFromName()
		assert.Equal(t, "Fallback Brand", result)
	})
}

func TestEmailRequest(t *testing.T) {
	req := EmailRequest{
		To:       []string{"user@example.com"},
		CC:       []string{"cc@example.com"},
		Subject:  "Test Subject",
		FromName: "Test Sender",
		Link:     "https://example.com/secret/123",
	}

	assert.Equal(t, []string{"user@example.com"}, req.To)
	assert.Equal(t, []string{"cc@example.com"}, req.CC)
	assert.Equal(t, "Test Subject", req.Subject)
	assert.Equal(t, "Test Sender", req.FromName)
	assert.Equal(t, "https://example.com/secret/123", req.Link)
}

func TestEmailConfig(t *testing.T) {
	cfg := EmailConfig{
		Enabled:     true,
		Host:        "smtp.example.com",
		Port:        587,
		Username:    "user",
		Password:    "pass",
		From:        "noreply@example.com",
		TLS:         true,
		Timeout:     30 * time.Second,
		ContentType: "text/html",
		Charset:     "UTF-8",
		Template:    "/path/to/template.html",
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "smtp.example.com", cfg.Host)
	assert.Equal(t, 587, cfg.Port)
	assert.Equal(t, "user", cfg.Username)
	assert.Equal(t, "pass", cfg.Password)
	assert.Equal(t, "noreply@example.com", cfg.From)
	assert.True(t, cfg.TLS)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
}

// mockEmailSender implements EmailSender for testing
type mockEmailSender struct {
	sendCalled bool
	lastReq    EmailRequest
	sendErr    error
}

func (m *mockEmailSender) Send(ctx context.Context, req EmailRequest) error {
	m.sendCalled = true
	m.lastReq = req
	return m.sendErr
}

func (m *mockEmailSender) RenderBody(link, fromName string) (string, error) {
	return "rendered body with " + link + " from " + fromName, nil
}

func TestMockEmailSender(t *testing.T) {
	mock := &mockEmailSender{}

	body, err := mock.RenderBody("https://example.com/link", "Sender")
	require.NoError(t, err)
	assert.Contains(t, body, "https://example.com/link")
	assert.Contains(t, body, "Sender")

	err = mock.Send(context.Background(), EmailRequest{
		To:      []string{"user@example.com"},
		Subject: "Test",
		Link:    "https://example.com/link",
	})
	require.NoError(t, err)
	assert.True(t, mock.sendCalled)
	assert.Equal(t, "Test", mock.lastReq.Subject)
}

func TestDefaultEmailTemplate(t *testing.T) {
	// verify template contains expected placeholders and structure
	assert.Contains(t, defaultEmailTemplate, "{{.Link}}")
	assert.Contains(t, defaultEmailTemplate, "{{.From}}")
	assert.Contains(t, defaultEmailTemplate, "{{.Branding}}")
	assert.Contains(t, defaultEmailTemplate, "View Secure Message")
	assert.Contains(t, defaultEmailTemplate, "<!DOCTYPE html>")
}
