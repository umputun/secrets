const gulp = require('gulp');
const del = require('del');

module.exports = path => {
	gulp.task('clean', () => del([path.output, path.tmp]));
};
