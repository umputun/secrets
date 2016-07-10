function buttonInit() {
	var copyButtons = new Clipboard('.button_content_copy');

	copyButtons.on('success', function(e) {
		e.trigger.textContent = 'Copied!';
		e.trigger.disabled = true;

		e.clearSelection();
	});

	copyButtons.on('error', function(e) {
		e.trigger.textContent = 'Can\'t copy :(';
		e.trigger.disabled = true;
	});
}

if (document.readyState != 'loading'){
	buttonInit();
} else {
	document.addEventListener('DOMContentLoaded', buttonInit);
}