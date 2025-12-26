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
- GCM nonce uniqueness: satisfied because each message uses fresh random key
```

**Web Crypto Requirements:**
- Web Crypto API requires HTTPS (or localhost for development)
- Runtime check required: `if (!crypto.subtle)` show error message
- Paranoid mode unavailable on plain HTTP (except localhost)

**File Handling in Paranoid Mode:**
- No server-side file detection - all content is opaque blob
- File metadata (filename, content-type) encrypted inside payload
- Client determines content type after decryption using type byte prefix
- Text format: `encrypt(0x00 || utf8_bytes(plaintext))` - type byte 0x00
- File format: `encrypt(0x01 || len_be16(filename) || utf8(filename) || len_be16(contentType) || utf8(contentType) || binaryData)`
  - Length fields: 2 bytes, big-endian
  - Strings: UTF-8 encoded
- Detection: first byte after decryption determines type (0x00=text, 0x01=file)
- Binary prefix avoids collision with user content (unlike string prefixes)

**Transport in Paranoid Mode:**
- All uploads (text and files) use same endpoint with base64 blob body
- No multipart form data - filename/content-type hidden from server
- Server stores/returns opaque blob without interpretation

**Server Response in Paranoid Mode:**
- GET /message/{key}: shows PIN entry form only (no content, same as normal mode)
- POST /load-message with valid PIN: returns encrypted blob in response
  - Web: `<div data-encrypted="...base64...">` in HTML response
  - API: `{"key": "...", "message": "...base64..."}` - same structure, encrypted content
- Encrypted blob only returned AFTER PIN validation (access control preserved)
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
- If empty in paranoid mode: show error, completely disable PIN form (not just warning)
- PIN submit button must be non-functional to prevent accidental message deletion
- Message explains: "Decryption key missing from URL - cannot decrypt this secret"
- Prevents accidental deletion when URL fragment stripped by chat apps, email clients, etc.

---

## Phase 1: SQLite Storage + Short IDs ✅ COMPLETED

Replace BoltDB with SQLite and switch to short IDs. Both MEMORY and SQLITE modes use SQLite engine - MEMORY uses `:memory:` for ephemeral storage.

### Task 1.1: Create SQLite Engine with Short IDs ✅

**Files:**
- Created: `app/store/sqlite.go`
- Created: `app/store/sqlite_test.go`
- Created: `app/store/store_test.go`
- Modified: `app/store/store.go`

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

**SQLite Configuration:**
```sql
PRAGMA journal_mode=WAL;      -- better concurrent read performance
PRAGMA synchronous=NORMAL;    -- good balance of safety and speed
```

**Implementation details:**
1. Added `GenerateID()` to store.go - 12-char base62 using crypto/rand with rejection sampling
2. Created SQLite engine with mutex protection for writes (SQLite single-writer pattern)
3. Added `NewInMemory()` wrapper that uses SQLite with `:memory:` - simplifies codebase to single engine
4. Used atomic `UPDATE...RETURNING` for IncErr
5. Added cleanup goroutine for expired messages

**Tests implemented:**
- `TestSQLite_Save`, `TestSQLite_Load`, `TestSQLite_Load_NotFound`
- `TestSQLite_IncErr`, `TestSQLite_IncErr_Concurrent`
- `TestSQLite_Remove`, `TestSQLite_Cleanup`, `TestSQLite_Cleanup_KeepsValid`
- `TestGenerateID_Length`, `TestGenerateID_Charset`, `TestGenerateID_Uniqueness`

### Task 1.2: Remove BoltDB and InMemory, Update Main ✅

**Files:**
- Modified: `app/main.go` - uses `NewInMemory()` for MEMORY, `NewSQLite()` for SQLITE
- Modified: `app/messager/messager.go` - uses `store.GenerateID()`
- Deleted: `app/store/bolt.go`, `app/store/bolt_test.go`
- Deleted: `app/store/in_memory.go`, `app/store/in_memory_test.go`
- Modified: `app/server/server_test.go` - updated test to use SQLite

**Changes:**
1. Engine choices: `MEMORY|SQLITE` (both use SQLite, MEMORY uses `:memory:`)
2. CLI flag: `--sqlite` (was `--bolt`) for persistent DB path
3. Removed `store.Key()` function, replaced with `store.GenerateID()`
4. All tests pass, linter clean

**Commit:** "replace boltdb with sqlite, use short ids"

### Task 1.3: Add Context Support ✅

**Files:**
- Modified: `app/store/sqlite.go` - all methods accept `context.Context`
- Modified: `app/store/sqlite_test.go` - tests pass context
- Modified: `app/messager/messager.go` - Engine interface and methods accept context
- Modified: `app/messager/messager_test.go` - tests pass context
- Modified: `app/server/server.go` - handlers pass `r.Context()` to messager
- Modified: `app/server/web.go` - handlers pass `r.Context()` to messager

**Changes:**
1. Engine interface methods accept `ctx context.Context` as first parameter
2. All SQLite methods use `ExecContext`/`QueryRowContext` with passed context
3. Messager methods (MakeMessage, LoadMessage, IsFile, MakeFileMessage) accept context
4. HTTP handlers pass `r.Context()` from request
5. Regenerated mocks with `go generate`

**Commit:** "add context support to storage and messager"

### Task 1.4: Linter Configuration and Fixes ✅

**Files:**
- Modified: `.golangci.yml` - added exclusions for test-specific patterns
- Modified: Multiple source files - fixed linter issues

**Linter exclusions added:**
- `Body.Close()` errors in test files
- `SA5008: duplicate struct tag` for go-flags choice syntax
- `os.Remove/os.RemoveAll` errors in test files

**Code fixes applied:**
- `perfsprint`: Use `errors.New` instead of `fmt.Errorf` for simple errors, string concatenation instead of `fmt.Sprintf`
- `whitespace`: Remove leading newlines in functions
- `wrapcheck`: Wrap errors from external packages/interfaces
- `nolintlint`: Fixed `// nolint` → `//nolint` format
- `noctx`: Use `ExecContext` with context in SQLite cleaner goroutine
- `nilnil`: Added `ErrEmailDisabled` sentinel error in email package
- `gocyclo`: Added nolint for complex but linear validation logic

### Task 1.5: Improve IP Anonymization ✅

**Files:**
- Modified: `app/server/middleware.go`
- Modified: `app/server/middleware_test.go`

**Changes:**
1. Replaced HMAC-SHA1 with HMAC-SHA256 (more secure hash)
2. Shortened anonymized IP from 12 chars to 8 chars (still 32 bits = 4 billion combinations)
3. Removed gosec nolint comment (no longer needed)

**Commit:** "improve linter config and ip anonymization"

### Task 1.6: Fix NewInMemory Isolation ✅

**Files:**
- Modified: `app/store/sqlite.go`

**Changes:**
1. Each `NewInMemory()` call now generates unique URI with random suffix
2. Uses `file:<random>?mode=memory&cache=shared` pattern
3. Prevents cross-test data leakage when tests run in parallel
4. Random bytes from `crypto/rand` ensure isolation

**Before:** `file::memory:?cache=shared` (all instances share same DB)
**After:** `file:7999cc8d98ee1b26?mode=memory&cache=shared` (isolated per instance)

**Commit:** "fix in-memory sqlite isolation"

---

## Phase 2: Paranoid Mode Flag ✅ COMPLETED

Add `--paranoid` flag and wire it through the system.

### Task 2.1: Add Flag and Expose in API ✅

**Files:**
- Modified: `app/main.go` - added `Paranoid bool` to opts struct, wired to server.Config
- Modified: `app/server/server.go` - added `Paranoid bool` to Config, exposed in params endpoint
- Modified: `app/server/server_test.go` - added test for paranoid mode in params

**Changes:**
1. Added `--paranoid` flag (env: `PARANOID`) to main.go opts
2. Added `Paranoid bool` to server.Config
3. Added `paranoid` field to `/api/v1/params` response
4. Added log message when paranoid mode is enabled
5. Updated `TestServer_getParams` to expect `paranoid:false` in response
6. Added `TestServer_getParams_Paranoid` to verify `paranoid:true` when enabled

**Commit:** "add --paranoid flag"

---

## Phase 3: Server-Side Paranoid Logic

Skip server encryption/decryption in paranoid mode. All content treated as opaque blob.

### Task 3.1: Messager Paranoid Mode

**Files:**
- Modify: `app/messager/messager.go`
- Modify: `app/messager/messager_test.go`
- Modify: `app/main.go`
- Modify: `app/server/server.go` (API handlers)
- Modify: `app/server/web.go` (web handlers)

**Steps:**
1. Add `Paranoid bool` to messager.Params, pass from main.go
2. In `MakeMessage`: if paranoid, store data as-is (skip encrypt)
3. In `LoadMessage`: if paranoid, return data as-is (skip decrypt)
4. Keep PIN validation (access control still works)
5. In paranoid mode, `MakeFileMessage` not used - all content goes through `MakeMessage` as opaque blob
6. `IsFile()` returns false in paranoid mode (server doesn't know content type)
7. In `getMessageCtrl`: return opaque blob in `message` field (same structure, encrypted content)
8. In paranoid mode, `generateFileLinkCtrl` (multipart) is NOT used - files go through `generateLinkCtrl` as base64 blob
9. Disable/hide multipart file upload route in paranoid mode (client handles file→blob conversion)
10. Write tests (see test list below)
11. Run tests: `go test -v ./app/messager/...`
12. Run linter

**Tests for messager_test.go:**
- `TestMakeMessage_Paranoid` - data stored as-is without encryption
- `TestLoadMessage_Paranoid` - data returned as-is without decryption
- `TestLoadMessage_Paranoid_WrongPin` - PIN validation still works
- `TestLoadMessage_Paranoid_Expired` - expiration still works
- `TestIsFile_Paranoid` - always returns false regardless of content

### Task 3.2: Adjust Size Limits for Paranoid Mode

**Files:**
- Modify: `app/server/server.go`

**Steps:**
1. In routes(), if paranoid mode: set sizeLimit to MaxFileSize * 1.4 (covers base64 overhead)
2. This applies to all requests since server can't distinguish text from files
3. Write test (see test list below)
4. Run tests

**Tests for server_test.go:**
- `TestSizeLimit_Paranoid` - verify request size limit is MaxFileSize * 1.4 when paranoid=true
- `TestSizeLimit_Normal` - verify original limits (64KB or MaxFileSize+10KB) when paranoid=false

**Commit:** "skip server-side crypto in paranoid mode"

---

## Phase 4: Client-Side Crypto

Create JavaScript crypto module.

### Task 4.1: Create Crypto Module

**Files:**
- Create: `app/server/assets/static/js/crypto.js`

**Functions:**
- `checkCryptoAvailable()` → boolean (checks `crypto.subtle` exists)
- `generateKey()` → 22-char base64url (128-bit)
- `encrypt(plaintext, key)` → base64url ciphertext (prepends 0x00 type byte)
- `decrypt(ciphertext, key)` → plaintext (checks 0x00 type byte)
- `encryptFile(data, filename, contentType, key)` → base64url (prepends 0x01 + metadata)
- `decryptFile(ciphertext, key)` → {filename, contentType, data} (parses 0x01 format)

**Steps:**
1. Add `checkCryptoAvailable()` that returns false if `!crypto.subtle` (HTTP without localhost)
2. Implement all crypto functions using Web Crypto API
3. Use binary type bytes: 0x00 for text, 0x01 for files (no string prefix collision)
4. Verify implementation compiles and loads in browser (no syntax errors)

Note: crypto.js is tested via Playwright in Phase 7 (`page.evaluate()` for unit tests, full E2E for integration).

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
3. On page load: check `checkCryptoAvailable()`, show error if false (HTTPS required)
4. Before encryption: validate content size against MaxFileSize limit, show error if exceeded
5. For text: encrypt with 0x00 prefix, POST as base64 blob
6. For files: encrypt with 0x01 prefix + metadata, POST as base64 blob (not multipart)
7. Both use same form field - server sees opaque blob either way
8. After success: append `#key` to displayed URL
9. Update copy button to include fragment

Note: Create flow tested via Playwright E2E in Phase 7.

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
2. In paranoid mode: show generic "Decrypt" button (not "Reveal"/"Download" since server doesn't know type)
3. On page load: check `location.hash` - if empty, show error and completely disable PIN form
4. Check `checkCryptoAvailable()`, show error if false
5. Read key from `window.location.hash`
6. After PIN validation: read encrypted blob from response, decrypt in browser
7. Detect content type: check first byte (0x00=text, 0x01=file)
8. Text (0x00): display plaintext in message container
9. File (0x01): parse metadata, use `Blob` + `URL.createObjectURL()` for download
   ```javascript
   const blob = new Blob([data], { type: contentType });
   const url = URL.createObjectURL(blob);
   const a = document.createElement('a');
   a.href = url; a.download = filename; a.click();  // download attr forces download, safe
   ```
   - Use `textContent` (not `innerHTML`) when displaying filename to prevent XSS
   - `a.download` forces download behavior regardless of content-type (no render risk)
10. Handle decryption errors gracefully (wrong key, corrupted data)

Note: Reveal flow tested via Playwright E2E in Phase 7.

**Commit:** "add client-side decryption to reveal page"

---

## Phase 7: E2E Tests

Add paranoid mode e2e tests covering crypto module, create flow, and reveal flow.

### Task 7.1: Paranoid Mode Tests

**Files:**
- Modify: `e2e/tests/secrets.spec.ts`

**Tests (secrets.spec.ts):**
- `test('paranoid: create text secret has fragment')` - URL contains `#` with key
- `test('paranoid: retrieve text secret decrypts')` - full round-trip works
- `test('paranoid: wrong PIN rejected')` - PIN validation works
- `test('paranoid: file upload and download')` - binary round-trip, correct filename
- `test('paranoid: one-time read enforced')` - second access fails
- `test('paranoid: missing fragment disables form')` - navigate without hash, form disabled
- `test('paranoid: wrong key shows error')` - tampered hash, graceful error
- `test('paranoid: crypto module round-trip')` - unit test via page.evaluate
- `test('normal mode still works')` - regression test without paranoid flag

**Steps:**
1. Add paranoid mode test cases
2. Add crypto.js unit tests via Playwright evaluate (see examples below)
3. Run: `make e2e`
4. All tests pass

**Crypto Module Unit Tests (via page.evaluate):**
```typescript
test('crypto: key generation', async ({ page }) => {
  await page.goto('/');
  const key = await page.evaluate(() => generateKey());
  expect(key).toHaveLength(22);
  expect(key).toMatch(/^[A-Za-z0-9_-]+$/);
});

test('crypto: text round-trip', async ({ page }) => {
  await page.goto('/');
  const result = await page.evaluate(async () => {
    const key = await generateKey();
    const cipher = await encrypt('hello world', key);
    return await decrypt(cipher, key);
  });
  expect(result).toBe('hello world');
});

test('crypto: file round-trip', async ({ page }) => {
  await page.goto('/');
  const result = await page.evaluate(async () => {
    const key = await generateKey();
    const data = new Uint8Array([1, 2, 3, 4, 5]);
    const cipher = await encryptFile(data, 'test.bin', 'application/octet-stream', key);
    return await decryptFile(cipher, key);
  });
  expect(result.filename).toBe('test.bin');
  expect(result.contentType).toBe('application/octet-stream');
  expect(result.data).toEqual(new Uint8Array([1, 2, 3, 4, 5]));
});

test('crypto: wrong key rejected', async ({ page }) => {
  await page.goto('/');
  const error = await page.evaluate(async () => {
    const key1 = await generateKey();
    const key2 = await generateKey();
    const cipher = await encrypt('secret', key1);
    try { await decrypt(cipher, key2); return null; }
    catch (e) { return e.message; }
  });
  expect(error).not.toBeNull();
});
```

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

- [x] SQLite storage works with WAL mode
- [x] Short IDs (12 chars) with rejection sampling (unbiased)
- [x] Atomic IncErr with UPDATE...RETURNING
- [x] Mutex protection for SQLite writes (single-writer pattern)
- [x] MEMORY mode uses SQLite with `:memory:` (unified engine)
- [x] Each NewInMemory() creates isolated database (unique URI)
- [x] Context propagation through all layers
- [x] Linter configuration with proper exclusions
- [x] IP anonymization uses HMAC-SHA256 (8 chars)
- [x] `--paranoid` flag works
- [ ] Server skips crypto in paranoid mode
- [ ] PIN still enforces access control
- [ ] Client-side crypto works with binary type bytes (0x00/0x01)
- [ ] Web Crypto availability check (HTTPS required)
- [ ] URLs include `#key` fragment in paranoid mode
- [ ] Missing fragment completely disables PIN form
- [ ] File upload/download works in paranoid mode (Blob + createObjectURL)
- [ ] Generic "Decrypt" button in paranoid mode (not Reveal/Download)
- [ ] Size validation before encryption
- [ ] E2E tests pass including crypto round-trip
- [ ] Normal mode regression tests pass
- [ ] Docs updated
- [x] Linter clean
