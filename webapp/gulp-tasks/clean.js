module.exports = function(gulp, $, path) {
	'use strict';

	gulp.task('clean', function() {
		return $.del([path.output, path.tmp])
	});
};