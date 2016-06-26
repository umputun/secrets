var API = (function() {
	var xhrPath = 'https://safesecret.info/api/v1/message';

	this.send = function(exp, message, pin, cb) {
		var request = new XMLHttpRequest(),
			data = JSON.stringify({
				exp: new Number(exp) * 60,
				message: new String(message),
				pin: new String(pin)
			});

		request.open('POST', xhrPath, true);

		request.onreadystatechange = function() {
			if (request.readyState == 4 && typeof cb == 'function') {
				if (request.status == 201) {
					cb(JSON.parse(request.responseText));
				}
  			}
		};

		request.send(data);
	}

	this.get = function(uid, pin, cb, err) {
		var request = new XMLHttpRequest();

		request.open('GET', [xhrPath, uid, pin].join('/'), true);

		request.onreadystatechange = function() {
			if (request.readyState == 4 && typeof cb == 'function') {
				if (request.status == 200) {
					cb(JSON.parse(request.responseText));
				} else {
					err();
				}
  			}
		};

		request.send();
	}

	return this;
})();
