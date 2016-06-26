'use strict';

const gulp 			= require('gulp');
const bs 			= require('browser-sync').create();
const pathResolver 	= require('path');

const $ = require('gulp-load-plugins')({
	replaceString: /^gulp(-|\.)|postcss-/,
	pattern: ['*'],
	rename: {
		'gulp-if': 'ifelse',
		'imagemin-pngquant': 'pngquant'
	}
});

const path = {
	tasks: './gulp-tasks/',
	releases: 'releases',
	output: 'public',
	tmp: '.tmp',

	inputBEMJSON: ['dev/bemjson/**/*.bemjson.js'],

	inputBEMHTML: ['dev/blocks/**/*.bemhtml.js'],
	BEMHTML: '_template.bemhtml.js',

	inputStyles: ['dev/blocks/**/*.{scss,css}'],
	outputStyles: 'public/css',
	
	inputJS: ['dev/blocks/**/*.js', '!dev/blocks/**/*.bemhtml.js'],
	outputJS: 'public/js',
	
	inputImages: ['dev/blocks/**/*.{jpg,png,gif,svg}'],
	outputImages: 'public/images',
	
	inputFiles: 'dev/files/**/*',
	outputFiles: 'public/files',
	
	inputFonts: 'dev/blocks/font/**/*.{eot,svg,ttf,woff,woff2}',
	outputFonts: 'public/fonts'
};

const taskList = require('fs').readdirSync(path.tasks);

const options = {
	isProduction: process.env.NODE_ENV == 'production',
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

gulp.task('build', gulp.parallel('html', 'styles', 'js', 'images', 'files', 'fonts'));

gulp.task('default', gulp.series('build', gulp.parallel('watch', 'server')));

gulp.task('clean', function() {
	return $.del([path.output, path.tmp])
});

gulp.task('zip', function() {
	const name = require('./package.json').version;

	return gulp.src(path.output + '/**/*.*')
		.pipe(options.errorHandler('zip'))

		.pipe($.zip(name + '.zip'))
		.pipe(gulp.dest(path.releases));
});

gulp.task('release', gulp.series('clean', 'build', 'zip'));
