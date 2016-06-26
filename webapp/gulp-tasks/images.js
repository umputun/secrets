module.exports = function(gulp, $, path, options) {
	'use strict';

	options = options || {
		errorHandler: function() {
			return $.plumber();
		}
	};

	gulp.task('images', function() {
		return gulp.src(path.inputImages, { since: gulp.lastRun('images') })
			.pipe(options.errorHandler('Images'))

			.pipe($.rename({ dirname: '' }))
			.pipe($.imagemin({
				progressive: true,
				use: [$.pngquant()],
				interlaced: true
			}))
			.pipe(gulp.dest(path.outputImages))
	});
};