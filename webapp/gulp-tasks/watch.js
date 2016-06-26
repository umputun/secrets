module.exports = function(gulp, $, path) {
	'use strict';

	gulp.task('watch', function() {
		gulp.watch([path.inputBEMHTML, path.inputBEMJSON], gulp.series('html'));
		gulp.watch(path.inputStyles, gulp.series('styles'));
		gulp.watch(path.inputJS, gulp.series('js'));
		gulp.watch(path.inputImages, gulp.series('images'));
		gulp.watch(path.inputFiles, gulp.series('files'));
		gulp.watch(path.inputFonts, gulp.series('fonts'));
	});
};