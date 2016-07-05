function isButtonNumber(e) {
	var charCode = (e.which) ? e.which : e.keyCode;

	if ((charCode < 48 || charCode > 57)) {
		return false;
	}

	return true;
}

function pinCheck(e) {
	if (!isButtonNumber(e)) {
		e.preventDefault();
		return;
	}
}

function numberCheck(e) {
	if (!isButtonNumber(e)) {
		e.preventDefault();
		return;
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