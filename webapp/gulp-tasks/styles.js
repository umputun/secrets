module.exports = function(gulp, $, path, options) {
	'use strict';

	options = options || {
		errorHandler: function() {
			return $.plumber();
		}
	};

	gulp.task('styles', function() {
		return gulp.src(path.inputStyles)
			.pipe(options.errorHandler('Styles'))

			.pipe($.order())
			.pipe($.concat('main.scss'))
			.pipe($.sass())
			
			.pipe($.postcss([
				$.autoprefixer({ browsers: '> 0.3%, not ie < 10' })
			]))
			
			.pipe($.combineMq({
				beautify: false
			}))
			.pipe($.cssnano())
			.pipe(gulp.dest(path.outputStyles))
	});	
};