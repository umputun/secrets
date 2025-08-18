document.body.addEventListener("setUpPinInputListeners", function(evt){
    setUpPinInputListeners();
})

// setUpPinInputListeners sets up listeners for the pin input field to ensure only numeric input and proper formatting
function setUpPinInputListeners() {
    const pinInputs = document.querySelectorAll('.pin-input');

    pinInputs.forEach((input) => {
        // Only allow numeric input
        input.addEventListener('input', (e) => {
            let value = e.target.value.replace(/\D/g, ''); // Remove non-digits
            
            // Limit to 5 digits
            if (value.length > 5) {
                value = value.slice(0, 5);
            }
            
            e.target.value = value;
        });

        // Prevent non-numeric key presses (except control keys)
        input.addEventListener('keydown', (e) => {
            const allowedKeys = [
                'Backspace', 'Delete', 'Tab', 'Escape', 'Enter',
                'ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown',
                'Home', 'End'
            ];
            
            const isNumeric = /^[0-9]$/.test(e.key);
            const isAllowed = allowedKeys.includes(e.key);
            const isCtrlCmd = e.ctrlKey || e.metaKey;
            
            if (!isNumeric && !isAllowed && !isCtrlCmd) {
                e.preventDefault();
            }
        });

        // Handle paste events to only allow numeric content
        input.addEventListener('paste', (e) => {
            e.preventDefault();
            const paste = (e.clipboardData || window.clipboardData).getData('text');
            const numericPaste = paste.replace(/\D/g, '').slice(0, 5);
            e.target.value = numericPaste;
        });

        // Auto-focus behavior for better UX
        input.addEventListener('focus', (e) => {
            // Select all text when focused for easy replacement
            setTimeout(() => e.target.select(), 10);
        });
    });
}

setUpPinInputListeners();