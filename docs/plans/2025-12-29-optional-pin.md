# Optional PIN for Client-Side Encrypted Messages

## Overview

Allow users to skip PIN protection when creating secrets via the web UI, for use cases where the sharing channel itself is already secure (e.g., Signal, iMessage).

**Problem:** When sharing secrets via E2E encrypted messengers, requiring a PIN adds friction without meaningful security benefit - the channel already provides the protection.

**Solution:** Add `--allow-no-pin` CLI flag (default false). When enabled and user leaves PIN empty, show confirmation modal. Store empty PIN hash. On retrieval, show "Reveal Secret" button instead of PIN form.

**Key Benefits:**
- Reduced friction for secure channel use cases
- Operator control via CLI flag (opt-in)
- Clear user confirmation to prevent accidental PIN-less secrets

## Context (from discovery)

**Files/components involved:**
- `app/main.go` - CLI flag definition
- `app/server/server.go` - config struct, params endpoint
- `app/server/web.go` - form validation, template data
- `app/messager/messager.go` - message creation/retrieval logic
- `app/server/assets/html/` - templates for modal and reveal button
- `app/server/assets/static/js/crypto.js` - client-side encryption flow

**Related patterns found:**
- PIN validation in `web.go` lines 167-170, 305-308
- Message struct uses `Pin` field for bcrypt hash
- Client-side encryption in `crypto.js` handles PIN in `encryptAndSubmit()`

**Dependencies:**
- No external dependencies required
- Uses existing modal pattern from email popup

## Iterative Development Approach

- Complete each iteration fully before moving to the next
- Make small, focused changes
- **CRITICAL: every iteration must end with adding/updating tests**
- **CRITICAL: all tests must pass before starting next iteration**
- Run tests after each change
- Maintain backward compatibility

## Progress Tracking

- Mark completed items with `[x]`
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix

## Implementation Steps

### Iteration 1: Backend Foundation (Config + Messager)

- [ ] Add `AllowNoPin bool` to opts struct in `app/main.go` with flag `--allow-no-pin` and env `ALLOW_NO_PIN`
- [ ] Add `AllowNoPin bool` to `server.Config` struct in `app/server/server.go`
- [ ] Pass value from opts to server config in `app/main.go`
- [ ] Update `messager.MsgReq` to support empty PIN (add `AllowEmptyPin bool` field or check PIN length)
- [ ] Update `MakeMessage` in `app/messager/messager.go` to skip PIN hashing when PIN is empty
- [ ] **Add tests in `app/messager/messager_test.go` for empty PIN creation**
- [ ] **Run `go test ./...` - must pass before iteration 2**

### Iteration 2: Backend Retrieval

- [ ] Add `HasPin(key string) bool` method to messager (checks if stored PIN hash is empty)
- [ ] Update `LoadMessage` to accept empty PIN when message has empty PIN hash
- [ ] Skip bcrypt compare when both stored hash and provided PIN are empty
- [ ] **Add tests in `app/messager/messager_test.go` for PIN-less retrieval**
- [ ] **Run `go test ./...` - must pass before iteration 3**

### Iteration 3: Web Layer (Form Validation + Template Data)

- [ ] Add `AllowNoPin` to `templateData` struct in `app/server/web.go`
- [ ] Pass `s.cfg.AllowNoPin` in `newTemplateData()`
- [ ] Modify PIN validation in `generateLinkCtrl` to skip when `AllowNoPin=true` and PIN is empty
- [ ] Add `HasPin` field to form data for retrieval view
- [ ] Update `showMessageViewCtrl` to check if message has PIN
- [ ] **Add tests in `app/server/web_test.go` for empty PIN submission**
- [ ] **Run `go test ./...` - must pass before iteration 4**

### Iteration 4: Frontend - Creation Modal

- [ ] Create `app/server/assets/html/partials/no-pin-modal.tmpl.html` with confirmation message
- [ ] Add modal trigger in `crypto.js` `encryptAndSubmit()` when PIN is empty and AllowNoPin is true
- [ ] Wire confirm button to proceed with encryption (empty PIN)
- [ ] Wire cancel button to close modal and focus PIN field
- [ ] **Manual test: verify modal appears, confirm works, cancel works**
- [ ] **Run `go test ./...` - must pass before iteration 5**

### Iteration 5: Frontend - Retrieval UI

- [ ] Create `app/server/assets/html/partials/reveal-button.tmpl.html` with "Reveal Secret" button
- [ ] Modify `message-form.tmpl.html` to conditionally show PIN form OR reveal button based on HasPin
- [ ] Wire reveal button to trigger decryption flow (same as PIN form, but no PIN)
- [ ] **Manual test: verify PIN-less message shows reveal button, click reveals content**
- [ ] **Run `go test ./...` - must pass before iteration 6**

### Iteration 6: API + E2E Tests

- [ ] Add `allow_no_pin` to params response in `getParamsCtrl`
- [ ] **Add test in `app/server/server_test.go` for params endpoint**
- [ ] Create `e2e/no_pin_test.go` with tests:
  - Create secret without PIN (when enabled)
  - Retrieve PIN-less secret via reveal button
  - Verify PIN-less creation fails when disabled
- [ ] **Run e2e tests - must pass before iteration 7**

### Iteration 7: Documentation & Cleanup

- [ ] Add `--allow-no-pin` to README.md configuration table
- [ ] Document use case (secure channels like Signal/iMessage)
- [ ] Update Security Architecture section if needed
- [ ] Code cleanup and remove any debug code
- [ ] **Run `go test ./...` and e2e tests - final validation**

## Technical Details

**Data flow - Creation:**
```
User leaves PIN empty → crypto.js detects empty PIN →
if AllowNoPin: show modal → on confirm: encrypt with empty PIN →
server stores with empty PIN hash
```

**Data flow - Retrieval:**
```
User opens link → server checks HasPin →
if no PIN: return hasPin=false → UI shows reveal button →
user clicks → decrypt with URL fragment → show content → delete
```

**Storage:**
- Empty string for PIN hash field when no PIN
- No schema changes required
- bcrypt compare skipped when both values empty

**Modal message:**
> "Create without PIN? Anyone with this link can access your secret once. After viewing, it will be permanently deleted."
