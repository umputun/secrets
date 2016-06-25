module.exports = function(gulp, $, path, options) {
	'use strict';

	options = options || {
		isProduction: false,
		errorHandler: function() {
			return $.plumber();
		}
	};

	gulp.task('images', function() {
		const production = $.imagemin({
			progressive: true,
			use: [$.pngquant()],
			interlaced: true
		});

		return gulp.src(path.inputImages, { since: gulp.lastRun('images') })
			.pipe(options.errorHandler('Images'))

			.pipe($.rename({ dirname: '' }))
			.pipe($.ifelse(options.isProduction, production))
			.pipe(gulp.dest(path.outputImages))
	});
};