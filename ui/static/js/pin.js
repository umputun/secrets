document.body.addEventListener("setUpPinInputListeners", function(evt){
    setUpPinInputListeners();
})

// setUpPinInputListeners sets up listeners for the pin input fields to move focus to the next input when the user types a digit
function setUpPinInputListeners() {
    let digitInputs = document.querySelectorAll('.pin-container input');

    digitInputs.forEach((input, index) => {
        input.addEventListener('input', (e) => {
            if (e.inputType === 'deleteContentBackward' && index > 0) {
                // If the user pressed backspace and it's not the first input, move focus to the previous input
                digitInputs[index - 1].focus();
            } else if (index < digitInputs.length - 1 && input.value !== '') {
                // If the input has a value and it's not the last input, move focus to the next input
                digitInputs[index + 1].focus();
            }
        });

        input.addEventListener('keydown', (e) => {
            if (e.key === 'Backspace' && index > 0 && input.value === '') {
                // If the user pressed backspace and it's not the first input and the input is empty, move focus to the previous input
                digitInputs[index - 1].focus();
            }
        });
    });
}

setUpPinInputListeners();