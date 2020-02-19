module.exports = function(gulp, $, path) {
	'use strict';

	gulp.task('build:html:bemhtml', function() {
		return gulp.src(path.inputBEMHTML)
			.pipe($.concat(path.BEMHTML))
			.pipe(gulp.dest(path.tmp));
	});

	gulp.task('build:html:bemjson', function() {
		return gulp.src(path.inputBEMJSON)
			.pipe($.bemjson2html({ template: path.tmp + '/' + path.BEMHTML }))
			.pipe($.rename(function(path) {
				var dirname = path.basename.replace(/.bemjson/, '');

				if (dirname != 'index') {
					path.dirname += '/' + dirname;
				}

				path.basename = 'index';
				path.extname = '.html';
			}))
			.pipe(gulp.dest(path.output));
	});

	gulp.task('build:html', gulp.series('build:html:bemhtml', 'build:html:bemjson'));

	gulp.task('build:styles', function() {
		return gulp.src(path.inputStyles)
			.pipe($.order())
			.pipe($.concat('main.scss'))
			.pipe($.sass())
			
			.pipe($.postcss([
				$.autoprefixer({ browsers: '> 0.3%, not ie < 10' })
			]))
			
			.pipe($.combineMq({
				beautify: false
			}))
			.pipe($.csso())
			.pipe(gulp.dest(path.outputStyles))
	});

	gulp.task('build:js', function() {
		return gulp.src(path.inputJS)
			.pipe($.order())
			.pipe($.concat('main.js'))
			.pipe($.uglify())
			.pipe(gulp.dest(path.outputJS))
	});

	gulp.task('build:files', function() {
		return gulp.src(path.inputFiles)
			.pipe(gulp.dest(path.output))
	});

	gulp.task('build', gulp.parallel('build:html', 'build:styles', 'build:js', 'build:files'));
};