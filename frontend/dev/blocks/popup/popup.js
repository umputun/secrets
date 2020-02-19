function popupInit() {
	var welcomeRead = JSON.parse(localStorage.getItem('welcome')),
		popup = document.getElementById('welcome'),
		popupButton = popup.querySelector('.popup__button');

	popupButton.addEventListener('click', function() {
		localStorage.setItem('welcome', true);
		popup.classList.remove('popup_shown', 'popup_fast');
	});

	if (!welcomeRead && location.pathname.indexOf('show') < 0) {
		popup.classList.add('popup_shown');
	}
}

if (document.readyState != 'loading'){
	popupInit();
} else {
	document.addEventListener('DOMContentLoaded', popupInit);
}