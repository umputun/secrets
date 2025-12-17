package server

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-pkgz/notify"
)

//go:generate moq -out email_sender_mock.go -fmt goimports . EmailSender

// EmailSender defines interface for sending emails with secret links
type EmailSender interface {
	Send(ctx context.Context, req EmailRequest) error
	RenderBody(link, fromName string) (string, error)
	GetDefaultFromName() string
}

// EmailRequest contains all parameters for sending an email
type EmailRequest struct {
	To       []string // recipient email addresses
	CC       []string // CC email addresses
	Subject  string   // email subject
	FromName string   // display name for From header
	Link     string   // the secret link to include in email body
}

// EmailConfig contains SMTP configuration
type EmailConfig struct {
	Enabled     bool
	Host        string
	Port        int
	Username    string
	Password    string
	From        string // format: "Display Name <email>" or just "email"
	TLS         bool
	Timeout     time.Duration
	ContentType string
	Charset     string
	Template    string // path to custom template file (optional)
}

// emailSender implements EmailSender using go-pkgz/notify
type emailSender struct {
	notifier *notify.Email
	cfg      EmailConfig
	branding string
	tmpl     *template.Template
}

// defaultEmailTemplate is the embedded default HTML email template
const defaultEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Secure Message</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; background-color: #f8fafc;">
    <table role="presentation" style="width: 100%; border-collapse: collapse;">
        <tr>
            <td style="padding: 40px 20px;">
                <table role="presentation" style="max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 12px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);">
                    <tr>
                        <td style="padding: 40px;">
                            <h1 style="margin: 0 0 24px 0; color: #0f172a; font-size: 24px; font-weight: 600;">
                                You've received a secure message
                            </h1>
                            <p style="margin: 0 0 24px 0; color: #475569; font-size: 16px; line-height: 1.6;">
                                <strong>{{.From}}</strong> has shared a secure message with you using {{.Branding}}.
                            </p>
                            <p style="margin: 0 0 32px 0; color: #475569; font-size: 16px; line-height: 1.6;">
                                Click the button below to access the message. Note that this link can only be used once and will expire after viewing.
                            </p>
                            <table role="presentation" style="margin: 0 0 32px 0;">
                                <tr>
                                    <td style="background: linear-gradient(135deg, #14b8a6 0%, #0d9488 100%); border-radius: 8px;">
                                        <a href="{{.Link}}" style="display: inline-block; padding: 16px 32px; color: #ffffff; text-decoration: none; font-size: 16px; font-weight: 600;">
                                            View Secure Message
                                        </a>
                                    </td>
                                </tr>
                            </table>
                            <p style="margin: 0 0 16px 0; color: #64748b; font-size: 14px; line-height: 1.5;">
                                Or copy and paste this link into your browser:
                            </p>
                            <p style="margin: 0 0 32px 0; color: #14b8a6; font-size: 14px; word-break: break-all;">
                                {{.Link}}
                            </p>
                            <hr style="border: none; border-top: 1px solid #e2e8f0; margin: 32px 0;">
                            <p style="margin: 0; color: #94a3b8; font-size: 12px; line-height: 1.5;">
                                This email was sent by {{.Branding}}. The sender will need to provide you with the PIN separately to access the message.
                            </p>
                        </td>
                    </tr>
                </table>
            </td>
        </tr>
    </table>
</body>
</html>`

// NewEmailSender creates a new email sender with the given configuration
func NewEmailSender(cfg EmailConfig, branding string) (EmailSender, error) {
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
	if cfg.ContentType == "" {
		cfg.ContentType = "text/html"
	}
	if cfg.Charset == "" {
		cfg.Charset = "UTF-8"
	}

	// create notifier
	notifier := notify.NewEmail(notify.SMTPParams{
		Host:        cfg.Host,
		Port:        cfg.Port,
		TLS:         cfg.TLS,
		ContentType: cfg.ContentType,
		Charset:     cfg.Charset,
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

	return &emailSender{
		notifier: notifier,
		cfg:      cfg,
		branding: branding,
		tmpl:     tmpl,
	}, nil
}

// Send sends an email with the secret link
func (e *emailSender) Send(ctx context.Context, req EmailRequest) error {
	// render the email body
	body, err := e.RenderBody(req.Link, req.FromName)
	if err != nil {
		return fmt.Errorf("failed to render email body: %w", err)
	}

	// build the from address with display name
	fromAddr := e.buildFromAddress(req.FromName)

	// build all recipient addresses (To + CC)
	allRecipients := make([]string, 0, len(req.To)+len(req.CC))
	allRecipients = append(allRecipients, req.To...)
	allRecipients = append(allRecipients, req.CC...)

	// build mailto destination with all recipients
	destination := e.buildMailtoDestination(allRecipients, req.Subject, fromAddr)

	// send the email
	if err := e.notifier.Send(ctx, destination, body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// RenderBody renders the email body with the given link and from name
func (e *emailSender) RenderBody(link, fromName string) (string, error) {
	data := struct {
		Link     string
		From     string
		Branding string
	}{
		Link:     link,
		From:     fromName,
		Branding: e.branding,
	}

	var buf bytes.Buffer
	if err := e.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return buf.String(), nil
}

// buildFromAddress builds the From header value with display name
func (e *emailSender) buildFromAddress(displayName string) string {
	// extract email from configured From (might be "Name <email>" or just "email")
	emailAddr := e.extractEmail(e.cfg.From)

	if displayName == "" {
		// use the original From as-is
		return e.cfg.From
	}

	// build new From with custom display name
	return fmt.Sprintf("%q <%s>", displayName, emailAddr)
}

// extractEmail extracts just the email address from a From string
func (e *emailSender) extractEmail(from string) string {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		// assume it's already just an email address
		return from
	}
	return addr.Address
}

// extractDisplayName extracts the display name from configured From address
func (e *emailSender) extractDisplayName() string {
	addr, err := mail.ParseAddress(e.cfg.From)
	if err != nil {
		return ""
	}
	return addr.Name
}

// GetDefaultFromName returns the default from name (from config or branding)
func (e *emailSender) GetDefaultFromName() string {
	name := e.extractDisplayName()
	if name != "" {
		return name
	}
	return e.branding
}

// buildMailtoDestination builds the mailto URL for go-pkgz/notify
func (e *emailSender) buildMailtoDestination(recipients []string, subject, from string) string {
	mailto := "mailto:" + strings.Join(recipients, ",")

	// add query params
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
