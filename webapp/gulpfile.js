'use strict';

var gulp = require('gulp');

var $ = require('gulp-load-plugins')({
	replaceString: /^gulp(-|\.)|postcss-/,
	pattern: ['*']
});

var path = {
	tasks: './gulp-tasks/',
	output: 'public',
	tmp: '.tmp',

	inputBEMJSON: ['dev/bemjson/**/*.bemjson.js'],

	inputBEMHTML: ['dev/blocks/**/*.bemhtml.js'],
	BEMHTML: '_template.bemhtml.js',

	inputStyles: ['dev/blocks/**/*.{scss,css}'],
	outputStyles: 'public/css',
	
	inputJS: ['dev/blocks/**/*.js', '!dev/blocks/**/*.bemhtml.js'],
	outputJS: 'public/js'
};

var taskList = require('fs').readdirSync(path.tasks);

var options = {
	isOnline: process.env.SERVER_MODE == 'online',
	errorHandler: function(title) {
		return $.plumber({
			errorHandler: $.notify.onError(function(err) {
				return {
					title: title  + ' (' + err.plugin + ')',
					message: err.message
				};
			})
		});
	}
};

taskList.forEach(function (taskFile) {
	require(path.tasks + taskFile)(gulp, $, path, options);
});

gulp.task('default', gulp.series('dev'));
