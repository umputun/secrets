module.exports = function(gulp, $, path, options) {
	'use strict';

	options = options || {
		errorHandler: function() {
			return $.plumber();
		}
	};

	gulp.task('files', function() {
		return gulp.src(path.inputFiles, { since: gulp.lastRun('files') })
			.pipe(options.errorHandler('Files'))
			
			.pipe($.imagemin({
				progressive: true,
				use: [$.pngquant()],
				interlaced: true
			}))
			.pipe(gulp.dest(path.outputFiles))
	});
};