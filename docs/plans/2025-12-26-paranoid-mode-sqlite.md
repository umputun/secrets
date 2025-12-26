# Paranoid Mode + SQLite Implementation Plan

**Goal:** Add zero-knowledge encryption mode (`--paranoid`) with client-side AES-128-GCM encryption, and replace BoltDB with SQLite storage.

**Architecture:** In paranoid mode, encryption/decryption happens entirely in the browser using Web Crypto API. Server stores opaque pre-encrypted blobs and validates PIN for access control only. Server cannot distinguish text from files - all content is opaque. SQLite replaces BoltDB for all storage.

**Tech Stack:**
- Go: `modernc.org/sqlite` (pure Go, no CGO)
- Frontend: Web Crypto API (AES-128-GCM), vanilla JavaScript
- IDs: 12-char base62 random strings

**Crypto Format (AES-128-GCM):**
```
Ciphertext = base64url(IV || encrypted || tag)
- IV: 12 bytes from crypto.getRandomValues()
- Tag: 16 bytes (built into Web Crypto GCM output)
- Key: 128-bit, encoded as 22-char base64url
```

**File Handling in Paranoid Mode:**
- No server-side file detection - all content is opaque blob
- File metadata (filename, content-type) encrypted inside payload
- Client determines content type after decryption using magic prefix
- File format: `encrypt("!!F!!" + filename + "!!" + contentType + "!!\n" + binaryData)`
- Text format: `encrypt(plaintext)` - no prefix
- Detection: after decryption, check for `!!F!!` prefix to distinguish file from text

**Transport in Paranoid Mode:**
- All uploads (text and files) use same endpoint with base64 blob body
- No multipart form data - filename/content-type hidden from server
- Server stores/returns opaque blob without interpretation

**Server Response in Paranoid Mode:**
- Reveal page embeds encrypted blob in HTML: `<div data-encrypted="...base64...">`
- API returns same JSON structure: `{"key": "...", "message": "...base64..."}` - field unchanged, content is encrypted
- Client JS decrypts and renders (text) or triggers download (file)

**Size Limits:**
- Current server uses `rest.SizeLimit`: 64KB text-only, MaxFileSize+10KB with files
- Paranoid mode: base64 adds ~33% overhead, adjust limits accordingly
- In paranoid mode, use MaxFileSize * 1.4 as request size limit (covers base64 expansion)
- Client-side: enforce original content size before encryption

**EnableFiles in Paranoid Mode:**
- Server cannot distinguish text from files - content type enforcement impossible
- `EnableFiles=false` hides file upload UI only (client-side enforcement)
- This is inherent to zero-knowledge: you can't restrict content types of encrypted blobs

**Missing Fragment Warning:**
- Client checks `location.hash` on reveal page load
- If empty in paranoid mode, show warning before PIN entry
- Prevents accidental deletion when URL fragment stripped

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

Skip server encryption/decryption in paranoid mode. All content treated as opaque blob.

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
5. In paranoid mode, `MakeFileMessage` not used - all content goes through `MakeMessage` as opaque blob
6. `IsFile()` returns false in paranoid mode (server doesn't know content type)
7. Write tests for paranoid mode behavior
8. Run tests: `go test -v ./app/messager/...`
9. Run linter

### Task 3.2: Adjust Size Limits for Paranoid Mode

**Files:**
- Modify: `app/server/server.go`

**Steps:**
1. In routes(), if paranoid mode: set sizeLimit to MaxFileSize * 1.4 (covers base64 overhead)
2. This applies to all requests since server can't distinguish text from files
3. Write test for size limit adjustment
4. Run tests

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

Integrate crypto into create forms. In paranoid mode, all content (text and files) encrypted client-side and sent as single blob.

### Task 5.1: Update Create Forms

**Files:**
- Modify: `app/server/assets/html/pages/index.tmpl.html`
- Modify: `app/server/assets/html/partials/secure-link.tmpl.html`
- Modify: `app/server/web.go`

**Steps:**
1. Pass Paranoid to templates
2. Include crypto.js when paranoid
3. For text: encrypt plaintext, POST as base64 blob
4. For files: encrypt with `!!F!!` prefix + metadata, POST as base64 blob (not multipart)
5. Both use same form field - server sees opaque blob either way
6. After success: append `#key` to displayed URL
7. Update copy button to include fragment
8. Manual test both text and file

**Commit:** "add client-side encryption to create forms"

---

## Phase 6: Frontend Reveal Flow

Integrate crypto into reveal page. Server returns encrypted blob, client decrypts.

### Task 6.1: Update Server Response for Paranoid Mode

**Files:**
- Modify: `app/server/web.go`
- Modify: `app/server/server.go`

**Steps:**
1. In paranoid mode, reveal page returns encrypted blob in `data-encrypted` attribute
2. API returns same JSON: `{"key": "...", "message": "..."}` - content is encrypted, structure unchanged
3. Server doesn't render/parse message content - passes through as-is

### Task 6.2: Update Reveal Page Client-Side

**Files:**
- Modify: `app/server/assets/html/pages/show-message.tmpl.html`
- Modify: `app/server/assets/html/partials/decoded-message.tmpl.html`

**Steps:**
1. Include crypto.js when paranoid
2. On page load: check `location.hash` - if empty, show warning and disable PIN form
3. Read key from `window.location.hash`
4. After PIN validation: read encrypted blob from response, decrypt in browser
5. Detect content type: check for `!!F!!` prefix after decryption
6. Text: display plaintext in message container
7. File: parse metadata, trigger download with correct filename/content-type
8. Handle decryption errors gracefully (wrong key, corrupted data)
9. Manual test full flow

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
