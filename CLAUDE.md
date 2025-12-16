# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Safesecret is a Go-based web service for sharing sensitive information securely. It encrypts messages with a PIN, stores them temporarily, and allows one-time retrieval. The service can use either in-memory or BoltDB storage engines.

## Build and Development Commands

### Run the application
```bash
# Build and run locally (single domain)
cd app && go build -o secrets && ./secrets --key=<SIGN_KEY> --domain=localhost --protocol=http

# Build and run locally (multiple domains)
cd app && go build -o secrets && ./secrets --key=<SIGN_KEY> --domain="localhost,127.0.0.1" --protocol=http

# Run with Docker (development)
docker-compose -f docker-compose-dev.yml up
```

### Testing
```bash
# Run all tests
go test -v -timeout=60s -covermode=count -coverprofile=coverage.out ./...

# Run tests with race detection
go test -race ./...

# Run specific test
go test -run TestMessageProc ./app/messager

# Generate coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

### Linting and Formatting
```bash
# Run linter (from project root)
golangci-lint run

# Format code
gofmt -s -w $(find . -type f -name "*.go" -not -path "./vendor/*")

# Run goimports
goimports -w $(find . -type f -name "*.go" -not -path "./vendor/*")
```

## Architecture

### Package Structure
- **app/main.go** - Entry point, initializes storage engine, crypter, and starts server
- **app/messager/** - Core business logic for message encryption/decryption and storage operations
  - Uses injected `Engine` interface for storage abstraction
  - Encrypts messages with user-provided PIN (hashed with bcrypt)
  - Implements rate limiting on PIN attempts
- **app/server/** - HTTP REST API and web UI serving
  - Handles routing with chi/v5
  - Implements middleware for rate limiting, logging, and request validation
  - Serves both API endpoints and web UI templates
- **app/store/** - Storage layer implementations
  - `Engine` interface with `Save`, `Load`, `IncErr`, and `Remove` methods
  - Two implementations: InMemory (with TTL cleanup) and BoltDB
- **ui/** - Frontend assets embedded via Go 1.16+ embed
  - Static HTML/CSS/JS for web interface
  - Templates for dynamic content rendering

### Key Interfaces

**Messager** (app/server/server.go):
```go
type Messager interface {
    MakeMessage(duration time.Duration, msg, pin string) (result *store.Message, err error)
    MakeFileMessage(req messager.FileRequest) (result *store.Message, err error)
    LoadMessage(key, pin string) (msg *store.Message, err error)
    IsFile(key string) bool // checks if message is a file without decrypting
}
```

**Engine** (app/messager/messeger.go):
```go
type Engine interface {
    Save(msg *store.Message) (err error)
    Load(key string) (result *store.Message, err error)
    IncErr(key string) (count int, err error)
    Remove(key string) (err error)
}
```

**Crypter** (app/messager/messeger.go):
```go
type Crypter interface {
    Encrypt(req Request) (result []byte, err error)
    Decrypt(req Request) (result []byte, err error)
}
```

### Data Flow
1. User submits message + PIN via web UI or API
2. Server validates input (PIN size, expiration limits)
3. MessageProc hashes PIN with bcrypt, encrypts message data
4. Encrypted message saved to storage engine with UUID key and expiration
5. For retrieval: validate PIN attempts, decrypt if correct, delete after successful read

### API Endpoints
- `POST /api/v1/message` - Create encrypted message
- `GET /api/v1/message/:key/:pin` - Retrieve and decrypt message
- `GET /api/v1/params` - Get service configuration
- `GET /ping` or `GET /api/v1/ping` - Health check

## Configuration

Key configuration via environment variables or flags:
- `SIGN_KEY` - Encryption signing key (required)
- `ENGINE` - Storage engine: MEMORY or BOLT (default: MEMORY)
- `MAX_EXPIRE` - Maximum message lifetime (default: 24h)
- `PIN_SIZE` - PIN length in characters (default: 5)
- `PIN_ATTEMPTS` - Max failed PIN attempts (default: 3)
- `DOMAIN` - Allowed domain(s), supports comma-separated list (e.g., "example.com,alt.example.com")
- `PROTOCOL` - http or https (default: https)
- `FILES_ENABLED` - Enable file uploads (default: false)
- `FILES_MAX_SIZE` - Maximum file size in bytes (default: 1MB)

## Testing Approach

- Table-driven tests using testify/assert
- Mock generation with moq for interfaces (see go:generate directives)
- Test files follow naming convention: `*_test.go` alongside source files
- Integration tests for storage engines with cleanup

## Dependencies

Core libraries used:
- `github.com/go-chi/chi/v5` - HTTP routing
- `github.com/go-pkgz/lgr` - Structured logging
- `github.com/go-pkgz/rest` - REST middleware utilities
- `go.etcd.io/bbolt` - BoltDB storage engine
- `golang.org/x/crypto` - Encryption and bcrypt hashing
- `github.com/stretchr/testify` - Testing assertions
- `github.com/didip/tollbooth` - Rate limiting

## Frontend Architecture

### Template System
- Go html/template with modular structure: base template (`index.tmpl.html`) + named blocks
- Partials in `ui/html/partials/` for reusable components (`decoded-message.tmpl.html`, `secure-link.tmpl.html`)
- HTMX integration for dynamic updates without page reloads
- Template inheritance pattern with `{{define "base"}}` and `{{template "main" .}}`

### Design System
- CSS custom properties-based design system with comprehensive theming
- 8px grid spacing system: `--spacing-xs` (4px) through `--spacing-5xl` (96px)
- Typography scale with Inter (UI) and Poppins (branding) fonts
- Light/dark theme support via CSS custom properties and `[data-theme]` selectors
- SVG icon system with `currentColor` for automatic theme adaptation

### Static Assets Organization
- CSS: Design system approach with modular sections (variables, typography, components)
- JavaScript: Feature-specific modules (copy-text.js, theme.js, pin.js)
- Icons: Inline SVG for consistency and theming
- Fonts: Google Fonts integration with preconnect optimization

### Local Development Configuration
- Server requires explicit local config: `--domain=localhost:8080 --protocol=http`
- Default protocol is HTTPS, must override for local development
- Link generation uses request domain if allowed, falls back to first configured domain
- Multiple domains supported via comma-separated list: `--domain="example.com,alt.example.com"`

## HTMX Implementation

### Version and Extensions
- HTMX v2.0.3 with response-targets extension for proper error handling
- Server-driven UI pattern with minimal client-side JavaScript
- Uses `hx-target-[status]` attributes for HTTP status-specific targeting (400, 403, 404, 500)

### Form Handling Pattern
- Forms use `hx-post` with `hx-target="#form-card"` and `hx-swap="outerHTML"` to replace entire form with result
- Error handling via response-targets: `hx-target-400="#form-errors"` for validation errors
- Loading indicators with `hx-indicator="#form-spinner"` for visual feedback during requests

### Theme System
- Server-side theme switching via POST /theme endpoint
- Theme stored in browser cookies (1-year expiry) for stateless per-user preferences
- Theme cycling: light → dark → auto → light
- Uses `HX-Refresh: true` header to trigger full page reload on theme change
- Template rendering passes theme to all views via `getTheme(r)` helper

### JavaScript Minimization
- All external JS files eliminated in favor of inline code
- Clipboard functionality using `hx-on::before-request` with native Clipboard API
- Popup handling with minimal inline event listeners (closePopup event, backdrop clicks)
- Only 3 script tags remain: HTMX core, response-targets extension, inline popup handlers

### Web Endpoints Structure
- Web UI endpoints grouped separately from API endpoints in router
- Web handlers in server/web.go, API handlers in server/server.go
- Template-specific controllers suffix: `*ViewCtrl` for pages, `*Ctrl` for actions
- All web controllers pass CurrentYear and Theme to templates for consistent rendering

### HTTP Status Code Conventions
- 200: Successful operations and normal page renders
- 400: Validation errors (triggers error display in form)
- 403: Authentication failures (wrong PIN)
- 404: Resource not found (expired/missing messages)
- HTMX requests return appropriate status codes, non-HTMX fallback to 200 with error template

### Template Data Pattern
- `templateData` struct wraps all template variables with consistent fields:
  - Form: Contains form-specific data and validation
  - PinSize: Configuration for PIN input rendering
  - CurrentYear: For copyright footer
  - Theme: Current user theme preference

## CI/CD Pipeline

### GitHub Actions Workflow
- CI runs on every push and pull request using `.github/workflows/ci.yml`
- Tests execute with 60s timeout and coverage reporting
- GolangCI-Lint v2.3.1 enforces code quality standards
- Docker images automatically built and pushed to DockerHub on master and tagged releases
- Tagged releases deploy as both `umputun/secrets:vX.Y.Z` and `umputun/secrets:latest`

### Release Process
- Hotfix releases follow semantic versioning (v1.5.0 → v1.5.1 for bug fixes)
- GitHub releases titled as "Version X.Y.Z" (no "v" prefix in title)
- Release notes should avoid emojis in GitHub communications
- Docker images available immediately after tagged release CI completion

## Template Functions

### Custom Template Functions
- `until(n int)`: Generates slice of integers from 0 to n-1 for iteration
- `add(a, b int)`: Addition function for template arithmetic
- Template FuncMap defined in `app/server/web.go` during template parsing
- Used for dynamic content generation based on configuration (e.g., PIN_SIZE)

## Configuration Nuances

### CLI Flags vs Environment Variables
- Both CLI flags and environment variables supported for all configuration
- CLI flag names use lowercase with no underscores (e.g., `--pinsize`)
- Environment variables use uppercase with underscores (e.g., `PIN_SIZE`)
- Some historical typos may exist in CLI flags but environment variables are reliable

## Testing Patterns

### Local Testing Workflow
- Port conflicts common during testing (8080 often in use)
- Tests may fail locally if port is occupied but pass in CI
- Build binary embeds UI assets, no need for `--web` flag after building
- Run formatter, goimports, and unfuck-ai-comments before committing