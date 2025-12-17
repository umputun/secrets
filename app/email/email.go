package email

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net/mail"
	"net/url"
	"os"
	"time"

	"github.com/go-pkgz/notify"
)

//go:embed email.tmpl.html
var defaultEmailTemplate string

// Request contains all parameters for sending an email
type Request struct {
	To       string // recipient email address
	Subject  string // email subject
	FromName string // display name for From header
	Link     string // the secret link to include in email body
}

// Config contains SMTP configuration
type Config struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string
	From     string // format: "Display Name <email>" or just "email"
	TLS      bool
	Timeout  time.Duration
	Template string // path to custom template file (optional)
}

// Sender sends emails with secret links using go-pkgz/notify
type Sender struct {
	notifier        *notify.Email
	cfg             Config
	branding        string
	tmpl            *template.Template
	defaultFromName string // cached default from name
}

// NewSender creates a new email sender with the given configuration
func NewSender(cfg Config, branding string) (*Sender, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	if cfg.Host == "" {
		return nil, fmt.Errorf("email host is required when email is enabled")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("email from address is required when email is enabled")
	}

	// set defaults
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// create notifier with HTML content type
	notifier := notify.NewEmail(notify.SMTPParams{
		Host:        cfg.Host,
		Port:        cfg.Port,
		TLS:         cfg.TLS,
		ContentType: "text/html",
		Charset:     "UTF-8",
		Username:    cfg.Username,
		Password:    cfg.Password,
		TimeOut:     cfg.Timeout,
	})

	// load template - use default unless custom template is configured
	tmplContent := defaultEmailTemplate
	if cfg.Template != "" {
		content, readErr := os.ReadFile(cfg.Template)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read email template file: %w", readErr)
		}
		tmplContent = string(content)
	}
	tmpl, err := template.New("email").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email template: %w", err)
	}

	s := &Sender{
		notifier: notifier,
		cfg:      cfg,
		branding: branding,
		tmpl:     tmpl,
	}
	// cache the default from name at construction time
	s.defaultFromName = s.computeDefaultFromName()
	return s, nil
}

// Send sends an email with the secret link
func (s *Sender) Send(ctx context.Context, req Request) error {
	body, err := s.RenderBody(req.Link, req.FromName)
	if err != nil {
		return fmt.Errorf("failed to render email body: %w", err)
	}

	fromAddr := s.buildFromAddress(req.FromName)
	destination := s.buildMailtoDestination(req.To, req.Subject, fromAddr)

	if err := s.notifier.Send(ctx, destination, body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

// RenderBody renders the email body with the given link and from name
func (s *Sender) RenderBody(link, fromName string) (string, error) {
	data := struct {
		Link     string
		From     string
		Branding string
	}{
		Link:     link,
		From:     fromName,
		Branding: s.branding,
	}

	var buf bytes.Buffer
	if err := s.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

// buildFromAddress builds the From header value with display name
func (s *Sender) buildFromAddress(displayName string) string {
	// extract email from configured From (might be "Name <email>" or just "email")
	emailAddr := s.extractEmail(s.cfg.From)

	if displayName == "" {
		// use the original From as-is
		return s.cfg.From
	}

	// build new From with custom display name
	return fmt.Sprintf("%q <%s>", displayName, emailAddr)
}

// extractEmail extracts just the email address from a From string
func (s *Sender) extractEmail(from string) string {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		// assume it's already just an email address
		return from
	}
	return addr.Address
}

// computeDefaultFromName computes the default from name from config or branding
func (s *Sender) computeDefaultFromName() string {
	addr, err := mail.ParseAddress(s.cfg.From)
	if err == nil && addr.Name != "" {
		return addr.Name
	}
	return s.branding
}

// GetDefaultFromName returns the cached default from name
func (s *Sender) GetDefaultFromName() string {
	return s.defaultFromName
}

// buildMailtoDestination builds the mailto URL for go-pkgz/notify
func (s *Sender) buildMailtoDestination(recipient, subject, from string) string {
	mailto := "mailto:" + recipient

	params := url.Values{}
	if subject != "" {
		params.Set("subject", subject)
	}
	if from != "" {
		params.Set("from", from)
	}

	if len(params) > 0 {
		mailto += "?" + params.Encode()
	}
	return mailto
}

// IsValidEmail performs email validation using RFC 5322 parsing
func IsValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	// ensure the parsed address matches the input (no display name was provided)
	return addr.Address == email
}
