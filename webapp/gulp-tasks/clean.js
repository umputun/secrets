module.exports = function(gulp, $, path, options) {
	'use strict';

	gulp.task('clean', function() {
		return $.del([path.output, path.tmp])
	});
};