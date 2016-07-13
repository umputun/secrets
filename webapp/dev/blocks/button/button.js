function buttonInit() {
	var copyButtonsSelector = '.button_content_copy';

	if (typeof document.queryCommandSupported === 'function'
		&& document.queryCommandSupported('copy')) {
		var copyButtons = new Clipboard(copyButtonsSelector);

		copyButtons.on('success', function(e) {
			e.trigger.textContent = 'Copied!';
			e.trigger.disabled = true;

			e.clearSelection();
		});

		copyButtons.on('error', function(e) {
			e.trigger.textContent = 'Can\'t copy :(';
			e.trigger.disabled = true;
		});
	} else {
		var copyButtons = document.querySelectorAll(copyButtonsSelector);

		for (var i = copyButtons.length - 1; i >= 0; i--) {
			copyButtons[i].classList.remove('button_shown');
		}
	}
	
}

if (document.readyState != 'loading'){
	buttonInit();
} else {
	document.addEventListener('DOMContentLoaded', buttonInit);
}