function headerInit() {
	var	popup = document.getElementById('welcome'),
		infoButton = document.getElementById('info');

	infoButton.addEventListener('click', function() {
		popup.classList.add('popup_shown', 'popup_fast');
	});
}

if (document.readyState != 'loading'){
	headerInit();
} else {
	document.addEventListener('DOMContentLoaded', headerInit);
}