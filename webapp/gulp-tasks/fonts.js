module.exports = function(gulp, $, path, options) {
	'use strict';

	options = options || {
		errorHandler: function() {
			return $.plumber();
		}
	};

	gulp.task('fonts', function() {
		return gulp.src(path.inputFonts, { since: gulp.lastRun('fonts') })
			.pipe(options.errorHandler('Fonts'))

			.pipe($.rename({ dirname: '' }))
			.pipe(gulp.dest(path.outputFonts))
	});
};