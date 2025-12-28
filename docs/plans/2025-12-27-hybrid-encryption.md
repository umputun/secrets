# Hybrid Encryption Mode Implementation Plan

**Goal:** UI always uses client-side encryption, API uses server-side encryption, removing the `--paranoid` flag.

**Architecture:** Route-based encryption mode - API routes (`/api/v1/*`) always use server-side encryption, web routes always use client-side encryption. No header detection needed for encryption decision. Store `ClientEnc` boolean in messages to determine decryption path on retrieval. Web routes require `HX-Request` header to prevent plaintext storage.

**Tech Stack:** Go, SQLite (schema migration), existing crypto.js

---

## Tasks

### Task 1: Add ClientEnc field to store.Message
- [x] **Status: Completed**

**Files:**
- Modify: `app/store/store.go` (Message struct)
- Modify: `app/store/sqlite.go` (schema, Save, Load)
- Test: `app/store/sqlite_test.go`

**Steps:**
1. Write test: `TestSQLite_SaveLoadClientEnc` - save message with `ClientEnc=true`, load it, verify field preserved
2. Write test: `TestSQLite_MigrateExistingDB` - create DB without column, reopen, verify migration adds column
3. Run tests, verify they fail (field doesn't exist)
4. Add `ClientEnc bool` field to `store.Message` struct
5. Update SQLite base schema: add `client_enc INTEGER NOT NULL DEFAULT 0` column
6. Add migration in `sqlite.go`: on startup, detect missing column via `PRAGMA table_info(messages)`, run `ALTER TABLE` if needed
7. Update `Save()` to persist `ClientEnc`
8. Update `Load()` to read `ClientEnc`
9. Run tests, verify they pass
10. Commit

---

### Task 2: Add requireHTMX middleware for web form routes
- [x] **Status: Completed**

**Files:**
- Modify: `app/server/server.go` (or `web.go`)
- Test: `app/server/server_test.go`

**Steps:**
1. Write tests for `requireHTMX()` middleware:
   - `TestRequireHTMX_WithHeader` - HX-Request=true passes through
   - `TestRequireHTMX_WithoutHeader` - no HX-Request returns 400 "JavaScript required"
2. Run tests, verify they fail
3. Implement `requireHTMX` middleware that checks for `HX-Request: true` header
4. If missing, return HTTP 400 with message "JavaScript is required for this service"
5. Run tests, verify they pass
6. Commit

**Note:** This middleware is applied to web form submission routes (`/generate-link`) to prevent non-JS clients from storing plaintext as "client-encrypted." API routes don't need this - they always use server encryption.

**Design change:** Encryption mode is now determined by route, not header:
- API routes (`/api/v1/*`) → `ClientEnc=false` (hard-coded)
- Web routes → `ClientEnc=true` (hard-coded, protected by requireHTMX)

---

### Task 3: Update MessageProc to accept ClientEnc parameter
- [x] **Status: Completed** (refactored to use `MsgReq` struct instead of multiple params)

**Files:**
- Modify: `app/messager/messager.go`
- Test: `app/messager/messager_test.go`

**Steps:**
1. Write tests:
   - `TestMakeMessage_ClientEnc` - with ClientEnc=true, data stored as-is (no encryption)
   - `TestMakeMessage_ServerEnc` - with ClientEnc=false, data encrypted
   - `TestLoadMessage_ClientEnc` - ClientEnc=true message returns raw blob
   - `TestLoadMessage_ServerEnc` - ClientEnc=false message decrypted
2. Run tests, verify they fail
3. Add `ClientEnc bool` parameter to `MakeMessage()` signature
4. Implement conditional encryption logic
5. Update `LoadMessage()` to check `msg.ClientEnc`
6. Update existing test callers
7. Run all tests, verify they pass
8. Commit

---

### Task 4: Remove Paranoid from Params
- [x] **Status: Completed** (removed from messager.Params, kept in server.Config for now)

**Files:**
- Modify: `app/messager/messager.go`
- Modify: `app/main.go`
- Test: `app/messager/messager_test.go`

**Steps:**
1. Remove `Paranoid bool` from `messager.Params`
2. Remove `--paranoid` flag from main.go opts
3. Remove paranoid-related log message
4. Update/remove tests that reference `Paranoid` field
5. Run tests, verify they pass
6. Commit

---

### Task 5: Wire route-based encryption into handlers
- [x] **Status: Completed** (used `With(RequireHTMX)` for single-route middleware)

**Files:**
- Modify: `app/server/server.go` (routes, saveMessageCtrl, getMessageCtrl)
- Modify: `app/server/web.go` (generateLinkCtrl, loadMessageCtrl)
- Test: `app/server/server_test.go`
- Test: `app/server/web_test.go`

**Steps:**
1. Write integration tests:
   - `TestSaveMessage_API_AlwaysServerEnc` - API POST always stores ClientEnc=false (even with HX-Request spoofed)
   - `TestGenerateLink_Web_AlwaysClientEnc` - web POST stores ClientEnc=true
   - `TestGenerateLink_Web_RequiresHTMX` - web POST without HX-Request returns 400
   - `TestGetMessage_ClientEnc_ReturnsRawBlob` - ClientEnc message returns raw (not decrypted)
   - `TestGetMessage_ServerEnc_ReturnsDecrypted` - server-encrypted message decrypted
   - `TestLoadMessageCtrl_ClientEnc_ReturnsRaw` - web retrieval of ClientEnc message returns raw for JS
2. Run tests, verify they fail
3. **Update size limit logic** in `routes()`: Use 1.4x globally (simplest approach). This increases API text limit from 64KB to ~90KB - acceptable trade-off for simplicity. File uploads have handler-level validation via `MaxFileSize`.
4. Apply `requireHTMX` middleware to `generateLinkCtrl` route
5. Update `saveMessageCtrl` (API) to ALWAYS pass `ClientEnc=false` (hard-coded, ignores headers)
6. Update `generateLinkCtrl` (web) to ALWAYS pass `ClientEnc=true` (hard-coded, protected by middleware)
7. Update `loadMessageCtrl` (web) to return raw ciphertext when `msg.ClientEnc=true` for JS decryption
8. Run tests, verify they pass
9. Commit

**Key insight:** No header detection in handlers. Route determines encryption mode. This prevents API clients from spoofing `HX-Request` to bypass server encryption.

---

### Task 6: Update templates and JS for hybrid mode
- [x] **Status: Completed** (removed Paranoid conditionals, runtime #key detection for retrieval)

**Files:**
- Modify: `app/server/assets/html/index.tmpl.html`
- Modify: `app/server/assets/html/pages/home.tmpl.html`
- Modify: `app/server/assets/html/pages/show-message.tmpl.html`
- Modify: `app/server/assets/html/partials/secure-link.tmpl.html`
- Modify: `app/server/assets/html/partials/decoded-message.tmpl.html`
- Modify: `app/server/assets/static/css/main.css`
- Modify: `app/server/assets/static/js/crypto.js`

**Steps:**
1. Update crypto.js header comment: "for paranoid mode" → "client-side encryption for UI"
2. Remove `{{if .Paranoid}}` blocks from all templates
3. Always include crypto.js
4. Always generate URLs with `#key` fragment for UI-created messages
5. **Remove paranoid-specific styling:**
   - Remove `data-paranoid` attribute from `<html>` tag
   - Remove conditional theme-color meta tag (use single teal color `#14b8a6`)
   - Remove entire "Paranoid Mode Colors" CSS section (`[data-paranoid]` selectors)
   - Client-side encryption is now default, doesn't need warning color
6. **Rename JS variables** (optional but cleaner):
   - `paranoidKey` → `encryptionKey`
   - `paranoidMaxFileSize` → `maxFileSize`
   - `doParanoidEncryption` → `doClientEncryption`
   - `showParanoidError` → `showEncryptionError`
   - `resetParanoidState` → `resetEncryptionState`
   - `formatSizeParanoid` → `formatSize`
7. Implement JS runtime branching for retrieval page (check `#key` fragment)
8. Rebuild binary
9. Manual smoke test: verify teal color scheme, UI encrypt/decrypt, API messages
10. Commit

**Note:** The #key fragment determines behavior at runtime. UI messages have #key (client decrypts), API messages don't (server decrypts). No special "warning" color needed - client encryption is now the standard.

---

### Task 7: Update params endpoint
- [x] **Status: Completed** (removed Paranoid from params response)

**Files:**
- Modify: `app/server/server.go`
- Test: `app/server/server_test.go`

**Steps:**
1. Write test: `TestParams_NoParanoidField` - verify Paranoid removed from response
2. Run test, verify it fails
3. Remove `Paranoid` from params response (don't add replacement - encryption mode is now implicit: UI=client, API=server)
4. Run test, verify it passes
5. Commit

**Note:** No `ClientEncryption` field added. The behavior is now implicit and doesn't need to be advertised via API.

---

### Task 8: Update Server struct and initialization
- [x] **Status: Completed** (removed Paranoid from Config, main.go, templateData)

**Files:**
- Modify: `app/server/server.go`
- Modify: `app/main.go`
- Test: `app/server/server_test.go`

**Steps:**
1. Remove `Paranoid` from `server.Params` if present
2. Update `main.go` initialization
3. Update/remove any remaining paranoid references
4. Run tests, verify they pass
5. Commit

---

### Task 9: Clean up file upload handling
- [x] **Status: Completed** (removed generateFileLinkCtrl, multipart rejected with 400)

**Files:**
- Modify: `app/messager/messager.go`
- Modify: `app/server/web.go` (generateFileLinkCtrl)
- Test: `app/messager/messager_test.go`
- Test: `app/server/web_test.go` (update/remove multipart upload tests)

**Clarification: File upload paths:**
- **UI files**: Client JS encrypts file+metadata into blob → sends via `MakeMessage` (text endpoint) → `ClientEnc=true`
- **API files**: NOT SUPPORTED. There is no API file upload endpoint.
- **Web file form** (`generateFileLinkCtrl`): MUST BE REMOVED. This handler would bypass client-side encryption. All web file uploads must go through JS encryption → text blob path.

**Steps:**
1. Write tests:
   - `TestIsFile_ClientEnc` - returns false for ClientEnc messages (server can't inspect opaque blob)
   - `TestIsFile_ServerEnc` - returns true for server-encrypted file messages with !!FILE!! prefix
   - `TestGenerateLink_RejectsMultipart` - verify multipart POST to /generate-link returns 400
2. Run tests, verify they fail
3. Remove `ErrParanoidFile` error and check from `MakeFileMessage`
4. Update `IsFile()`: if `ClientEnc=true` return false (can't inspect), else check !!FILE!! prefix
5. **REQUIRED**: Remove multipart dispatch in `generateLinkCtrl` - return 400 for multipart requests
6. **REQUIRED**: Remove or deprecate `generateFileLinkCtrl` (dead code after step 5)
7. Keep `MakeFileMessage` for potential future API use (sets `ClientEnc=false`)
8. Run tests, verify they pass
9. Commit

**Security note:** Removing `generateFileLinkCtrl` prevents server-side file encryption bypass. All web uploads MUST use client-side encryption.

---

### Task 10: Write e2e tests for hybrid mode
- [ ] **Status: Not started**

**Files:**
- Modify: `e2e/hybrid_test.go` (new file)
- Modify: `e2e/paranoid_test.go` (remove or repurpose)

**Steps:**
1. Write e2e tests:
   - `TestE2E_UIFlow_ClientSideEncryption` - create via UI form, verify URL has #key, retrieve and decrypt client-side
   - `TestE2E_APIFlow_ServerSideEncryption` - create via JSON API, retrieve via API, verify plaintext returned
   - `TestE2E_UIFile_ClientSideEncryption` - file upload via UI, client-side encrypted
   - `TestE2E_CrossMode_UICreateAPIRetrieve` - verify API can retrieve UI-created message (returns blob)
   - `TestE2E_CrossMode_APICreateUIRetrieve` - verify UI can retrieve API-created message (server decrypts)
   - `TestE2E_WebWithoutJS_Returns400` - verify web form POST without HX-Request returns 400
2. Run e2e tests, verify they fail
3. Fix any issues discovered
4. Run e2e tests, verify they pass
5. Commit

**Note:** No `TestE2E_APIFile_ServerSideEncryption` - API file uploads are not supported (no endpoint exists).

---

### Task 11: Remove old paranoid e2e tests
- [ ] **Status: Not started**

**Files:**
- Delete or modify: `e2e/paranoid_test.go`

**Steps:**
1. Review `e2e/paranoid_test.go`
2. Remove tests that test `--paranoid` flag behavior
3. Keep/adapt tests that verify client-side encryption (now default for UI)
4. Run e2e tests, verify they pass
5. Commit

---

### Task 12: Update documentation
- [ ] **Status: Not started**

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`

**Steps:**
1. Remove references to `--paranoid` flag
2. Document new behavior: UI = client-side, API = server-side
3. Update architecture section in CLAUDE.md
4. Update configuration section in README.md
5. Add API documentation note: server-side encryption is transparent, API clients wanting extra security can pre-encrypt their payload (double-encryption) - server layer is handled automatically on both store and retrieve
6. Commit

---

### Task 13: Final validation
- [ ] **Status: Not started**

**Steps:**
1. Run `go test -race ./...`
2. Run `golangci-lint run`
3. Run all e2e tests: `go test -v ./e2e/...`
4. Manual smoke test: create secret via UI, retrieve, verify encryption
5. Manual smoke test: create secret via curl JSON, retrieve, verify
6. Fix any issues
7. Move plan to `docs/plans/completed/`
8. Final commit

---

## Notes

- TDD enforced: write failing tests first for each task
- Breaking change: `--paranoid` flag removed
- Breaking change: `/api/v1/params` no longer returns `Paranoid` field
- API message creation/retrieval behavior unchanged for existing clients
- All UI requests now zero-knowledge by default
- **JavaScript required**: UI encryption requires JavaScript (Web Crypto API). The `/generate-link` route returns HTTP 400 without `HX-Request` header. This is the only web POST that stores messages; other web POSTs (`/load-message`, `/theme`, etc.) don't create encrypted content.

## Migration Warning

**IMPORTANT for users upgrading from `--paranoid` mode:**

Existing messages created with `--paranoid` will have `ClientEnc=false` after upgrade (default value). The server will attempt to decrypt client-encrypted blobs and fail.

**Recommended upgrade path:**
1. Check your `MAX_EXPIRE` setting (default 24h, max 31 days)
2. Wait for all existing messages to expire before upgrading
3. Then deploy the new version

Alternatively, if you must upgrade immediately:
- Accept that existing paranoid-mode messages will be unreadable
- New messages will work correctly
