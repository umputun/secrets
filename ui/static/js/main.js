// Global function to close popup
window.closePopup = function() {
    const popup = document.getElementById("popup");
    if (popup) {
        popup.style.transition = "visibility 0s linear 0.33s, opacity 0.33s linear";
        popup.style.opacity = "0";
        popup.style.visibility = "hidden";
    }
};

// Function to initialize popup event handlers
function initializePopupHandlers() {
    // Close popup with close button
    const closeButton = document.getElementById("close-popup");
    if (closeButton && !closeButton.hasAttribute('data-popup-handler-attached')) {
        closeButton.addEventListener("click", window.closePopup);
        closeButton.setAttribute('data-popup-handler-attached', 'true');
    }

    // Close popup when clicking on backdrop
    const popup = document.getElementById("popup");
    if (popup && !popup.hasAttribute('data-backdrop-handler-attached')) {
        popup.addEventListener("click", function(event) {
            if (event.target === this) {
                window.closePopup();
            }
        });
        popup.setAttribute('data-backdrop-handler-attached', 'true');
    }
}

// Initialize handlers when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializePopupHandlers);
} else {
    // DOM is already ready, initialize immediately
    initializePopupHandlers();
}

// Also initialize handlers after a short delay to catch dynamically loaded content
setTimeout(initializePopupHandlers, 100);