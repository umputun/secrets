'use strict';

const gulp = require('gulp');

const path = {
	tasks: './gulp-tasks/',
	output: 'public',
	tmp: '.tmp',

	inputBEMJSON: ['dev/bemjson/**/*.bemjson.js'],

	inputBEMHTML: ['dev/blocks/**/*.bemhtml.js'],
	BEMHTML: '_template.bemhtml.js',

	inputStyles: ['dev/blocks/**/*.{scss,css}'],
	outputStyles: 'public/css',
	
	inputJS: ['dev/blocks/**/*.js', '!dev/blocks/**/*.bemhtml.js'],
	outputJS: 'public/js',

	inputFiles: ['dev/blocks/favicon/**/*']
};

const taskList = require('fs').readdirSync(path.tasks);

taskList.forEach(taskFile => require(path.tasks + taskFile)(path));

if (process.env.NODE_ENV == 'dev') {
	gulp.task('default', gulp.series('dev'));
} else {
	gulp.task('default', gulp.series('build'));
}
