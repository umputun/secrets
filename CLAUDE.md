# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Safesecret is a Go-based web service for sharing sensitive information securely. It encrypts messages with a PIN, stores them temporarily, and allows one-time retrieval. The service can use either in-memory or BoltDB storage engines.

## Build and Development Commands

### Run the application
```bash
# Build and run locally
cd app && go build -o secrets && ./secrets --key=<SIGN_KEY> --domain=localhost --protocol=http

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
    LoadMessage(key, pin string) (msg *store.Message, err error)
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
- `GET /api/v1/ping` - Health check

## Configuration

Key configuration via environment variables or flags:
- `SIGN_KEY` - Encryption signing key (required)
- `ENGINE` - Storage engine: MEMORY or BOLT (default: MEMORY)
- `MAX_EXPIRE` - Maximum message lifetime (default: 24h)
- `PIN_SIZE` - PIN length in characters (default: 5)
- `PIN_ATTEMPTS` - Max failed PIN attempts (default: 3)
- `DOMAIN` - Service domain (required)
- `PROTOCOL` - http or https (default: https)

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