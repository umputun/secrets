/* Modernizr 2.8.3 (Custom Build) | MIT & BSD
 * Build: http://modernizr.com/download/#-cssanimations-shiv-cssclasses-prefixed-teststyles-testprop-testallprops-prefixes-domprefixes-css_calc-css_vhunit-load
 */
;window.Modernizr=function(a,b,c){function z(a){j.cssText=a}function A(a,b){return z(m.join(a+";")+(b||""))}function B(a,b){return typeof a===b}function C(a,b){return!!~(""+a).indexOf(b)}function D(a,b){for(var d in a){var e=a[d];if(!C(e,"-")&&j[e]!==c)return b=="pfx"?e:!0}return!1}function E(a,b,d){for(var e in a){var f=b[a[e]];if(f!==c)return d===!1?a[e]:B(f,"function")?f.bind(d||b):f}return!1}function F(a,b,c){var d=a.charAt(0).toUpperCase()+a.slice(1),e=(a+" "+o.join(d+" ")+d).split(" ");return B(b,"string")||B(b,"undefined")?D(e,b):(e=(a+" "+p.join(d+" ")+d).split(" "),E(e,b,c))}var d="2.8.3",e={},f=!0,g=b.documentElement,h="modernizr",i=b.createElement(h),j=i.style,k,l={}.toString,m=" -webkit- -moz- -o- -ms- ".split(" "),n="Webkit Moz O ms",o=n.split(" "),p=n.toLowerCase().split(" "),q={},r={},s={},t=[],u=t.slice,v,w=function(a,c,d,e){var f,i,j,k,l=b.createElement("div"),m=b.body,n=m||b.createElement("body");if(parseInt(d,10))while(d--)j=b.createElement("div"),j.id=e?e[d]:h+(d+1),l.appendChild(j);return f=["&#173;",'<style id="s',h,'">',a,"</style>"].join(""),l.id=h,(m?l:n).innerHTML+=f,n.appendChild(l),m||(n.style.background="",n.style.overflow="hidden",k=g.style.overflow,g.style.overflow="hidden",g.appendChild(n)),i=c(l,a),m?l.parentNode.removeChild(l):(n.parentNode.removeChild(n),g.style.overflow=k),!!i},x={}.hasOwnProperty,y;!B(x,"undefined")&&!B(x.call,"undefined")?y=function(a,b){return x.call(a,b)}:y=function(a,b){return b in a&&B(a.constructor.prototype[b],"undefined")},Function.prototype.bind||(Function.prototype.bind=function(b){var c=this;if(typeof c!="function")throw new TypeError;var d=u.call(arguments,1),e=function(){if(this instanceof e){var a=function(){};a.prototype=c.prototype;var f=new a,g=c.apply(f,d.concat(u.call(arguments)));return Object(g)===g?g:f}return c.apply(b,d.concat(u.call(arguments)))};return e}),q.cssanimations=function(){return F("animationName")};for(var G in q)y(q,G)&&(v=G.toLowerCase(),e[v]=q[G](),t.push((e[v]?"":"no-")+v));return e.addTest=function(a,b){if(typeof a=="object")for(var d in a)y(a,d)&&e.addTest(d,a[d]);else{a=a.toLowerCase();if(e[a]!==c)return e;b=typeof b=="function"?b():b,typeof f!="undefined"&&f&&(g.className+=" "+(b?"":"no-")+a),e[a]=b}return e},z(""),i=k=null,function(a,b){function l(a,b){var c=a.createElement("p"),d=a.getElementsByTagName("head")[0]||a.documentElement;return c.innerHTML="x<style>"+b+"</style>",d.insertBefore(c.lastChild,d.firstChild)}function m(){var a=s.elements;return typeof a=="string"?a.split(" "):a}function n(a){var b=j[a[h]];return b||(b={},i++,a[h]=i,j[i]=b),b}function o(a,c,d){c||(c=b);if(k)return c.createElement(a);d||(d=n(c));var g;return d.cache[a]?g=d.cache[a].cloneNode():f.test(a)?g=(d.cache[a]=d.createElem(a)).cloneNode():g=d.createElem(a),g.canHaveChildren&&!e.test(a)&&!g.tagUrn?d.frag.appendChild(g):g}function p(a,c){a||(a=b);if(k)return a.createDocumentFragment();c=c||n(a);var d=c.frag.cloneNode(),e=0,f=m(),g=f.length;for(;e<g;e++)d.createElement(f[e]);return d}function q(a,b){b.cache||(b.cache={},b.createElem=a.createElement,b.createFrag=a.createDocumentFragment,b.frag=b.createFrag()),a.createElement=function(c){return s.shivMethods?o(c,a,b):b.createElem(c)},a.createDocumentFragment=Function("h,f","return function(){var n=f.cloneNode(),c=n.createElement;h.shivMethods&&("+m().join().replace(/[\w\-]+/g,function(a){return b.createElem(a),b.frag.createElement(a),'c("'+a+'")'})+");return n}")(s,b.frag)}function r(a){a||(a=b);var c=n(a);return s.shivCSS&&!g&&!c.hasCSS&&(c.hasCSS=!!l(a,"article,aside,dialog,figcaption,figure,footer,header,hgroup,main,nav,section{display:block}mark{background:#FF0;color:#000}template{display:none}")),k||q(a,c),a}var c="3.7.0",d=a.html5||{},e=/^<|^(?:button|map|select|textarea|object|iframe|option|optgroup)$/i,f=/^(?:a|b|code|div|fieldset|h1|h2|h3|h4|h5|h6|i|label|li|ol|p|q|span|strong|style|table|tbody|td|th|tr|ul)$/i,g,h="_html5shiv",i=0,j={},k;(function(){try{var a=b.createElement("a");a.innerHTML="<xyz></xyz>",g="hidden"in a,k=a.childNodes.length==1||function(){b.createElement("a");var a=b.createDocumentFragment();return typeof a.cloneNode=="undefined"||typeof a.createDocumentFragment=="undefined"||typeof a.createElement=="undefined"}()}catch(c){g=!0,k=!0}})();var s={elements:d.elements||"abbr article aside audio bdi canvas data datalist details dialog figcaption figure footer header hgroup main mark meter nav output progress section summary template time video",version:c,shivCSS:d.shivCSS!==!1,supportsUnknownElements:k,shivMethods:d.shivMethods!==!1,type:"default",shivDocument:r,createElement:o,createDocumentFragment:p};a.html5=s,r(b)}(this,b),e._version=d,e._prefixes=m,e._domPrefixes=p,e._cssomPrefixes=o,e.testProp=function(a){return D([a])},e.testAllProps=F,e.testStyles=w,e.prefixed=function(a,b,c){return b?F(a,b,c):F(a,"pfx")},g.className=g.className.replace(/(^|\s)no-js(\s|$)/,"$1$2")+(f?" js "+t.join(" "):""),e}(this,this.document),function(a,b,c){function d(a){return"[object Function]"==o.call(a)}function e(a){return"string"==typeof a}function f(){}function g(a){return!a||"loaded"==a||"complete"==a||"uninitialized"==a}function h(){var a=p.shift();q=1,a?a.t?m(function(){("c"==a.t?B.injectCss:B.injectJs)(a.s,0,a.a,a.x,a.e,1)},0):(a(),h()):q=0}function i(a,c,d,e,f,i,j){function k(b){if(!o&&g(l.readyState)&&(u.r=o=1,!q&&h(),l.onload=l.onreadystatechange=null,b)){"img"!=a&&m(function(){t.removeChild(l)},50);for(var d in y[c])y[c].hasOwnProperty(d)&&y[c][d].onload()}}var j=j||B.errorTimeout,l=b.createElement(a),o=0,r=0,u={t:d,s:c,e:f,a:i,x:j};1===y[c]&&(r=1,y[c]=[]),"object"==a?l.data=c:(l.src=c,l.type=a),l.width=l.height="0",l.onerror=l.onload=l.onreadystatechange=function(){k.call(this,r)},p.splice(e,0,u),"img"!=a&&(r||2===y[c]?(t.insertBefore(l,s?null:n),m(k,j)):y[c].push(l))}function j(a,b,c,d,f){return q=0,b=b||"j",e(a)?i("c"==b?v:u,a,b,this.i++,c,d,f):(p.splice(this.i++,0,a),1==p.length&&h()),this}function k(){var a=B;return a.loader={load:j,i:0},a}var l=b.documentElement,m=a.setTimeout,n=b.getElementsByTagName("script")[0],o={}.toString,p=[],q=0,r="MozAppearance"in l.style,s=r&&!!b.createRange().compareNode,t=s?l:n.parentNode,l=a.opera&&"[object Opera]"==o.call(a.opera),l=!!b.attachEvent&&!l,u=r?"object":l?"script":"img",v=l?"script":u,w=Array.isArray||function(a){return"[object Array]"==o.call(a)},x=[],y={},z={timeout:function(a,b){return b.length&&(a.timeout=b[0]),a}},A,B;B=function(a){function b(a){var a=a.split("!"),b=x.length,c=a.pop(),d=a.length,c={url:c,origUrl:c,prefixes:a},e,f,g;for(f=0;f<d;f++)g=a[f].split("="),(e=z[g.shift()])&&(c=e(c,g));for(f=0;f<b;f++)c=x[f](c);return c}function g(a,e,f,g,h){var i=b(a),j=i.autoCallback;i.url.split(".").pop().split("?").shift(),i.bypass||(e&&(e=d(e)?e:e[a]||e[g]||e[a.split("/").pop().split("?")[0]]),i.instead?i.instead(a,e,f,g,h):(y[i.url]?i.noexec=!0:y[i.url]=1,f.load(i.url,i.forceCSS||!i.forceJS&&"css"==i.url.split(".").pop().split("?").shift()?"c":c,i.noexec,i.attrs,i.timeout),(d(e)||d(j))&&f.load(function(){k(),e&&e(i.origUrl,h,g),j&&j(i.origUrl,h,g),y[i.url]=2})))}function h(a,b){function c(a,c){if(a){if(e(a))c||(j=function(){var a=[].slice.call(arguments);k.apply(this,a),l()}),g(a,j,b,0,h);else if(Object(a)===a)for(n in m=function(){var b=0,c;for(c in a)a.hasOwnProperty(c)&&b++;return b}(),a)a.hasOwnProperty(n)&&(!c&&!--m&&(d(j)?j=function(){var a=[].slice.call(arguments);k.apply(this,a),l()}:j[n]=function(a){return function(){var b=[].slice.call(arguments);a&&a.apply(this,b),l()}}(k[n])),g(a[n],j,b,n,h))}else!c&&l()}var h=!!a.test,i=a.load||a.both,j=a.callback||f,k=j,l=a.complete||f,m,n;c(h?a.yep:a.nope,!!i),i&&c(i)}var i,j,l=this.yepnope.loader;if(e(a))g(a,0,l,0);else if(w(a))for(i=0;i<a.length;i++)j=a[i],e(j)?g(j,0,l,0):w(j)?B(j):Object(j)===j&&h(j,l);else Object(a)===a&&h(a,l)},B.addPrefix=function(a,b){z[a]=b},B.addFilter=function(a){x.push(a)},B.errorTimeout=1e4,null==b.readyState&&b.addEventListener&&(b.readyState="loading",b.addEventListener("DOMContentLoaded",A=function(){b.removeEventListener("DOMContentLoaded",A,0),b.readyState="complete"},0)),a.yepnope=k(),a.yepnope.executeStack=h,a.yepnope.injectJs=function(a,c,d,e,i,j){var k=b.createElement("script"),l,o,e=e||B.errorTimeout;k.src=a;for(o in d)k.setAttribute(o,d[o]);c=j?h:c||f,k.onreadystatechange=k.onload=function(){!l&&g(k.readyState)&&(l=1,c(),k.onload=k.onreadystatechange=null)},m(function(){l||(l=1,c(1))},e),i?k.onload():n.parentNode.insertBefore(k,n)},a.yepnope.injectCss=function(a,c,d,e,g,i){var e=b.createElement("link"),j,c=i?h:c||f;e.href=a,e.rel="stylesheet",e.type="text/css";for(j in d)e.setAttribute(j,d[j]);g||(n.parentNode.insertBefore(e,n),m(c,0))}}(this,document),Modernizr.load=function(){yepnope.apply(window,[].slice.call(arguments,0))},Modernizr.addTest("csscalc",function(){var a="width:",b="calc(10px);",c=document.createElement("div");return c.style.cssText=a+Modernizr._prefixes.join(b+a),!!c.style.length}),Modernizr.addTest("cssvhunit",function(){var a;return Modernizr.testStyles("#modernizr { height: 50vh; }",function(b,c){var d=parseInt(window.innerHeight/2,10),e=parseInt((window.getComputedStyle?getComputedStyle(b,null):b.currentStyle).height,10);a=e==d}),a});
var API = (function() {
	var that = this,
		basePath = 'https://safesecret.info/api/v1/';

	var xhrPath = basePath + 'message',
		paramsPath = basePath + 'params';

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
				} else if (typeof err == 'function') {
					if (request.responseText.length > 0) {
						err(JSON.parse(request.responseText));
					} else {
						err({error: "something gone wrong"});
					}
				}
  			}
		};

		request.send();
	}

	this._getParams = function() {
		var request = new XMLHttpRequest();

		request.open('GET', paramsPath, true);

		request.onreadystatechange = function() {
			if (request.readyState == 4) {
				if (request.status == 200) {
					that.params = JSON.parse(request.responseText);
				}
  			}
		};

		request.send();
	}

	// default values
	this.params = {
		pin_size: 5,
		max_pin_attempts: 3,
		max_exp_sec: 86400
	};

	this._getParams();

	return this;
})();

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
/*!
 * classie - class helper functions
 * from bonzo https://github.com/ded/bonzo
 * 
 * classie.has( elem, 'my-class' ) -> true/false
 * classie.add( elem, 'my-new-class' )
 * classie.remove( elem, 'my-unwanted-class' )
 * classie.toggle( elem, 'my-class' )
 */

/*jshint browser: true, strict: true, undef: true */
/*global define: false */

( function( window ) {

'use strict';

// class helper functions from bonzo https://github.com/ded/bonzo

function classReg( className ) {
  return new RegExp("(^|\\s+)" + className + "(\\s+|$)");
}

// classList support for class management
// altho to be fair, the api sucks because it won't accept multiple classes at once
var hasClass, addClass, removeClass;

if ( 'classList' in document.documentElement ) {
  hasClass = function( elem, c ) {
    return elem.classList.contains( c );
  };
  addClass = function( elem, c ) {
    var classList = [].concat(c.split(' '));

    for (var i = classList.length - 1; i >= 0; i--) {
      elem.classList.add(classList[i]);
    }
  };
  removeClass = function( elem, c ) {
    elem.classList.remove( c );
  };
}
else {
  hasClass = function( elem, c ) {
    return classReg( c ).test( elem.className );
  };
  addClass = function( elem, c ) {
    if ( !hasClass( elem, c ) ) {
      elem.className = elem.className + ' ' + c;
    }
  };
  removeClass = function( elem, c ) {
    elem.className = elem.className.replace( classReg( c ), ' ' );
  };
}

function toggleClass( elem, c ) {
  var fn = hasClass( elem, c ) ? removeClass : addClass;
  fn( elem, c );
}

var classie = {
  // full names
  hasClass: hasClass,
  addClass: addClass,
  removeClass: removeClass,
  toggleClass: toggleClass,
  // short names
  has: hasClass,
  add: addClass,
  remove: removeClass,
  toggle: toggleClass
};

// transport
if ( typeof define === 'function' && define.amd ) {
  // AMD
  define( classie );
} else {
  // browser global
  window.classie = classie;
}

})( window );

/*!
 * clipboard.js v1.5.12
 * https://zenorocha.github.io/clipboard.js
 *
 * Licensed MIT © Zeno Rocha
 */
!function(t){if("object"==typeof exports&&"undefined"!=typeof module)module.exports=t();else if("function"==typeof define&&define.amd)define([],t);else{var e;e="undefined"!=typeof window?window:"undefined"!=typeof global?global:"undefined"!=typeof self?self:this,e.Clipboard=t()}}(function(){var t,e,n;return function t(e,n,o){function i(a,c){if(!n[a]){if(!e[a]){var s="function"==typeof require&&require;if(!c&&s)return s(a,!0);if(r)return r(a,!0);var l=new Error("Cannot find module '"+a+"'");throw l.code="MODULE_NOT_FOUND",l}var u=n[a]={exports:{}};e[a][0].call(u.exports,function(t){var n=e[a][1][t];return i(n?n:t)},u,u.exports,t,e,n,o)}return n[a].exports}for(var r="function"==typeof require&&require,a=0;a<o.length;a++)i(o[a]);return i}({1:[function(t,e,n){var o=t("matches-selector");e.exports=function(t,e,n){for(var i=n?t:t.parentNode;i&&i!==document;){if(o(i,e))return i;i=i.parentNode}}},{"matches-selector":5}],2:[function(t,e,n){function o(t,e,n,o,r){var a=i.apply(this,arguments);return t.addEventListener(n,a,r),{destroy:function(){t.removeEventListener(n,a,r)}}}function i(t,e,n,o){return function(n){n.delegateTarget=r(n.target,e,!0),n.delegateTarget&&o.call(t,n)}}var r=t("closest");e.exports=o},{closest:1}],3:[function(t,e,n){n.node=function(t){return void 0!==t&&t instanceof HTMLElement&&1===t.nodeType},n.nodeList=function(t){var e=Object.prototype.toString.call(t);return void 0!==t&&("[object NodeList]"===e||"[object HTMLCollection]"===e)&&"length"in t&&(0===t.length||n.node(t[0]))},n.string=function(t){return"string"==typeof t||t instanceof String},n.fn=function(t){var e=Object.prototype.toString.call(t);return"[object Function]"===e}},{}],4:[function(t,e,n){function o(t,e,n){if(!t&&!e&&!n)throw new Error("Missing required arguments");if(!c.string(e))throw new TypeError("Second argument must be a String");if(!c.fn(n))throw new TypeError("Third argument must be a Function");if(c.node(t))return i(t,e,n);if(c.nodeList(t))return r(t,e,n);if(c.string(t))return a(t,e,n);throw new TypeError("First argument must be a String, HTMLElement, HTMLCollection, or NodeList")}function i(t,e,n){return t.addEventListener(e,n),{destroy:function(){t.removeEventListener(e,n)}}}function r(t,e,n){return Array.prototype.forEach.call(t,function(t){t.addEventListener(e,n)}),{destroy:function(){Array.prototype.forEach.call(t,function(t){t.removeEventListener(e,n)})}}}function a(t,e,n){return s(document.body,t,e,n)}var c=t("./is"),s=t("delegate");e.exports=o},{"./is":3,delegate:2}],5:[function(t,e,n){function o(t,e){if(r)return r.call(t,e);for(var n=t.parentNode.querySelectorAll(e),o=0;o<n.length;++o)if(n[o]==t)return!0;return!1}var i=Element.prototype,r=i.matchesSelector||i.webkitMatchesSelector||i.mozMatchesSelector||i.msMatchesSelector||i.oMatchesSelector;e.exports=o},{}],6:[function(t,e,n){function o(t){var e;if("INPUT"===t.nodeName||"TEXTAREA"===t.nodeName)t.focus(),t.setSelectionRange(0,t.value.length),e=t.value;else{t.hasAttribute("contenteditable")&&t.focus();var n=window.getSelection(),o=document.createRange();o.selectNodeContents(t),n.removeAllRanges(),n.addRange(o),e=n.toString()}return e}e.exports=o},{}],7:[function(t,e,n){function o(){}o.prototype={on:function(t,e,n){var o=this.e||(this.e={});return(o[t]||(o[t]=[])).push({fn:e,ctx:n}),this},once:function(t,e,n){function o(){i.off(t,o),e.apply(n,arguments)}var i=this;return o._=e,this.on(t,o,n)},emit:function(t){var e=[].slice.call(arguments,1),n=((this.e||(this.e={}))[t]||[]).slice(),o=0,i=n.length;for(o;i>o;o++)n[o].fn.apply(n[o].ctx,e);return this},off:function(t,e){var n=this.e||(this.e={}),o=n[t],i=[];if(o&&e)for(var r=0,a=o.length;a>r;r++)o[r].fn!==e&&o[r].fn._!==e&&i.push(o[r]);return i.length?n[t]=i:delete n[t],this}},e.exports=o},{}],8:[function(e,n,o){!function(i,r){if("function"==typeof t&&t.amd)t(["module","select"],r);else if("undefined"!=typeof o)r(n,e("select"));else{var a={exports:{}};r(a,i.select),i.clipboardAction=a.exports}}(this,function(t,e){"use strict";function n(t){return t&&t.__esModule?t:{"default":t}}function o(t,e){if(!(t instanceof e))throw new TypeError("Cannot call a class as a function")}var i=n(e),r="function"==typeof Symbol&&"symbol"==typeof Symbol.iterator?function(t){return typeof t}:function(t){return t&&"function"==typeof Symbol&&t.constructor===Symbol?"symbol":typeof t},a=function(){function t(t,e){for(var n=0;n<e.length;n++){var o=e[n];o.enumerable=o.enumerable||!1,o.configurable=!0,"value"in o&&(o.writable=!0),Object.defineProperty(t,o.key,o)}}return function(e,n,o){return n&&t(e.prototype,n),o&&t(e,o),e}}(),c=function(){function t(e){o(this,t),this.resolveOptions(e),this.initSelection()}return t.prototype.resolveOptions=function t(){var e=arguments.length<=0||void 0===arguments[0]?{}:arguments[0];this.action=e.action,this.emitter=e.emitter,this.target=e.target,this.text=e.text,this.trigger=e.trigger,this.selectedText=""},t.prototype.initSelection=function t(){this.text?this.selectFake():this.target&&this.selectTarget()},t.prototype.selectFake=function t(){var e=this,n="rtl"==document.documentElement.getAttribute("dir");this.removeFake(),this.fakeHandlerCallback=function(){return e.removeFake()},this.fakeHandler=document.body.addEventListener("click",this.fakeHandlerCallback)||!0,this.fakeElem=document.createElement("textarea"),this.fakeElem.style.fontSize="12pt",this.fakeElem.style.border="0",this.fakeElem.style.padding="0",this.fakeElem.style.margin="0",this.fakeElem.style.position="absolute",this.fakeElem.style[n?"right":"left"]="-9999px",this.fakeElem.style.top=(window.pageYOffset||document.documentElement.scrollTop)+"px",this.fakeElem.setAttribute("readonly",""),this.fakeElem.value=this.text,document.body.appendChild(this.fakeElem),this.selectedText=(0,i.default)(this.fakeElem),this.copyText()},t.prototype.removeFake=function t(){this.fakeHandler&&(document.body.removeEventListener("click",this.fakeHandlerCallback),this.fakeHandler=null,this.fakeHandlerCallback=null),this.fakeElem&&(document.body.removeChild(this.fakeElem),this.fakeElem=null)},t.prototype.selectTarget=function t(){this.selectedText=(0,i.default)(this.target),this.copyText()},t.prototype.copyText=function t(){var e=void 0;try{e=document.execCommand(this.action)}catch(n){e=!1}this.handleResult(e)},t.prototype.handleResult=function t(e){e?this.emitter.emit("success",{action:this.action,text:this.selectedText,trigger:this.trigger,clearSelection:this.clearSelection.bind(this)}):this.emitter.emit("error",{action:this.action,trigger:this.trigger,clearSelection:this.clearSelection.bind(this)})},t.prototype.clearSelection=function t(){this.target&&this.target.blur(),window.getSelection().removeAllRanges()},t.prototype.destroy=function t(){this.removeFake()},a(t,[{key:"action",set:function t(){var e=arguments.length<=0||void 0===arguments[0]?"copy":arguments[0];if(this._action=e,"copy"!==this._action&&"cut"!==this._action)throw new Error('Invalid "action" value, use either "copy" or "cut"')},get:function t(){return this._action}},{key:"target",set:function t(e){if(void 0!==e){if(!e||"object"!==("undefined"==typeof e?"undefined":r(e))||1!==e.nodeType)throw new Error('Invalid "target" value, use a valid Element');if("copy"===this.action&&e.hasAttribute("disabled"))throw new Error('Invalid "target" attribute. Please use "readonly" instead of "disabled" attribute');if("cut"===this.action&&(e.hasAttribute("readonly")||e.hasAttribute("disabled")))throw new Error('Invalid "target" attribute. You can\'t cut text from elements with "readonly" or "disabled" attributes');this._target=e}},get:function t(){return this._target}}]),t}();t.exports=c})},{select:6}],9:[function(e,n,o){!function(i,r){if("function"==typeof t&&t.amd)t(["module","./clipboard-action","tiny-emitter","good-listener"],r);else if("undefined"!=typeof o)r(n,e("./clipboard-action"),e("tiny-emitter"),e("good-listener"));else{var a={exports:{}};r(a,i.clipboardAction,i.tinyEmitter,i.goodListener),i.clipboard=a.exports}}(this,function(t,e,n,o){"use strict";function i(t){return t&&t.__esModule?t:{"default":t}}function r(t,e){if(!(t instanceof e))throw new TypeError("Cannot call a class as a function")}function a(t,e){if(!t)throw new ReferenceError("this hasn't been initialised - super() hasn't been called");return!e||"object"!=typeof e&&"function"!=typeof e?t:e}function c(t,e){if("function"!=typeof e&&null!==e)throw new TypeError("Super expression must either be null or a function, not "+typeof e);t.prototype=Object.create(e&&e.prototype,{constructor:{value:t,enumerable:!1,writable:!0,configurable:!0}}),e&&(Object.setPrototypeOf?Object.setPrototypeOf(t,e):t.__proto__=e)}function s(t,e){var n="data-clipboard-"+t;if(e.hasAttribute(n))return e.getAttribute(n)}var l=i(e),u=i(n),f=i(o),d=function(t){function e(n,o){r(this,e);var i=a(this,t.call(this));return i.resolveOptions(o),i.listenClick(n),i}return c(e,t),e.prototype.resolveOptions=function t(){var e=arguments.length<=0||void 0===arguments[0]?{}:arguments[0];this.action="function"==typeof e.action?e.action:this.defaultAction,this.target="function"==typeof e.target?e.target:this.defaultTarget,this.text="function"==typeof e.text?e.text:this.defaultText},e.prototype.listenClick=function t(e){var n=this;this.listener=(0,f.default)(e,"click",function(t){return n.onClick(t)})},e.prototype.onClick=function t(e){var n=e.delegateTarget||e.currentTarget;this.clipboardAction&&(this.clipboardAction=null),this.clipboardAction=new l.default({action:this.action(n),target:this.target(n),text:this.text(n),trigger:n,emitter:this})},e.prototype.defaultAction=function t(e){return s("action",e)},e.prototype.defaultTarget=function t(e){var n=s("target",e);return n?document.querySelector(n):void 0},e.prototype.defaultText=function t(e){return s("text",e)},e.prototype.destroy=function t(){this.listener.destroy(),this.clipboardAction&&(this.clipboardAction.destroy(),this.clipboardAction=null)},e}(u.default);t.exports=d})},{"./clipboard-action":8,"good-listener":4,"tiny-emitter":7}]},{},[9])(9)});
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
		this.ctrlContinue = createElement('button', { cName: 'button button_content_continue', inner: 'Continue', appendTo: this.ctrls });
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
			pin = document.getElementById('pin').value,
			resField = document.getElementById('result__info');

		var res = resField.parentNode;

		API.send(exp, message, pin, function(data) {
			var link = location.protocol + '//' + location.host + '/show/' + data.key;

			res.classList.add('result_loaded');

			resField.value = link;
			resField.focus();
			resField.select();
		});
	}

	function getInfo() {
		var pin = document.getElementById('pin').value,
			key = location.pathname.replace('/show/', ''),
			resField = document.getElementById('result__info'),
			resTip = document.getElementById('result__tip');

		var res = resTip.parentNode;

		API.get(key, pin, function(data) {
			res.classList.add('result_loaded');

			resTip.textContent = 'Here is your info:';
			resField.value = data.message;
			resField.focus();
			resField.select();
		}, function(json) {
			res.classList.add('result_error');

			resTip.textContent = json.error.charAt(0).toUpperCase() + json.error.slice(1);
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