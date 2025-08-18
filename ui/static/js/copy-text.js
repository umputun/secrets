document.querySelectorAll("[data-copy]")
    .forEach(el => {
        let copyBtn = el.querySelector("[data-copy-btn]");

        copyBtn.addEventListener("click", async function () {
            const textarea = el.querySelector("[data-copy-text]");
            const textToCopy = textarea.value;
            const originalContent = copyBtn.innerHTML;

            if (textToCopy) {
                try {
                    // Use modern Clipboard API if available
                    if (navigator.clipboard && window.isSecureContext) {
                        await navigator.clipboard.writeText(textToCopy);
                    } else {
                        // Fallback for older browsers
                        const textArea = document.createElement("textarea");
                        textArea.value = textToCopy;
                        document.body.appendChild(textArea);
                        textArea.focus();
                        textArea.select();
                        document.execCommand('copy');
                        document.body.removeChild(textArea);
                    }

                    // Visual feedback - update button temporarily
                    copyBtn.innerHTML = `
                        <svg class="btn-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                            <polyline points="20,6 9,17 4,12" stroke="currentColor" stroke-width="2"/>
                        </svg>
                        <span class="btn-text">Copied!</span>
                    `;
                    copyBtn.style.background = 'var(--color-success)';

                    // Reset button after 2 seconds
                    setTimeout(() => {
                        copyBtn.innerHTML = originalContent;
                        copyBtn.style.background = 'var(--color-primary)';
                    }, 2000);

                    // Show popup notification
                    let popup = document.getElementById("popup");
                    const popupText = popup.querySelector(".popup-text");

                    popup.style.opacity = 1;
                    popup.style.visibility = "visible";
                    popup.style.animation = "fadeInOut 3s ease-in-out";

                    if (copyBtn.id === "copy-link-btn") {
                        popupText.innerHTML = `<strong>Link copied!</strong><br/>Share this link to access your secret content`;
                    } else {
                        popupText.innerHTML = `<strong>Message copied!</strong>`;
                    }

                } catch (err) {
                    console.error('Failed to copy text: ', err);
                    // Show error feedback
                    copyBtn.innerHTML = `
                        <svg class="btn-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                            <line x1="18" y1="6" x2="6" y2="18" stroke="currentColor" stroke-width="2"/>
                            <line x1="6" y1="6" x2="18" y2="18" stroke="currentColor" stroke-width="2"/>
                        </svg>
                        <span class="btn-text">Error!</span>
                    `;
                    copyBtn.style.background = 'var(--color-error)';
                    
                    setTimeout(() => {
                        copyBtn.innerHTML = originalContent;
                        copyBtn.style.background = 'var(--color-primary)';
                    }, 2000);
                }
            }
        });
    });