function popupInit() {
	if (location.pathname.indexOf('show') > -1) return;

	var welcomeRead = JSON.parse(localStorage.getItem('welcome')),
		popup = document.getElementById('welcome'),
		popupButton = popup.querySelector('.popup__button');

	if (!welcomeRead) {
		popup.classList.add('popup_shown');

		popupButton.addEventListener('click', function() {
			localStorage.setItem('welcome', true);
			popup.classList.remove('popup_shown');
		})
	}
}

if (document.readyState != 'loading'){
	popupInit();
} else {
	document.addEventListener('DOMContentLoaded', popupInit);
}