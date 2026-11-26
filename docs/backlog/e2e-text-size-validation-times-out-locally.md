---
worth: maybe
where: e2e/hybrid_test.go:TestHybrid_TextSizeValidation
added: 2026-11-25
---
# e2e text size validation times out on a local Mac

`TestHybrid_TextSizeValidation` fails locally on a clean master (6c68d58): `page.Locator("#message").Fill(...)`
with a 1,048,577-char string hits Playwright's 30s timeout. It fails the same way when run alone, at load ~4 on
a 20-core Mac, while the e2e workflow passes on the same commit in CI (ubuntu). Every other e2e test passes
locally.

Not investigated: the likely difference is the local Playwright browser build or driver vs CI's. Filling the
textarea through JS (setting `value` plus an `input` event) instead of typing would probably avoid it, but
that changes what the test exercises, so decide the cause first.
