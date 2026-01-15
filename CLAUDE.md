# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Safesecret is a Go-based web service for sharing sensitive information securely. It encrypts messages with a PIN, stores them temporarily, and allows one-time retrieval. The service uses SQLite for storage (both in-memory and persistent modes).

**Hybrid Encryption:** Route-based encryption mode where UI (web browser) always uses zero-knowledge client-side encryption (Web Crypto API AES-128-GCM), while API always uses server-side encryption. This provides maximum security for interactive users while maintaining API simplicity.

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
  - SQLite-based implementation (both in-memory and file-backed)
  - Short 12-char base62 IDs with unbiased rejection sampling
- **app/server/assets/** - Frontend assets embedded via Go 1.16+ embed
  - Static HTML/CSS/JS for web interface
  - Templates for dynamic content rendering

### Key Interfaces

**Messager** (app/server/server.go):
```go
type Messager interface {
    MakeMessage(req messager.MsgReq) (result *store.Message, err error)
    LoadMessage(key, pin string) (msg *store.Message, err error)
    IsFile(key string) bool // checks if message is a file without decrypting (false for ClientEnc)
}
```

**Engine** (app/messager/messeger.go):
```go
type Engine interface {
    Save(ctx context.Context, msg *store.Message) (err error)
    Load(ctx context.Context, key string) (result *store.Message, err error)
    IncErr(ctx context.Context, key string) (count int, err error)
    Remove(ctx context.Context, key string) (err error)
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

### File Message Format
File messages use a distinct storage format with encrypted metadata:
- **Stored format**: `!!FILE!!<encrypted blob>`
- **Encrypted blob contains**: `filename!!content-type!!\n<binary data>`
- Only the `!!FILE!!` prefix is unencrypted (used to detect file vs text messages)
- Filename and content-type are encrypted together with file content for privacy

**Validation requirements** (to prevent header parsing issues):
- Filename: no `!!`, `\n`, `\r`, `\x00`, `/`, `\`, `..`, control chars; max 255 chars
- Content-type: no `!!`, `\n`, `\r`, `\x00`

**Key functions**:
- `IsFileMessage(data)` - checks for `!!FILE!!` prefix
- `ParseFileHeader(data)` - extracts filename, content-type, data start position (4KB scan limit)
- `IsFile(key)` - loads message to check type without decrypting (used for UI to show "Download" vs "Reveal")

### Hybrid Encryption Architecture

Route-based encryption where UI uses client-side encryption and API uses server-side encryption:

**Encryption modes:**
- **UI routes** (`/generate-link`): Always client-side encryption (`ClientEnc=true`)
- **API routes** (`/api/v1/message`): Always server-side encryption (`ClientEnc=false`)
- The `ClientEnc` field in `store.Message` tracks which mode was used

**Client-side encryption (UI):**
- Client generates 128-bit AES-GCM key per message (22-char base64url)
- Key stored only in URL fragment (`#key`) - never sent to server
- Encryption/decryption happens entirely in browser via Web Crypto API
- Server stores opaque encrypted blobs, validates PIN via bcrypt hash
- RequireHTMX middleware ensures JavaScript is present (prevents plaintext storage)

**Payload format (client-encrypted, before encryption):**
- Text: `0x00 || utf8(plaintext)`
- File: `0x01 || len_be16(filename) || filename || len_be16(contentType) || contentType || data`

**Ciphertext format:** `base64url(IV[12] || encrypted || tag[16])`

**Implementation details:**
- `msg.ClientEnc` field determines encryption/decryption path on retrieval
- `IsFile()` returns false for `ClientEnc=true` messages (server cannot inspect content)
- Size limits adjusted: `MaxFileSize * 1.4` for base64 overhead (UI route)
- Client-side crypto module: `app/server/assets/static/js/crypto.js`

### Optional PIN Architecture

When `--allow-no-pin` / `ALLOW_NO_PIN` is enabled, secrets can be created without PIN protection for use cases where the sharing channel is already secure (Signal or other E2E encrypted messengers).

**Feature control:**
- Operator-controlled flag (default: false)
- Only affects web UI routes - API always requires PIN
- `AllowNoPin` in server.Config, passed to templates via `templateData.AllowNoPin`

**Creation flow (UI only):**
1. User leaves PIN field empty during secret creation
2. JavaScript detects empty PIN, shows confirmation modal
3. On confirm: proceeds with client-side encryption, stores empty `PinHash`
4. On cancel: refocuses PIN field

**Storage:**
- `PinHash` stored as empty string (`""`) for PIN-less messages
- No schema changes - empty string is valid
- `MsgReq.AllowEmptyPin` flag controls whether empty PIN is accepted

**Retrieval flow:**
- `HasPin(key)` checks if message requires PIN (`msg.PinHash != ""`)
- UI adapts: PIN-less shows "Reveal Secret" button, with-PIN shows decrypt form
- `checkHash()` returns true when both stored hash and provided PIN are empty

**Key methods:**
- `HasPin(ctx, key) (bool, error)` - checks if message requires PIN
- `checkHash(msg, pin)` - returns true when both are empty
- `validatePIN(pin, pinValues, pinSize) error` - validates PIN format

### Middleware Architecture
- **rest.RealIP middleware** - Extracts client IP from headers for CDN/proxy compatibility (from go-pkgz/rest v1.20.6+)
  - Header priority: X-Real-IP → CF-Connecting-IP → leftmost public IP in X-Forwarded-For → RemoteAddr
  - Filters private/loopback/link-local IPs automatically
- **HashedIP middleware** - Anonymizes client IP using HMAC-SHA1 hash (12-char hex) for audit logging
  - Must run after rest.RealIP middleware (reads `r.RemoteAddr` set by RealIP)
- **Logger middleware** - Logs requests with masked sensitive paths (PINs) and anonymized IPs
  - Must run after HashedIP middleware (reads hashed IP from context)
- **Middleware chain order matters**: rest.RealIP → HashedIP → Logger

### API Endpoints
- `POST /api/v1/message` - Create encrypted message
- `GET /api/v1/message/:key/:pin` - Retrieve and decrypt message
- `GET /api/v1/params` - Get service configuration
- `GET /ping` or `GET /api/v1/ping` - Health check

## Configuration

Key configuration via environment variables or flags:
- `SIGN_KEY` - Encryption signing key (required)
- `ENGINE` - Storage engine: MEMORY or SQLITE (default: MEMORY)
- `SQLITE_FILE` - SQLite database file path (default: /data/secrets.db in Docker, /tmp/secrets.db otherwise)
- `MAX_EXPIRE` - Maximum message lifetime (default: 24h)
- `PIN_SIZE` - PIN length in characters (default: 5)
- `PIN_ATTEMPTS` - Max failed PIN attempts (default: 3)
- `ALLOW_NO_PIN` - Allow creating secrets without PIN protection (default: false)
- `DOMAIN` - Allowed domain(s), supports comma-separated list (e.g., "example.com,alt.example.com")
- `PROTOCOL` - http or https (default: https)
- `LISTEN` - Server listen address, ip:port or :port format (default: :8080)
- `FILES_ENABLED` - Enable file uploads (default: false)
- `FILES_MAX_SIZE` - Maximum file size in bytes (default: 1MB)
- `AUTH_HASH` - bcrypt hash of password (enables auth for link generation if set)
- `AUTH_SESSION_TTL` - Session lifetime (default: 168h / 7 days)

## Testing Approach

- Table-driven tests using testify/assert
- Mock generation with moq for interfaces (see go:generate directives)
- Test files follow naming convention: `*_test.go` alongside source files
- Integration tests for storage engines with cleanup

## Dependencies

Core libraries used:
- `github.com/go-chi/chi/v5` - HTTP routing
- `github.com/go-pkgz/lgr` - Structured logging
- `github.com/go-pkgz/rest` - REST middleware utilities (v1.20.6+ for CDN-compatible RealIP)
- `modernc.org/sqlite` - Pure Go SQLite storage engine
- `golang.org/x/crypto` - Encryption and bcrypt hashing
- `github.com/stretchr/testify` - Testing assertions
- `github.com/didip/tollbooth` - Rate limiting

## Frontend Architecture

### Template System
- Go html/template with modular structure: base template (`index.tmpl.html`) + named blocks
- Partials in `app/server/assets/html/partials/` for reusable components (`decoded-message.tmpl.html`, `secure-link.tmpl.html`)
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
- JavaScript: External `app.js` file for all application logic (CSP-compliant, no inline scripts)
- Icons: Inline SVG for consistency and theming
- Fonts: Google Fonts integration with preconnect optimization

### Local Development Configuration
- Server requires explicit local config: `--domain=localhost:8080 --protocol=http`
- Default protocol is HTTPS, must override for local development
- Link generation uses request domain if allowed, falls back to first configured domain
- Multiple domains supported via comma-separated list: `--domain="example.com,alt.example.com"`
- **Embedded assets require rebuild**: UI assets (CSS, JS, templates) are embedded via `//go:embed`. Changes to `app/server/assets/` files require rebuilding and restarting the server. After killing the server, verify it stopped with `curl http://localhost:8080/ping` before restarting.

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

### JavaScript Architecture (CSP-Compliant)
- All JavaScript in external `app/server/assets/static/js/app.js` file
- No inline scripts or `hx-on::` handlers (enables strict CSP with `script-src 'self'`)
- Event delegation pattern with data attributes (`data-action`, `data-numeric-only`, etc.)
- MutationObserver for `data-autofocus` on dynamically loaded content
- Script tags: HTMX core, response-targets extension, crypto.js, app.js

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

### Dynamic Content Patterns
- **Autofocus on HTMX-loaded content**: `autofocus` attribute doesn't work on dynamically loaded content. Add `data-autofocus` attribute to the element - app.js uses MutationObserver to detect and focus these elements automatically.
- **Form state preservation after popup**: `HX-Refresh: true` causes full page reload, losing form data. Better pattern: return `HX-Trigger: eventName` header + on form use `hx-trigger="submit, eventName from:body"` to re-submit with preserved data (works with multipart file uploads too).
- **HX-Trigger event targeting**: Events from `HX-Trigger` header dispatch to the *target* element. To catch on other elements, use `hx-trigger="eventName from:body"` syntax.
- **Event handlers via data attributes**: Use `data-action="action-name"` instead of inline onclick/hx-on:: handlers. app.js handles these via event delegation.

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

## Email/SMTP Configuration

### Mailgun SMTP Settings
- **Use port 465 with implicit TLS** (`--email.tls`), NOT port 587 with STARTTLS
- Mailgun requires IP allowlisting for SMTP authentication (implemented April 2024)
- Authentication failures (535) often indicate IP not in allowlist, not wrong credentials
- Test SMTP credentials via Mailgun HTTP API first: `curl --user 'user:pass' https://api.mailgun.net/v3/domain/messages`

### Email Feature Flags
- `--email.enabled` - enables email sharing feature
- `--email.host`, `--email.port`, `--email.username`, `--email.password` - SMTP server config
- `--email.tls` - use implicit TLS (port 465), `--email.starttls` - use STARTTLS (port 587)
- `--email.from` - sender address with display name format: `"Name <email@domain>"`

## Testing Patterns

### Local Testing Workflow
- Port conflicts common during testing (8080 often in use)
- Tests may fail locally if port is occupied but pass in CI
- Build binary embeds UI assets, no need for `--web` flag after building
- Run formatter, goimports, and unfuck-ai-comments before committing
- **AUTH_HASH in environment**: When testing without auth, use `--auth.hash=""` explicitly - the env may have AUTH_HASH set which overrides the default

### E2E Tests
- Located in `e2e/` directory, require build tag: `go test -tags=e2e ./e2e/...`
- Use Playwright for browser automation
- **Timeout**: Use 120 seconds max (`timeout 120 go test -tags=e2e ./e2e/...`)
- **Failfast**: Use `-failfast` to stop on first failure (`go test -failfast -tags=e2e ./e2e/...`)
- Run headless by default, set `E2E_HEADLESS=false` for UI debugging

## Deployment Process

### Beta Deployment (beta.safesecret.info)
- Master image deploys to beta after CI passes
- Server: ssh to eclipse-love.exe.xyz
- Deploy commands:
  ```bash
  cd /srv
  docker compose pull
  docker compose up -d
  ```
- Verify deployment: `curl -I https://beta.safesecret.info/ping` and check `App-Version` header