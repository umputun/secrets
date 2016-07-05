/**
 * fullscreenForm.js v1.0.0
 * http://www.codrops.com
 *
 * Licensed under the MIT license.
 * http://www.opensource.org/licenses/mit-license.php
 * 
 * Copyright 2014, Codrops
 * http://www.codrops.com
 * 
 * BEMificated & edited by Igor Adamenko
 * http://igoradamenko.com
 */
;(function(window) {
	
	'use strict';

	var support = { animations: Modernizr.cssanimations },
		animEndEventNames = { 'WebkitAnimation': 'webkitAnimationEnd', 'OAnimation': 'oAnimationEnd', 'msAnimation': 'MSAnimationEnd', 'animation': 'animationend' },
		// animation end event name
		animEndEventName = animEndEventNames[ Modernizr.prefixed('animation') ];

	/**
	 * extend obj function
	 */
	function extend(a, b) {
		for (var key in b) { 
			if (b.hasOwnProperty(key)) {
				a[key] = b[key];
			}
		}
		return a;
	}

	/**
	 * createElement function
	 * creates an element with tag = tag, className = opt.cName, innerHTML = opt.inner and appends it to opt.appendTo
	 */
	function createElement(tag, opt) {
		var el = document.createElement(tag)
		if (opt) {
			if (opt.cName) {
				el.className = opt.cName;
			}
			if (opt.inner) {
				el.innerHTML = opt.inner;
			}
			if (opt.appendTo) {
				opt.appendTo.appendChild(el);
			}
		}	
		return el;
	}

	/**
	 * FForm function
	 */
	function FForm(el, options) {
		this.el = el;
		this.options = extend({}, this.options);
  		extend(this.options, options);
  		this._init();
	}

	/**
	 * FForm options
	 */
	FForm.prototype.options = {
		// show progress bar
		ctrlProgress: true,
		// show navigation dots
		ctrlNavDots: true,
		// show [current field]/[total fields] status
		ctrlNavPosition: true,
		// reached the review and submit step
		onReview: function() { return false; }
	};

	/**
	 * init function
	 * initialize and cache some vars
	 */
	FForm.prototype._init = function() {
		// the form element
		this.formEl = this.el.querySelector('.fs-form__content');

		// list of fields
		this.fieldsList = this.formEl.querySelector('ol.fields');

		// current field position
		this.current = 0;

		// all fields
		this.fields = [].slice.call(this.fieldsList.children);
		
		// total fields
		this.fieldsCount = this.fields.length;
		
		// show first field
		classie.add(this.fields[ this.current ], 'field field_current');

		// create/add controls
		this._addControls();

		// create/add messages
		this._addErrorMsg();
		
		// init events
		this._initEvents();
	};

	/**
	 * addControls function
	 * create and insert the structure for the controls
	 */
	FForm.prototype._addControls = function() {
		// main controls wrapper
		this.ctrls = createElement('div', { cName: 'controls', appendTo: this.el });

		// continue button (jump to next field)
		this.ctrlContinue = createElement('button', { cName: 'continue', inner: 'Continue', appendTo: this.ctrls });
		this._showCtrl(this.ctrlContinue);

		// navigation dots
		if (this.options.ctrlNavDots) {
			this.ctrlNav = createElement('nav', { cName: 'nav-dots', appendTo: this.ctrls });
			var dots = '';
			for (var i = 0; i < this.fieldsCount; ++i) {
				dots += i === this.current ? '<button class="nav-dots__item nav-dots__item_current"></button>': '<button class="nav-dots__item" disabled></button>';
			}
			this.ctrlNav.innerHTML = dots;
			this._showCtrl(this.ctrlNav);
			this.ctrlNavDots = [].slice.call(this.ctrlNav.children);
		}

		// field number status
		if (this.options.ctrlNavPosition) {
			this.ctrlFldStatus = createElement('span', { cName: 'numbers', appendTo: this.ctrls });

			// current field placeholder
			this.ctrlFldStatusCurr = createElement('span', { cName: 'numbers__item numbers__item_current', inner: Number(this.current + 1) });
			this.ctrlFldStatus.appendChild(this.ctrlFldStatusCurr);

			// total fields placeholder
			this.ctrlFldStatusTotal = createElement('span', { cName: 'numbers__item', inner: this.fieldsCount });
			this.ctrlFldStatus.appendChild(this.ctrlFldStatusTotal);
			this._showCtrl(this.ctrlFldStatus);
		}

		// progress bar
		if (this.options.ctrlProgress) {
			this.ctrlProgress = createElement('div', { cName: 'progress', appendTo: this.ctrls });
			this._showCtrl(this.ctrlProgress);
		}
	}

	/**
	 * addErrorMsg function
	 * create and insert the structure for the error message
	 */
	FForm.prototype._addErrorMsg = function() {
		// error message
		this.msgError = createElement('span', { cName: 'error', appendTo: this.el });
	}

	/**
	 * init events
	 */
	FForm.prototype._initEvents = function() {
		var self = this;

		// show next field
		this.ctrlContinue.addEventListener('click', function() {
			self._nextField(); 
		});

		// navigation dots
		if (this.options.ctrlNavDots) {
			this.ctrlNavDots.forEach(function(dot, pos) {
				dot.addEventListener('click', function() {
					self._showField(pos);
				});
			});
		}

		// keyboard navigation events - jump to next field when pressing enter
		document.addEventListener('keydown', function(ev) {
			if (!self.isLastStep) {
				var keyCode = ev.keyCode || ev.which,
					tagName = ev.target.tagName.toLowerCase(),
					isMetaPressed = ev.metaKey || ev.ctrlKey;
				
				if (keyCode === 13 && (tagName !== 'textarea' || tagName === 'textarea' && isMetaPressed)) {
					ev.preventDefault();
					self._nextField();
				}
			}
		});
	};

	/**
	 * nextField function
	 * jumps to the next field
	 */
	FForm.prototype._nextField = function(backto) {
		if (this.isLastStep || !this._validade() || this.isAnimating) {
			return false;
		}
		this.isAnimating = true;

		// check if on last step
		this.isLastStep = this.current === this.fieldsCount - 1 && backto === undefined ? true: false;
		
		// clear any previous error messages
		this._clearError();

		// current field
		var currentFld = this.fields[ this.current ];

		// save the navigation direction
		this.navdir = backto !== undefined ? backto < this.current ? 'prev': 'next': 'next';

		// update current field
		this.current = backto !== undefined ? backto: this.current + 1;

		if (backto === undefined) {
			// update progress bar (unless we navigate backwards)
			this._progress();

			// save farthest position so far
			this.farthest = this.current;
		}

		// add class "fs-display-next" or "fs-display-prev" to the list of fields
		classie.add(this.fieldsList, 'fields fields_' + this.navdir);

		// remove class "fs-current" from current field and add it to the next one
		// also add class "fs-show" to the next field and the class "fs-hide" to the current one
		classie.remove(currentFld, 'field_current');
		classie.add(currentFld, 'field_hide');
		
		if (!this.isLastStep) {
			// update nav
			this._updateNav();

			// change the current field number/status
			this._updateFieldNumber();

			var nextField = this.fields[ this.current ];
			classie.add(nextField, 'field_current');
			classie.add(nextField, 'field_shown');
		}

		// after animation ends remove added classes from fields
		var self = this,
			onEndAnimationFn = function(ev) {
				if (support.animations) {
					this.removeEventListener(animEndEventName, onEndAnimationFn);
				}
				
				classie.remove(self.fieldsList, 'fields_' + self.navdir);
				classie.remove(currentFld, 'field_hide');

				if (self.isLastStep) {
					// show the complete form and hide the controls
					self._hideCtrl(self.ctrlNav);
					self._hideCtrl(self.ctrlProgress);
					self._hideCtrl(self.ctrlContinue);
					self._hideCtrl(self.ctrlFldStatus);
					// replace class fs-form-full with fs-form-overview
					classie.remove(self.formEl, 'fs-form__content_full');
					classie.add(self.formEl, 'fs-form__content_overview');
					classie.add(self.formEl, 'form__content_shown');
					// callback
					self.options.onReview();
				}
				else {
					classie.remove(nextField, 'field_shown');
					
					if (self.options.ctrlNavPosition) {
						self.ctrlFldStatusCurr.innerHTML = self.ctrlFldStatusNew.innerHTML;
						self.ctrlFldStatus.removeChild(self.ctrlFldStatusNew);
						classie.remove(self.ctrlFldStatus, 'numbers_shown-' + self.navdir);
					}

					nextField.querySelectorAll('.input, .textarea')[0].focus();
				}

				self.isAnimating = false;
			};

		if (support.animations) {
			if (this.navdir === 'next') {
				if (this.isLastStep) {
					currentFld.querySelector('.animation__upper').addEventListener(animEndEventName, onEndAnimationFn);
				}
				else {
					nextField.querySelector('.animation__lower').addEventListener(animEndEventName, onEndAnimationFn);
				}
			}
			else {
				nextField.querySelector('.animation__upper').addEventListener(animEndEventName, onEndAnimationFn);
			}
		}
		else {
			onEndAnimationFn();
		}
	}

	/**
	 * showField function
	 * jumps to the field at position pos
	 */
	FForm.prototype._showField = function(pos) {
		if (pos === this.current || pos < 0 || pos > this.fieldsCount - 1) {
			return false;
		}
		this._nextField(pos);
	}

	/**
	 * updateFieldNumber function
	 * changes the current field number
	 */
	FForm.prototype._updateFieldNumber = function() {
		if (this.options.ctrlNavPosition) {
			// first, create next field number placeholder
			this.ctrlFldStatusNew = document.createElement('span');
			this.ctrlFldStatusNew.className = 'numbers__item numbers__item_new';
			this.ctrlFldStatusNew.innerHTML = Number(this.current + 1);
			
			// insert it in the DOM
			this.ctrlFldStatus.appendChild(this.ctrlFldStatusNew);
			
			// add class "fs-show-next" or "fs-show-prev" depending on the navigation direction
			var self = this;
			setTimeout(function() {
				classie.add(self.ctrlFldStatus, self.navdir === 'next' ? 'numbers_shown-next': 'numbers_shown-prev');
			}, 25);
		}
	}

	/**
	 * progress function
	 * updates the progress bar by setting its width
	 */
	FForm.prototype._progress = function() {
		if (this.options.ctrlProgress) {
			this.ctrlProgress.style.width = this.current * (100 / this.fieldsCount) + '%';
		}
	}

	/**
	 * updateNav function
	 * updates the navigation dots
	 */
	FForm.prototype._updateNav = function() {
		if (this.options.ctrlNavDots) {
			classie.remove(this.ctrlNav.querySelector('button.nav-dots__item_current'), 'nav-dots__item_current');
			classie.add(this.ctrlNavDots[ this.current ], 'nav-dots__item_current');
			this.ctrlNavDots[ this.current ].disabled = false;
		}
	}

	/**
	 * showCtrl function
	 * shows a control
	 */
	FForm.prototype._showCtrl = function(ctrl) {
		classie.add(ctrl, 'shown');
	}

	/**
	 * hideCtrl function
	 * hides a control
	 */
	FForm.prototype._hideCtrl = function(ctrl) {
		classie.remove(ctrl, 'shown');
	}

	// TODO: this is a very basic validation function. Only checks for required fields..
	FForm.prototype._validade = function() {
		var fld = this.fields[ this.current ],
			input = fld.querySelector('.input[required]') || fld.querySelector('.textarea[required]'),
			error;

		if (!input) return true;

		if (input.value.length == 0) {
			error = 'NOVAL';
		} else if (input.classList.contains('input_type_number')) {
			var num = new Number(input.value),
				maxNum = Math.floor(API.params.max_exp_sec / 60);

			if (input.value.substr(0, 1) == '-') {
				error = 'NONEG';
			} else if (input.value != num.toString()) {
				error = 'WRONGVAL';
			} else if (num > maxNum) {
				error = 'MAXNUM';
			}
		} else if (input.classList.contains('input_type_pin')) {
			if (!input.value.match(/^[0-9]{5}$/)) {
				error = 'WRONGVAL';
			} else if (input.value.length != API.params.pin_size) {
				error = 'PINSIZE';
			}
		}

		if (error != undefined) {
			this._showError(error);
			return false;
		}

		return true;
	}

	// TODO
	FForm.prototype._showError = function(err) {
		var message = '';
		switch(err) {
			case 'NOVAL': 
				message = 'Please fill the field before continuing';
				break;
			case 'NONEG': 
				message = 'This value can\'t be negative';
				break;
			case 'WRONGVAL': 
				message = 'Please enter the valid value';
				break;
			case 'MAXNUM':
				message = 'Maximum keeping time is ' + Math.floor(API.params.max_exp_sec / 60) + ' minutes';
				break;
			case 'MAXPINSIZE':
				message = 'PIN length must be ' + API.params.pin_size + ' symbols';
		};

		this.msgError.innerHTML = message;
		this._showCtrl(this.msgError);
	}

	// clears/hides the current error message
	FForm.prototype._clearError = function() {
		this._hideCtrl(this.msgError);
	}

	// add to global namespace
	window.FForm = FForm;

})(window);

function formInit() {
	function getLink() {
		var message = document.getElementById('text').value,
			exp = document.getElementById('time').value,
			pin = document.getElementById('pin').value;

		API.send(exp, message, pin, function(data) {
			var link = location.protocol + '//' + location.host + '/show/' + data.key,
				resField = document.getElementById('result__info');

			resField.value = link;
			resField.focus();
			resField.select();
		});
	}

	function getInfo() {
		var pin = document.getElementById('pin').value,
			key = location.pathname.replace('/show/', '');

		API.get(key, pin, function(data) {
			var resField = document.getElementById('result__info');

			document.getElementById('result__tip').textContent = 'Here is your info:';
			resField.value = data.message;
			resField.focus();
			resField.select();
		}, function() {
			var tip = document.getElementById('result__tip');

			tip.classList.add('result__tip_error');
			tip.textContent = 'Sorry, but your PIN is wrong.';
		});
	}

	new FForm(document.getElementById('fs-form'), {
		onReview: location.pathname.indexOf('show') > 0 ? getInfo: getLink
	});
}

if (document.readyState != 'loading'){
	formInit();
} else {
	document.addEventListener('DOMContentLoaded', formInit);
}

