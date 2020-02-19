function getCharCode(e) {
	return (e.which) ? e.which : e.keyCode;
}

function isButtonNumber(charCode) {
	if ((charCode < 48 || charCode > 57)) {
		return false;
	}

	return true;
}

function pinCheck(e) {
	var charCode = getCharCode(e);

	if (!isButtonNumber(charCode) || e.target.value.length + 1 > API.params.pin_size) {
		e.preventDefault();
		return;
	}
}

function numberCheck(e) {
	var charCode = getCharCode(e);

	if (!isButtonNumber(charCode)) {
		e.preventDefault();
		return;
	}
}

function getRandomPlaceholder(length) {
	var text = '';
	var possible = '0123456789';

	for (var i = 0; i < length; i++)
		text += possible.charAt(Math.floor(Math.random() * possible.length));

	return text;
}

function numInputsInit() {
	var pins = document.querySelectorAll('.input_type_pin'),
		numbers = document.querySelectorAll('.input_type_number');

	for (var i = pins.length - 1; i >= 0; i--) {
		pins[i].addEventListener('keypress', pinCheck);

		if (pins[i].classList.contains('input_random')) {
			pins[i].placeholder = getRandomPlaceholder(API.params.pin_size);
		}
	}	

	for (var i = numbers.length - 1; i >= 0; i--) {
		numbers[i].addEventListener('keypress', numberCheck);
	}	
}

if (document.readyState != 'loading'){
	numInputsInit();
} else {
	document.addEventListener('DOMContentLoaded', numInputsInit);
}