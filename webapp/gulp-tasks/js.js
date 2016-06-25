module.exports = function(gulp, $, path, options) {
	'use strict';

	options = options || {
		isProduction: false,
		errorHandler: function() {
			return $.plumber();
		}
	};

	gulp.task('js', function() {
		return gulp.src(path.inputJS)
			.pipe(options.errorHandler('JS'))
			
			.pipe($.order())
			.pipe($.concat('main.js'))
			.pipe($.ifelse(options.isProduction, $.uglify()))
			.pipe(gulp.dest(path.outputJS))
	});
};