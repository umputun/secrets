# safesecret review profile

## What it is
- Public web service (safesecret.info) and self-hosted Docker image for one-time secret sharing
- UI: zero-knowledge, browser encrypts with AES-GCM, key only in the URL fragment
- API: server-side encryption with SIGN_KEY + PIN; PINs bcrypt-hashed
- Storage: SQLite (memory or file), messages expire and are deleted on first read

## What a real failure is
- A secret readable by anyone but the intended recipient: plaintext or key reaching the server
  on a UI path, the fragment decryption key or a PIN in logs, a read that skips the PIN check
- A one-time message readable twice, or consumed without reaching the reader
- PIN brute force beyond `PIN_ATTEMPTS`, or a rate limit that a client can bypass
- A legitimate recipient told "expired or deleted" while the message still exists
- An unauthenticated client exhausting memory, connections or disk

## Blast radius
- Users are anonymous and the operator cannot recover a lost or leaked secret
- Self-hosters run the Docker image with defaults; a default that is unsafe ships everywhere
- Example configs (`secrets-nginx.conf`, compose files) are copied into production as-is

## Reporting bar
- Report anything that can leak, double-read or lose a secret, at any likelihood
- Availability issues: only when one client or IP can trigger them
- No findings for style, naming or comment wording unless the text is wrong about behavior

## Deliberate conventions
- UI always client-encrypted, API always server-encrypted (route decides, not a flag)
- Email sharing posts the full link with the key; documented trade-off, auth expected on public
  instances
- 12-char message IDs (`msg.Key`) are logged at INFO for auditing; decryption keys and PINs never are
- No inline scripts or `hx-on::` (CSP `script-src 'self'`); JS lives in `app.js` and `crypto.js`
- Errors returned and wrapped; storage `ErrLoadRejected` means "gone", nothing else
- Tests: testify, table-driven, moq mocks; e2e under the `e2e` build tag (Playwright)
