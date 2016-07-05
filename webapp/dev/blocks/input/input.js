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

	if (!isButtonNumber(charCode) || e.target.value + 1 > API.params.pin_size) {
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

	var newValue = new Number(e.target.value + String.fromCharCode(charCode));

	if (newValue > API.params.max_exp_sec / 60) {
		e.preventDefault();
	}
}

function numInputsInit() {
	var pins = document.querySelectorAll('.input_type_pin'),
		numbers = document.querySelectorAll('.input_type_number');

	for (var i = pins.length - 1; i >= 0; i--) {
		pins[i].addEventListener('keypress', pinCheck);
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