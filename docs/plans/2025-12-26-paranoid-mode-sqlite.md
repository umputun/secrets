# Paranoid Mode + SQLite Implementation Plan

**Goal:** Add zero-knowledge encryption mode (`--paranoid`) with client-side AES-128-GCM encryption, and migrate storage from BoltDB to SQLite.

**Architecture:** In paranoid mode, encryption/decryption happens entirely in the browser using Web Crypto API. Server stores pre-encrypted blobs and validates PIN for access control only. SQLite replaces BoltDB for all storage.

**Tech Stack:**
- Go: `modernc.org/sqlite` (pure Go, no CGO)
- Frontend: Web Crypto API (AES-128-GCM), vanilla JavaScript
- IDs: 12-char base62 random strings

---

## Phase 1: SQLite Storage + Short IDs

Replace BoltDB with SQLite and switch to short IDs.

### Task 1.1: Create SQLite Engine with Short IDs

**Files:**
- Create: `app/store/sqlite.go`
- Create: `app/store/sqlite_test.go`
- Modify: `app/store/store.go`

**Schema:**
```sql
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    exp INTEGER NOT NULL,
    data BLOB NOT NULL,
    pin_hash TEXT NOT NULL,
    errors INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_messages_exp ON messages(exp);
```

**Steps:**
1. Add `GenerateID()` to store.go - 12-char base62 using crypto/rand
2. Replace `Key(ts, uuid)` with simple `GenerateID()` call in messager
3. Create SQLite engine implementing Engine interface
4. Add cleanup goroutine: `DELETE FROM messages WHERE exp < ?`
5. Write tests for all operations
6. Run tests: `go test -v ./app/store/...`

### Task 1.2: Remove BoltDB, Update Main

**Files:**
- Modify: `app/main.go`
- Modify: `app/messager/messager.go`
- Delete: `app/store/bolt.go`
- Delete: `app/store/bolt_test.go`

**Steps:**
1. Update main.go: change engine choices to `MEMORY|SQLITE`
2. Update messager to use `store.GenerateID()` instead of `store.Key()`
3. Remove bolt.go and bolt_test.go
4. Run `go mod tidy`
5. Run all tests: `go test ./...`
6. Run linter: `golangci-lint run`

**Commit:** "replace boltdb with sqlite, use short ids"

---

## Phase 2: Paranoid Mode Flag

Add `--paranoid` flag and wire it through the system.

### Task 2.1: Add Flag and Expose in API

**Files:**
- Modify: `app/main.go`
- Modify: `app/server/server.go`
- Modify: `app/server/server_test.go`

**Steps:**
1. Add `Paranoid bool` to opts, server.Config
2. Add `Paranoid bool` to params endpoint response
3. Write tests for params endpoint
4. Run tests: `go test -v ./app/server/...`
5. Run linter

**Commit:** "add --paranoid flag"

---

## Phase 3: Server-Side Paranoid Logic

Skip server encryption/decryption in paranoid mode.

### Task 3.1: Messager Paranoid Mode

**Files:**
- Modify: `app/messager/messager.go`
- Modify: `app/messager/messager_test.go`
- Modify: `app/main.go`

**Steps:**
1. Add `Paranoid bool` to messager.Params, pass from main.go
2. In `MakeMessage`: if paranoid, store data as-is (skip encrypt)
3. In `LoadMessage`: if paranoid, return data as-is (skip decrypt)
4. Keep PIN validation (access control still works)
5. Same for `MakeFileMessage` - skip encryption if paranoid
6. Write tests for paranoid mode behavior
7. Run tests: `go test -v ./app/messager/...`
8. Run linter

**Commit:** "skip server-side crypto in paranoid mode"

---

## Phase 4: Client-Side Crypto

Create JavaScript crypto module.

### Task 4.1: Create Crypto Module

**Files:**
- Create: `app/server/assets/static/js/crypto.js`

**Functions:**
- `generateKey()` → 22-char base64url (128-bit)
- `encrypt(plaintext, key)` → base64url ciphertext
- `decrypt(ciphertext, key)` → plaintext
- `encryptFile(data, filename, contentType, key)` → base64url
- `decryptFile(ciphertext, key)` → {filename, contentType, data}

**Steps:**
1. Implement all crypto functions using Web Crypto API
2. Test in browser with manual HTML page
3. Verify round-trip encryption works

**Commit:** "add client-side crypto module"

---

## Phase 5: Frontend Create Flow

Integrate crypto into create forms.

### Task 5.1: Update Create Forms

**Files:**
- Modify: `app/server/assets/html/pages/index.tmpl.html`
- Modify: `app/server/assets/html/partials/secure-link.tmpl.html`
- Modify: `app/server/web.go`

**Steps:**
1. Pass Paranoid to templates
2. Include crypto.js when paranoid
3. Intercept form submit: encrypt message/file before POST
4. After success: append `#key` to displayed URL
5. Update copy button to include fragment
6. Manual test both text and file upload

**Commit:** "add client-side encryption to create forms"

---

## Phase 6: Frontend Reveal Flow

Integrate crypto into reveal page.

### Task 6.1: Update Reveal Page

**Files:**
- Modify: `app/server/assets/html/pages/show-message.tmpl.html`
- Modify: `app/server/assets/html/partials/decoded-message.tmpl.html`

**Steps:**
1. Include crypto.js when paranoid
2. Read key from `window.location.hash`
3. After PIN validation: decrypt response in browser
4. Display decrypted content (text or file download)
5. Handle errors gracefully
6. Manual test full flow

**Commit:** "add client-side decryption to reveal page"

---

## Phase 7: E2E Tests

Add paranoid mode e2e tests.

### Task 7.1: Paranoid Mode Tests

**Files:**
- Modify: `e2e/tests/secrets.spec.ts`

**Tests:**
- Create text secret → URL has fragment
- Retrieve text secret → decryption works
- Wrong PIN rejected
- Create/retrieve file works
- One-time read enforced

**Steps:**
1. Add paranoid mode test cases
2. Run: `make e2e`
3. All tests pass

**Commit:** "add paranoid mode e2e tests"

---

## Phase 8: Documentation

Update docs.

### Task 8.1: Update README and CLAUDE.md

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Steps:**
1. Document `--paranoid` flag and zero-knowledge mode
2. Update storage engine options
3. Final test run and linter
4. Format code

**Commit:** "update documentation"

---

## Final Checklist

- [ ] SQLite storage works
- [ ] Short IDs (12 chars) used everywhere
- [ ] `--paranoid` flag works
- [ ] Server skips crypto in paranoid mode
- [ ] PIN still enforces access control
- [ ] Client-side crypto works
- [ ] URLs include `#key` fragment in paranoid mode
- [ ] File upload/download works in paranoid mode
- [ ] E2E tests pass
- [ ] Docs updated
- [ ] Linter clean
