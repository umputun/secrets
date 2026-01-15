# CSP-Compatible Frontend: Eliminate Inline Scripts

## Overview
- Refactor frontend to remove all inline JavaScript for strict CSP compatibility
- Enables `script-src 'self'` without `'unsafe-inline'`, maximizing security
- App runs behind reverse proxies (reproxy) with strict CSP headers
- No functionality changes - pure refactoring to external JS

## Context (from discovery)

**Files with inline `<script>` blocks:**
- `app/server/assets/html/index.tmpl.html:156-169` - popup handlers
- `app/server/assets/html/pages/home.tmpl.html:121-359` - encryption logic
- `app/server/assets/html/pages/home.tmpl.html:362-448` - file upload/tabs
- `app/server/assets/html/pages/show-message.tmpl.html:192-436` - decryption logic
- `app/server/assets/html/partials/login-popup.tmpl.html:31` - autofocus
- `app/server/assets/html/partials/email-popup.tmpl.html:67` - autofocus

**Files with inline event handlers:**
- `home.tmpl.html:11,12` - `onclick="switchTab(...)"`
- `home.tmpl.html:54,98,153` - `oninput` for PIN filtering
- `home.tmpl.html:502` - `onclick` for file input trigger
- `secure-link.tmpl.html:41-55` - `onclick` for copy button
- `show-message.tmpl.html:24` - `onsubmit="return false;"`
- `show-message.tmpl.html:262` - dynamic `onclick="copyMessage()"`

**External JS already exists:**
- `crypto.js` - encryption/decryption functions (CSP-compatible)
- `htmx.min.js`, `htmx-response-targets.js` - HTMX library

## Iterative Development Approach
- Complete each step fully before moving to the next
- Make small, focused changes
- **CRITICAL: every iteration must end with adding/updating tests**
- **CRITICAL: all tests must pass before starting next iteration**
- **CRITICAL: update this plan file when scope changes during implementation**
- Run tests after each change
- Maintain backward compatibility

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix
- Update plan if implementation deviates from original scope

## Implementation Steps

### Iteration 1: Create app.js Foundation
- [x] Create `app/server/assets/static/js/app.js` with module structure
- [x] Add popup handlers (closePopup event, backdrop click)
- [x] Add autofocus handler using MutationObserver for `data-autofocus`
- [x] Add numeric-only input handler for `data-numeric-only`
- [x] Add `index.tmpl.html` script tag for app.js
- [x] Remove inline popup script from `index.tmpl.html`
- [x] **run tests - must pass before iteration 2**

### Iteration 2: Extract Home Page Scripts (Encryption)
- [x] Move encryption logic to `app.js` (htmx:confirm, htmx:afterSwap handlers)
- [x] Move no-PIN modal handling to `app.js`
- [x] Move error reset handlers to `app.js`
- [x] Add `data-max-file-size` attribute to form for config injection
- [x] Remove first inline script block from `home.tmpl.html`
- [x] Replace `oninput` handlers with `data-numeric-only` attribute
- [x] **run tests - must pass before iteration 3**

### Iteration 3: Extract Home Page Scripts (File Upload)
- [x] Move `switchTab()` to `app.js`
- [x] Move `initDragDrop()` to `app.js`
- [x] Pre-render both text/file inputs in HTML (toggle visibility vs innerHTML)
- [x] Replace tab button `onclick` with `data-action="switch-tab" data-mode="text|file"`
- [x] Replace drop-zone `onclick` with `data-action="trigger-file-input"`
- [x] Remove second inline script block from `home.tmpl.html`
- [x] **run tests - must pass before iteration 4**

### Iteration 4: Extract Show Message Scripts (Decryption)
- [x] Move URL fragment detection and routing to `app.js`
- [x] Move client-side decryption form handler to `app.js`
- [x] Move server-side file download handler to `app.js`
- [x] Move `copyMessage()` and `escapeHtml()` to `app.js`
- [x] Replace `onsubmit="return false;"` with JS handler
- [x] Replace `oninput` handlers with `data-numeric-only`
- [x] Remove inline script from `show-message.tmpl.html`
- [x] **run tests - must pass before iteration 5**

### Iteration 5: Extract Partial Scripts
- [x] Replace `secure-link.tmpl.html` onclick with `data-action="copy-link"`
- [x] Add copy link handler to `app.js`
- [x] Replace `login-popup.tmpl.html` autofocus script with `data-autofocus`
- [x] Replace `email-popup.tmpl.html` autofocus script with `data-autofocus`
- [x] **run tests - must pass before iteration 6**

### Iteration 6: Verification & Cleanup
- [x] Build and run locally with all features enabled
- [x] Test with strict CSP header in browser dev tools
- [x] Test text secret creation (with PIN, without PIN)
- [x] Test file secret creation
- [x] Test decryption flows (client-side and server-side)
- [x] Test copy button, email popup, login popup
- [x] Verify theme toggle still works
- [x] Run E2E tests: `go test -tags=e2e -timeout=120s -failfast ./e2e/...`
- [x] **verify all tests still pass**
- [x] **final validation**

### Iteration 7: Completion
- [x] Mark all tasks above as completed
- [x] Verify plan reflects actual implementation
- [x] Run full test suite one final time
- [x] Move this plan to `docs/plans/completed/`

## Technical Details

### Data Attribute Mapping
| Inline Handler | Data Attribute | JS Selector |
|----------------|----------------|-------------|
| `onclick="switchTab('text')"` | `data-action="switch-tab" data-mode="text"` | `[data-action="switch-tab"]` |
| `onclick="switchTab('file')"` | `data-action="switch-tab" data-mode="file"` | `[data-action="switch-tab"]` |
| `onclick="...file.click()"` | `data-action="trigger-file-input"` | `[data-action="trigger-file-input"]` |
| `onclick="copyMessage()"` | `data-action="copy-message"` | `[data-action="copy-message"]` |
| `onclick="copyLink()"` | `data-action="copy-link"` | `[data-action="copy-link"]` |
| `oninput="...replace(/[^0-9]/g,'')"` | `data-numeric-only` | `[data-numeric-only]` |
| autofocus scripts | `data-autofocus` | `[data-autofocus]` |

### Config Injection
Server config passed via data attributes on form element:
```html
<form id="secret-form" data-max-file-size="{{.MaxFileSize}}" ...>
```

### app.js Structure
```javascript
'use strict';

// config from data attributes
const config = { ... };

// event handlers
function handleCopyLink(btn) { ... }
function handleSwitchTab(mode) { ... }
function handleNumericInput(input) { ... }

// initialization
document.addEventListener('DOMContentLoaded', function() {
    // setup event delegation
    document.body.addEventListener('click', handleClick);
    document.body.addEventListener('input', handleInput);

    // htmx integration
    document.body.addEventListener('htmx:confirm', handleHtmxConfirm);
    document.body.addEventListener('htmx:afterSwap', handleHtmxAfterSwap);

    // autofocus observer
    setupAutofocusObserver();

    // page-specific init
    initHomePage();
    initShowMessagePage();
});
```

## Testing Commands
```bash
# unit tests
go test -v ./...

# build and run locally
cd app && go build -o secrets && ./secrets --key=test123 --domain=localhost:8080 --protocol=http --files.enabled --auth.hash=""

# e2e tests (if available)
go test -tags=e2e -timeout=120s ./e2e/...
```
