// app.js - CSP-compatible application logic
// all event handlers use data attributes instead of inline handlers
'use strict';

// ============================================================================
// configuration (read from data attributes)
// ============================================================================

function getConfig() {
    const form = document.getElementById('secret-form');
    return {
        maxFileSize: parseInt(form?.dataset.maxFileSize || '0', 10)
    };
}

// ============================================================================
// autofocus handler (replaces inline autofocus scripts in popups)
// ============================================================================

function setupAutofocusObserver() {
    const observer = new MutationObserver(function(mutations) {
        for (const mutation of mutations) {
            for (const node of mutation.addedNodes) {
                if (node.nodeType !== 1) continue;
                // check if node itself or its children have data-autofocus
                const el = node.matches?.('[data-autofocus]') ? node :
                          node.querySelector?.('[data-autofocus]');
                if (el) {
                    setTimeout(function() { el.focus(); }, 50);
                }
            }
        }
    });
    observer.observe(document.body, { childList: true, subtree: true });
}

// ============================================================================
// popup handlers (from index.tmpl.html)
// ============================================================================

function setupPopupHandlers() {
    // handle closePopup custom event
    document.body.addEventListener('closePopup', function() {
        htmx.ajax('GET', '/close-popup', {target: '#popup', swap: 'outerHTML'});
    });

    // handle backdrop clicks on popup (except email popup)
    document.body.addEventListener('click', function(evt) {
        const popup = evt.target.closest('#popup');
        if (popup && evt.target === popup && !popup.querySelector('.email-popup')) {
            htmx.ajax('GET', '/close-popup', {target: '#popup', swap: 'outerHTML'});
        }
    });
}

// ============================================================================
// numeric input handler (replaces oninput for PIN fields)
// ============================================================================

function setupNumericInputHandler() {
    document.body.addEventListener('input', function(evt) {
        const input = evt.target;
        if (!input.hasAttribute('data-numeric-only')) return;
        input.value = input.value.replace(/[^0-9]/g, '');

        // also clear associated error display if specified
        const errorId = input.dataset.errorTarget;
        if (errorId) {
            const errorEl = document.getElementById(errorId);
            if (errorEl) errorEl.innerHTML = '';
            input.classList.remove('error-input');
        }
    });
}

// ============================================================================
// expire field error clearing (replaces inline hx-on:input/change handlers)
// ============================================================================

function setupExpireErrorHandler() {
    function clearExpireErrors(el) {
        if (!el.hasAttribute('data-clear-expire-errors')) return;

        const container = el.closest('.expire-container');
        if (!container) return;

        // remove error spans that follow the container
        let sibling = container.nextElementSibling;
        while (sibling && sibling.classList.contains('error')) {
            const next = sibling.nextElementSibling;
            sibling.remove();
            sibling = next;
        }

        // clear error-input class from both fields
        container.querySelectorAll('.error-input').forEach(function(field) {
            field.classList.remove('error-input');
        });
    }

    document.body.addEventListener('input', function(evt) {
        clearExpireErrors(evt.target);
    });

    document.body.addEventListener('change', function(evt) {
        clearExpireErrors(evt.target);
    });
}

// ============================================================================
// copy handlers (from secure-link.tmpl.html and show-message.tmpl.html)
// ============================================================================

function setupCopyHandlers() {
    // copy link handler (for secure-link textarea)
    document.body.addEventListener('click', function(evt) {
        const btn = evt.target.closest('[data-action="copy-link"]');
        if (!btn) return;

        const textareaId = btn.dataset.source || 'msg-text';
        const textarea = document.getElementById(textareaId);
        if (!textarea) return;

        navigator.clipboard.writeText(textarea.value).then(function() {
            const originalHtml = btn.innerHTML;
            btn.innerHTML = "<svg class='btn-icon' width='16' height='16' viewBox='0 0 24 24' fill='none'><polyline points='20,6 9,17 4,12' stroke='currentColor' stroke-width='2'/></svg>Copied!";
            btn.style.background = 'var(--color-success)';
            setTimeout(function() {
                btn.innerHTML = originalHtml;
                btn.style.background = '';
            }, 2000);
        }).catch(function() {
            btn.innerHTML = "<svg class='btn-icon' width='16' height='16' viewBox='0 0 24 24' fill='none'><line x1='18' y1='6' x2='6' y2='18' stroke='currentColor' stroke-width='2'/><line x1='6' y1='6' x2='18' y2='18' stroke='currentColor' stroke-width='2'/></svg>Error";
            btn.style.background = 'var(--color-error)';
        });
    });

    // copy message handler (for decoded message textarea on show-message page)
    document.body.addEventListener('click', function(evt) {
        const btn = evt.target.closest('[data-action="copy-message"]');
        if (!btn) return;

        const textarea = document.getElementById('decoded-msg-text');
        if (!textarea) return;

        navigator.clipboard.writeText(textarea.value).then(function() {
            btn.textContent = 'Copied!';
            btn.style.background = 'var(--color-success)';
            setTimeout(function() {
                btn.textContent = 'Copy';
                btn.style.background = '';
            }, 2000);
        }).catch(function() {
            btn.textContent = 'Error';
            btn.style.background = 'var(--color-error)';
        });
    });

    // copy decoded message handler (for server-decrypted messages with visual feedback)
    document.body.addEventListener('click', function(evt) {
        const btn = evt.target.closest('[data-action="copy-decoded-message"]');
        if (!btn) return;

        const textareaId = btn.dataset.source || 'decoded-msg-text';
        const textarea = document.getElementById(textareaId);
        if (!textarea) return;

        const originalHtml = btn.innerHTML;
        navigator.clipboard.writeText(textarea.value).then(function() {
            btn.innerHTML = "<svg class='btn-icon' width='16' height='16' viewBox='0 0 24 24' fill='none' xmlns='http://www.w3.org/2000/svg'><polyline points='20,6 9,17 4,12' stroke='currentColor' stroke-width='2'/></svg><span class='btn-text'>Copied!</span>";
            btn.style.background = 'var(--color-success)';
            setTimeout(function() {
                btn.innerHTML = originalHtml;
                btn.style.background = '';
            }, 2000);
        }).catch(function() {
            btn.innerHTML = "<svg class='btn-icon' width='16' height='16' viewBox='0 0 24 24' fill='none' xmlns='http://www.w3.org/2000/svg'><line x1='18' y1='6' x2='6' y2='18' stroke='currentColor' stroke-width='2'/><line x1='6' y1='6' x2='18' y2='18' stroke='currentColor' stroke-width='2'/></svg><span class='btn-text'>Error!</span>";
            btn.style.background = 'var(--color-error)';
            setTimeout(function() {
                btn.innerHTML = originalHtml;
                btn.style.background = '';
            }, 2000);
        });
    });

    // copy text from data attribute (used by copy-button partial, clipboard before HTMX request)
    document.body.addEventListener('click', function(evt) {
        const btn = evt.target.closest('[data-action="copy-text"]');
        if (!btn) return;

        const text = btn.dataset.text;
        if (text) {
            navigator.clipboard.writeText(text).catch(function() {});
        }
    });
}

// ============================================================================
// file upload handlers (from home.tmpl.html)
// ============================================================================

function setupFileUploadHandlers() {
    // tab switching
    document.body.addEventListener('click', function(evt) {
        const btn = evt.target.closest('[data-action="switch-tab"]');
        if (!btn) return;

        const mode = btn.dataset.mode;
        switchTab(mode);
    });

    // trigger file input click
    document.body.addEventListener('click', function(evt) {
        const el = evt.target.closest('[data-action="trigger-file-input"]');
        if (!el) return;

        const fileInput = document.getElementById('file');
        if (fileInput) fileInput.click();
    });
}

function switchTab(mode) {
    const form = document.getElementById('secret-form');
    const textTab = document.getElementById('text-tab');
    const fileTab = document.getElementById('file-tab');
    const textInput = document.getElementById('text-input-container');
    const fileInput = document.getElementById('file-input-container');

    if (!form || !textTab || !fileTab) return;

    if (mode === 'file') {
        textTab.classList.remove('active');
        fileTab.classList.add('active');
        form.setAttribute('enctype', 'multipart/form-data');
        if (textInput) textInput.style.display = 'none';
        if (fileInput) {
            fileInput.style.display = 'block';
            initDragDrop();
        }
    } else {
        fileTab.classList.remove('active');
        textTab.classList.add('active');
        form.removeAttribute('enctype');
        if (fileInput) fileInput.style.display = 'none';
        // clear actual file input to prevent wrong payload when switching to text mode
        const actualFileInput = document.getElementById('file');
        if (actualFileInput) actualFileInput.value = '';
        // reset file-info display and error state
        const fileInfo = document.getElementById('file-info');
        if (fileInfo) {
            fileInfo.style.display = 'none';
            fileInfo.textContent = '';
            fileInfo.classList.remove('error');
        }
        // reset drop zone error state
        const dropZone = document.getElementById('drop-zone');
        if (dropZone) dropZone.classList.remove('error-input');
        // re-enable submit button (may have been disabled by file size error)
        const submitBtn = form.querySelector('button[type="submit"]');
        if (submitBtn) submitBtn.disabled = false;
        if (textInput) textInput.style.display = 'block';
    }
}

function initDragDrop() {
    const dropZone = document.getElementById('drop-zone');
    const fileInput = document.getElementById('file');
    const fileInfo = document.getElementById('file-info');
    const submitBtn = document.querySelector('button[type="submit"]');
    const config = getConfig();

    if (!dropZone || dropZone.dataset.initialized) return;
    dropZone.dataset.initialized = 'true';

    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(function(e) {
        dropZone.addEventListener(e, function(ev) {
            ev.preventDefault();
            ev.stopPropagation();
        });
    });

    ['dragenter', 'dragover'].forEach(function(e) {
        dropZone.addEventListener(e, function() {
            dropZone.classList.add('drag-over');
        });
    });

    ['dragleave', 'drop'].forEach(function(e) {
        dropZone.addEventListener(e, function() {
            dropZone.classList.remove('drag-over');
        });
    });

    dropZone.addEventListener('drop', function(e) {
        const files = e.dataTransfer.files;
        if (files.length && fileInput) {
            fileInput.files = files;
            updateFileInfo(files[0]);
        }
    });

    if (fileInput) {
        fileInput.addEventListener('change', function(e) {
            if (e.target.files.length) {
                updateFileInfo(e.target.files[0]);
            }
        });
    }

    function updateFileInfo(file) {
        if (!fileInfo) return;

        const sizeStr = formatSize(file.size);
        const placeholder = dropZone.querySelector('svg');
        const placeholderText = dropZone.querySelector('p:not(.file-info)');
        const sizeLimit = config.maxFileSize;

        if (file.size > sizeLimit) {
            fileInfo.textContent = file.name + ' (' + sizeStr + ') - too large! Max: ' + formatSize(sizeLimit);
            fileInfo.style.display = 'block';
            fileInfo.classList.add('error');
            dropZone.classList.add('error-input');
            if (submitBtn) submitBtn.disabled = true;
            if (fileInput) fileInput.value = '';
            if (placeholder) placeholder.style.display = 'none';
            if (placeholderText) placeholderText.style.display = 'none';
        } else {
            fileInfo.textContent = file.name + ' (' + sizeStr + ')';
            fileInfo.style.display = 'block';
            fileInfo.classList.remove('error');
            dropZone.classList.remove('error-input');
            if (submitBtn) submitBtn.disabled = false;
            if (placeholder) placeholder.style.display = 'none';
            if (placeholderText) placeholderText.style.display = 'none';
        }
    }
}

// ============================================================================
// encryption handlers (from home.tmpl.html)
// ============================================================================

let encryptionKey = null;
let encryptionDone = false;
let pendingForm = null;
let pendingEvent = null;
let noPinConfirmed = false;

function setupEncryptionHandlers() {
    const formCard = document.getElementById('form-card');
    if (!formCard) return; // not on home page

    // check Web Crypto availability
    if (!checkCryptoAvailable()) {
        formCard.innerHTML = '<div class="card-header"><h2 class="card-title">Encryption Unavailable</h2>' +
            '<p class="card-description error">Client-side encryption requires HTTPS. Web Crypto API is not available on plain HTTP connections.</p></div>';
        return;
    }

    // wire up no-pin modal buttons
    const confirmBtn = document.getElementById('no-pin-confirm');
    const cancelBtn = document.getElementById('no-pin-cancel');

    if (confirmBtn) {
        confirmBtn.addEventListener('click', function() {
            document.getElementById('no-pin-modal').classList.remove('active');
            noPinConfirmed = true;
            if (pendingForm) doEncryptionWork(pendingForm);
        });
    }

    if (cancelBtn) {
        cancelBtn.addEventListener('click', function() {
            document.getElementById('no-pin-modal').classList.remove('active');
            pendingForm = null;
            pendingEvent = null;
            document.getElementById('pin')?.focus();
        });
    }

    // intercept htmx before it sends the request to do encryption first
    document.body.addEventListener('htmx:confirm', function(evt) {
        if (evt.detail.elt.id !== 'secret-form') return;
        if (encryptionDone) return;

        evt.preventDefault();
        pendingEvent = evt;
        doClientEncryption(evt.detail.elt);
    });

    // after successful swap, append key to URL and update email button
    document.body.addEventListener('htmx:afterSwap', handleAfterSwap);

    // handle non-swap failures
    document.body.addEventListener('htmx:sendError', resetEncryptionState);
    document.body.addEventListener('htmx:timeout', resetEncryptionState);
    document.body.addEventListener('htmx:sendAbort', resetEncryptionState);
    document.body.addEventListener('htmx:swapError', resetEncryptionState);
    document.body.addEventListener('htmx:targetError', resetEncryptionState);
    document.body.addEventListener('htmx:responseError', function(evt) {
        const status = evt.detail.xhr?.status;
        if (status && status !== 400 && status !== 401 && status !== 500) {
            resetEncryptionState();
        }
    });

    // reset state if auth popup is closed manually
    document.body.addEventListener('click', function(evt) {
        if (!encryptionDone) return;
        if (evt.target.closest('.close-popup')) {
            resetEncryptionState();
            return;
        }
        const popup = document.getElementById('popup');
        if (popup && popup.classList.contains('active') && evt.target === popup) {
            resetEncryptionState();
        }
    });
}

async function doClientEncryption(form) {
    const errDiv = document.getElementById('form-errors');
    if (errDiv) errDiv.innerHTML = '';

    const pinInput = document.getElementById('pin');
    const pinValue = pinInput ? pinInput.value.trim() : '';
    const noPinModal = document.getElementById('no-pin-modal');

    if (pinValue === '' && noPinModal && !noPinConfirmed) {
        pendingForm = form;
        noPinModal.classList.add('active');
        return;
    }

    await doEncryptionWork(form);
}

async function doEncryptionWork(form) {
    const errDiv = document.getElementById('form-errors');
    if (errDiv) errDiv.innerHTML = '';

    const config = getConfig();

    try {
        encryptionKey = await generateKey();

        const fileInput = document.getElementById('file');
        const isFileUpload = fileInput && fileInput.files.length > 0;

        let encryptedBlob;
        let messageEl = document.getElementById('message');

        if (isFileUpload) {
            const file = fileInput.files[0];
            if (file.size > config.maxFileSize) {
                showEncryptionError('File too large. Maximum size: ' + formatSize(config.maxFileSize));
                return;
            }
            const arrayBuffer = await file.arrayBuffer();
            encryptedBlob = await encryptFile(arrayBuffer, file.name, file.type || 'application/octet-stream', encryptionKey);

            if (!messageEl) {
                messageEl = document.createElement('input');
                messageEl.type = 'hidden';
                messageEl.name = 'message';
                messageEl.id = 'message';
                form.appendChild(messageEl);
            }
            fileInput.remove();
            form.removeAttribute('enctype');
        } else {
            const message = messageEl ? messageEl.value : '';
            if (!message) {
                showEncryptionError('Message cannot be empty');
                return;
            }
            const msgBytes = new TextEncoder().encode(message).length;
            if (msgBytes > config.maxFileSize) {
                showEncryptionError('Message too large. Maximum size: ' + formatSize(config.maxFileSize));
                return;
            }
            encryptedBlob = await encrypt(message, encryptionKey);
        }

        messageEl.value = encryptedBlob;
        encryptionDone = true;

        if (pendingEvent) {
            pendingEvent.detail.issueRequest();
            pendingEvent = null;
            pendingForm = null;
        }
    } catch (e) {
        showEncryptionError('Encryption failed: ' + e.message);
    }
}

function handleAfterSwap(evt) {
    if (!encryptionKey) return;

    const textarea = document.getElementById('msg-text');
    if (textarea && textarea.value.includes('/message/')) {
        const fullUrl = textarea.value + '#' + encryptionKey;
        textarea.value = fullUrl;

        const emailBtn = document.querySelector('button[hx-get^="/email-popup"]');
        if (emailBtn) {
            emailBtn.setAttribute('hx-get', '/email-popup?link=' + encodeURIComponent(fullUrl));
            htmx.process(emailBtn);
        }

        encryptionKey = null;
        encryptionDone = false;
    } else if (evt.detail.target && evt.detail.target.id === 'form-card') {
        clearFormFields();
    } else if (evt.detail.target && evt.detail.target.id === 'notifications') {
        clearFormFields();
    }
}

function resetEncryptionState() {
    if (!encryptionDone) return;
    clearFormFields();
}

function clearFormFields() {
    const msgField = document.getElementById('message');
    if (msgField) msgField.value = '';
    const fileField = document.getElementById('file');
    if (fileField) fileField.value = '';
    encryptionKey = null;
    encryptionDone = false;
    noPinConfirmed = false;
}

function showEncryptionError(msg) {
    const errDiv = document.getElementById('form-errors');
    if (errDiv) {
        errDiv.innerHTML = '<span class="error">' + msg + '</span>';
    }
}

// ============================================================================
// decryption handlers (from show-message.tmpl.html)
// ============================================================================

function setupDecryptionHandlers() {
    const showMsg = document.getElementById('show-msg');
    if (!showMsg) return; // not on show-message page

    const cryptoKey = window.location.hash.slice(1);
    const hasKey = cryptoKey && cryptoKey.length > 0;

    const keyError = document.getElementById('key-error');
    const clientForm = document.getElementById('client-decrypt-form');
    const serverForms = document.getElementById('server-forms');

    if (hasKey) {
        // client-encrypted message
        if (serverForms) serverForms.style.display = 'none';

        if (typeof checkCryptoAvailable !== 'function' || !checkCryptoAvailable()) {
            document.getElementById('message-container').innerHTML =
                '<div class="card error-card"><div class="card-header">' +
                '<h2 class="card-title">Encryption Unavailable</h2></div>' +
                '<p class="error-message">Web Crypto API is not available. HTTPS is required for encrypted messages.</p>' +
                '<a href="/" class="main-btn">Back to Main Page</a></div>';
            return;
        }

        if (clientForm) {
            clientForm.style.display = 'block';
            const clientPin = document.getElementById('client-pin');
            if (clientPin) clientPin.focus();
        }

        const decryptForm = document.getElementById('decrypt-form');
        if (decryptForm) {
            decryptForm.addEventListener('submit', function(e) {
                e.preventDefault();
                handleClientDecryption(cryptoKey);
            });
        }
    } else {
        // server-side decryption
        if (clientForm) clientForm.style.display = 'none';

        const downloadForm = document.getElementById('download-form');
        if (downloadForm) {
            downloadForm.addEventListener('submit', function(e) {
                e.preventDefault();
                handleFileDownload();
            });
        }
    }
}

async function handleClientDecryption(cryptoKey) {
    const form = document.getElementById('decrypt-form');
    const btn = document.getElementById('decrypt-btn');
    const btnText = document.getElementById('btn-text');
    const btnLoading = document.getElementById('btn-loading');
    const pinError = document.getElementById('client-pin-error');
    const pinInput = document.getElementById('client-pin');

    btn.disabled = true;
    btnText.style.display = 'none';
    btnLoading.style.display = 'inline';
    pinError.innerHTML = '';

    try {
        const formData = new URLSearchParams(new FormData(form));
        const resp = await fetch('/load-message', {
            method: 'POST',
            headers: {'Content-Type': 'application/x-www-form-urlencoded'},
            body: formData
        });

        const contentType = resp.headers.get('Content-Type') || '';
        const isEncryptedBlob = contentType.includes('text/plain');

        if (resp.ok && isEncryptedBlob) {
            const encryptedBlob = await resp.text();
            const result = await decryptAuto(encryptedBlob, cryptoKey);

            if (result.type === 'text') {
                document.getElementById('message-container').innerHTML =
                    '<div class="card decoded-message">' +
                    '<div class="card-header"><h2 class="card-title">Decrypted Message</h2>' +
                    '<p class="card-description">This message has been permanently deleted from the server.</p></div>' +
                    '<div class="form-group"><textarea id="decoded-msg-text" readonly class="message-output">' +
                    escapeHtml(result.text) + '</textarea></div>' +
                    '<div class="form-row two-cols">' +
                    '<button type="button" class="main-btn" data-action="copy-message">Copy</button>' +
                    '<a href="/" class="second-btn">New Secret</a></div></div>';
            } else if (result.type === 'file') {
                const blob = new Blob([result.data], { type: result.contentType });
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = result.filename;
                a.click();
                URL.revokeObjectURL(url);

                document.getElementById('message-container').innerHTML =
                    '<div class="card success-card"><div class="card-header">' +
                    '<h2 class="card-title">File Downloaded</h2></div>' +
                    '<p class="success-message">The file "<span id="downloaded-filename"></span>" has been decrypted and downloaded. ' +
                    'It has been permanently deleted from the server.</p>' +
                    '<a href="/" class="main-btn">Create New Secret</a></div>';
                document.getElementById('downloaded-filename').textContent = result.filename;
            }
        } else {
            const html = await resp.text();
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, 'text/html');

            const pinErr = doc.querySelector('.error');
            if (pinErr) {
                pinError.innerHTML = '<span class="error">' + escapeHtml(pinErr.textContent) + '</span>';
                pinInput.classList.add('error-input');
                pinInput.value = '';
                pinInput.focus();
                btn.disabled = false;
                btnText.style.display = 'inline-flex';
                btnLoading.style.display = 'none';
                return;
            }

            const expiredErr = doc.querySelector('.error-message');
            if (expiredErr) {
                document.getElementById('message-container').innerHTML =
                    '<div class="card error-card"><div class="card-header">' +
                    '<h2 class="card-title">Message Unavailable</h2></div>' +
                    '<p class="error-message">' + escapeHtml(expiredErr.textContent) + '</p>' +
                    '<a href="/" class="main-btn">Back to Main Page</a></div>';
                return;
            }

            pinError.innerHTML = '<span class="error">Failed to load message.</span>';
            btn.disabled = false;
            btnText.style.display = 'inline-flex';
            btnLoading.style.display = 'none';
        }
    } catch (err) {
        pinError.innerHTML = '<span class="error">Decryption failed. The key may be incorrect or data corrupted.</span>';
        pinInput.value = '';
        pinInput.focus();
        btn.disabled = false;
        btnText.style.display = 'inline-flex';
        btnLoading.style.display = 'none';
    }
}

async function handleFileDownload() {
    const form = document.getElementById('download-form');
    const btn = document.getElementById('download-btn');
    const btnText = document.getElementById('download-btn-text');
    const btnLoading = document.getElementById('download-btn-loading');
    const pinError = document.getElementById('pin-error');
    const pinInput = document.getElementById('pin');

    btn.disabled = true;
    btnText.style.display = 'none';
    btnLoading.style.display = 'inline';

    try {
        const formData = new URLSearchParams(new FormData(form));
        const resp = await fetch('/load-message', {
            method: 'POST',
            headers: {'Content-Type': 'application/x-www-form-urlencoded'},
            body: formData
        });

        if (resp.ok && resp.headers.get('Content-Disposition')) {
            const blob = await resp.blob();
            const disposition = resp.headers.get('Content-Disposition');
            const match = disposition.match(/filename="([^"]+)"/);
            const filename = match ? match[1] : 'download';

            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = filename;
            a.click();
            URL.revokeObjectURL(url);

            document.getElementById('message-container').innerHTML =
                '<div class="card success-card">' +
                '<div class="card-header"><h2 class="card-title">File Downloaded</h2></div>' +
                '<p class="success-message">The file has been decrypted, downloaded, and permanently deleted from the server.</p>' +
                '<a href="/" class="main-btn">Create New Secret</a></div>';
        } else {
            const html = await resp.text();
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, 'text/html');

            const pinErr = doc.querySelector('#pin-error');
            if (pinErr && pinErr.innerHTML.trim()) {
                pinError.innerHTML = pinErr.innerHTML;
                pinInput.classList.add('error-input');
                pinInput.value = '';
                pinInput.focus();
                btn.disabled = false;
                btnText.style.display = 'inline-flex';
                btnLoading.style.display = 'none';
                return;
            }

            const expiredErr = doc.querySelector('.error-message');
            if (expiredErr) {
                document.getElementById('message-container').innerHTML =
                    '<div class="card error-card">' +
                    '<div class="card-header"><h2 class="card-title">Message Unavailable</h2></div>' +
                    '<p class="error-message">' + escapeHtml(expiredErr.textContent) + '</p>' +
                    '<a href="/" class="main-btn">Back to Main Page</a></div>';
                return;
            }

            pinError.innerHTML = '<span class="error">An error occurred. Please try again.</span>';
            pinInput.classList.add('error-input');
            pinInput.value = '';
            pinInput.focus();
            btn.disabled = false;
            btnText.style.display = 'inline-flex';
            btnLoading.style.display = 'none';
        }
    } catch (err) {
        pinError.innerHTML = '<span class="error">Download failed. Please try again.</span>';
        pinInput.value = '';
        pinInput.focus();
        btn.disabled = false;
        btnText.style.display = 'inline-flex';
        btnLoading.style.display = 'none';
    }
}

// ============================================================================
// utility functions
// ============================================================================

function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1048576).toFixed(1) + ' MB';
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ============================================================================
// initialization
// ============================================================================

document.addEventListener('DOMContentLoaded', function() {
    setupAutofocusObserver();
    setupPopupHandlers();
    setupNumericInputHandler();
    setupExpireErrorHandler();
    setupCopyHandlers();
    setupFileUploadHandlers();
    setupEncryptionHandlers();
    setupDecryptionHandlers();
});
