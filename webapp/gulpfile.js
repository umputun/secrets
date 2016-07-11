'use strict';

const gulp 			= require('gulp');
const bs 			= require('browser-sync').create();
const pathResolver 	= require('path');

const $ = require('gulp-load-plugins')({
	replaceString: /^gulp(-|\.)|postcss-/,
	pattern: ['*'],
	rename: {
		'gulp-if': 'ifelse'
	}
});

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
	outputJS: 'public/js'
};

const taskList = require('fs').readdirSync(path.tasks);

const options = {
	isOnline: process.env.SERVER_MODE == 'online',
	bs: bs,
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

gulp.task('build', gulp.parallel('html', 'styles', 'js'));

gulp.task('default', gulp.series('build', gulp.parallel('watch', 'server')));

gulp.task('clean', function() {
	return $.del([path.output, path.tmp])
});
