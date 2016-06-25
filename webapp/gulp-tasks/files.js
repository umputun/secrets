module.exports = function(gulp, $, path, options) {
	'use strict';

	options = options || {
		isProduction: false,
		errorHandler: function() {
			return $.plumber();
		}
	};

	gulp.task('files', function() {
		const production = $.imagemin({
			progressive: true,
			use: [$.pngquant()],
			interlaced: true
		});

		return gulp.src(path.inputFiles, { since: gulp.lastRun('files') })
			.pipe(options.errorHandler('Files'))
			
			.pipe($.ifelse(options.isProduction, production))
			.pipe(gulp.dest(path.outputFiles))
	});
};