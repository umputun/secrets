# Changelog

All notable changes to this project are documented in this file.

## [2.2.3] - 2025-12-30

### Fixed
- Improve card header layout with proper alignment of title and mode toggle
- Replace iMessage with Signal in secure messenger examples

## [2.2.2] - 2025-12-29

### Fixed
- Show friendly 404 error page for non-existent messages instead of confusing PIN form

## [2.2.1] - 2025-12-29

### Fixed
- Add version-based cache busting for static assets to prevent browser cache issues after deployments

## [2.2.0] - 2025-12-29

### Added
- Optional PIN protection for secrets (#98)
  - Users can skip PIN for convenience while maintaining encryption
  - Secrets without PIN are directly accessible via link
- Ciphertext format validation for UI routes (#97)

### Changed
- Bump modernc.org/sqlite from 1.41.0 to 1.42.2 (#99)

## [2.1.0] - 2025-12-28

### Changed
- Replace paranoid mode with hybrid encryption (#96)
- UI always uses client-side AES-128-GCM encryption, API uses server-side encryption
- Add RequireHTMX middleware to ensure JavaScript for UI encryption
- Add security headers middleware (CSP, X-Frame-Options, HSTS)

## [2.0.0] - 2025-12-26

### Added
- SQLite storage engine as alternative to BoltDB and in-memory (#96)
- Paranoid mode for zero-knowledge client-side AES-128-GCM encryption
  - Server stores only encrypted blobs, never sees plaintext
  - Encryption key derived from PIN, never transmitted to server
  - Visual indicator with shield icon when paranoid mode active
- Playwright E2E test suite for paranoid mode

### Changed
- **BREAKING**: Replace BoltDB with SQLite for persistent storage
  - `--engine=BOLT` option removed
  - Existing BoltDB databases not migrated automatically
- Module path changed to `github.com/umputun/secrets/v2`

## [1.9.3] - 2025-12-22

### Changed
- Update go-pkgz/rest to v1.20.6 for CDN-compatible RealIP middleware

## [1.9.2] - 2025-12-22

### Added
- Playwright E2E test suite for web UI (#93)
- Improved audit logging for security and usage analytics (#94)

## [1.9.1] - 2025-12-18

### Fixed
- Security review findings in server package

## [1.9.0] - 2025-12-18

### Added
- Email sharing for secure links (#92)
  - SMTP configuration with TLS/STARTTLS support
  - Custom email templates
  - Rate limiting on email sending

## [1.8.1] - 2025-12-16

### Added
- GoReleaser for binary releases
- Comprehensive SEO improvements

### Fixed
- CI workflow for proper release builds
- Digits-only input validation for PIN fields

## [1.8.0] - 2025-12-16

### Changed
- Complete frontend redesign with modern dark theme (#91)

## [1.7.1] - 2025-12-16

### Fixed
- Content-Type header not being set for JSON responses (#90)

## [1.7.0] - 2025-12-16

### Added
- Optional authentication for link generation (#89)
  - bcrypt password hash configuration
  - Session-based authentication with configurable TTL
- Encrypted file upload and download support (#88)
  - Configurable max file size
  - Content-type preservation
  - One-time download with PIN protection

## [1.6.3] - 2025-10-18

### Fixed
- Version detection in Docker builds using baseimage script

## [1.6.2] - 2025-10-18

### Added
- Multiple domains support via comma-separated list (#79)
- Comprehensive SEO improvements (#76)
  - Open Graph and Twitter Card meta tags
  - JSON-LD structured data
  - Sitemap and robots.txt

### Changed
- Replaced purple/indigo color scheme with professional blue palette
- Simplified theme toggle to two states (light/dark) with system preference detection

### Fixed
- Graceful server shutdown with 10-second timeout
- IPv6 URL generation with proper bracketing

## [1.6.1] - 2025-08-26

### Fixed
- Regression on ping endpoints (#72)

## [1.6.0] - 2025-08-21

### Changed
- Migrate from chi router to go-pkgz/routegroup for Go stdlib alignment
- Update tollbooth to v8 for improved rate limiting
- Enhanced logging for message access events
- Remove pkg/errors dependency in favor of standard library
- Routes use Go 1.22+ patterns with method prefixes

## [1.5.1] - 2025-08-20

### Fixed
- PIN_SIZE configuration not working in UI (#70)
  - Dynamic PIN length in web interface
  - CLI flag typo: `--pinszie` → `--pinsize`
- Pluralization in duration display (#69)
  - Correctly shows singular forms for single units

## [1.5.0] - 2025-08-19

### Added
- Customizable branding via `--branding` parameter
  - Allows company-specific application title
  - Support for environment variable (`BRANDING`)

## [1.4.0] - 2025-08-19

### Added
- Complete UI redesign with modern card-based layout
- Light/dark/auto theme support with persistent cookies
- HTMX v2 for dynamic interactions without JavaScript
- Copy feedback with server-side popups
- Copyright footer with dynamic year

### Changed
- Replaced emoji icons with accessible SVG icons
- Improved mobile responsiveness
- Enhanced accessibility with prefers-reduced-motion support
- Removed all Bootstrap dependencies
- Updated to Go 1.24
- Configured Dependabot for security updates only

### Fixed
- HTMX error handling for invalid PIN entry
- XSS vulnerability in copy feedback
- Embedded static files serving
- Theme toggle icon sizing and positioning
- Improved test coverage from 40.7% to 82.7%

## [1.3.0] - 2024-09-16

### Changed
- Code refactor for configurable protocol (#40, #41, #43)
- Updated Go version and dependencies (#44)
- Pin golangci-lint version (#37)

### Added
- Dependabot updates for GitHub Actions and Go modules (#36)
- Integrations section in README (#35)

## [1.2.4] - 2024-01-13

### Changed
- Bump dependencies

## [1.2.0] - 2023-10-08

### Changed
- Replace separate UI with embedded HTMX-based interface (#31, #32, #33)
- Remove old frontend

### Added
- Improved test coverage for messager (#30)

## [1.1.0] - 2021-03-16

### Changed
- Update JavaScript dependencies
- Remove external nacl and rewriter middleware
- Switch to current Go version

## [1.0.0] - 2020-02-21

Initial release.

### Added
- REST API for encrypted message storage and retrieval
- Web UI for message creation
- PIN-protected message access with attempt limiting
- Configurable message expiration
- In-memory and BoltDB storage engines
- Docker deployment support
