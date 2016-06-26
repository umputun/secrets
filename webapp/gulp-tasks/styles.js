module.exports = function(gulp, $, path, options) {
	'use strict';

	options = options || {
		errorHandler: function() {
			return $.plumber();
		}
	};

	gulp.task('styles', function() {
		function urlMapper(url, type) {
			if (url[0] == '.'
				|| url[0] == '/'
				|| url.indexOf('http') == 0) {
				return url;
			}

			if (type == 'src') {
				return '../fonts/' + url;
			} else {
				return '../images/' + url;
			}
		}

		return gulp.src(path.inputStyles)
			.pipe(options.errorHandler('Styles'))

			.pipe($.order())
			.pipe($.concat('main.scss'))
			.pipe($.sass())
			
			.pipe($.postcss([
				$.autoprefixer({ browsers: '> 0.3%, not ie < 10' }),
				$.urlMapper(urlMapper)
			]))
			
			.pipe($.combineMq({
				beautify: false
			}))
			.pipe($.cssnano())
			.pipe(gulp.dest(path.outputStyles))
	});	
};