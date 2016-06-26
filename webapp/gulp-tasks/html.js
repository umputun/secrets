module.exports = function(gulp, $, path, options) {
	'use strict';

	options = options || {
		errorHandler: function() {
			return $.plumber();
		}
	};

	gulp.task('html:bemhtml', function() {
		return gulp.src(path.inputBEMHTML)
			.pipe(options.errorHandler('bemhtml'))

			.pipe($.concat(path.BEMHTML))
			.pipe(gulp.dest(path.tmp));
	});

	gulp.task('html:bemjson', function() {
		return gulp.src(path.inputBEMJSON)
			.pipe(options.errorHandler('bemjson'))

			.pipe($.bemjson2html({ template: path.tmp + '/' + path.BEMHTML }))
			.pipe($.rename(function(path) {
				var dirname = path.basename.replace(/.bemjson/, '');

				if (dirname != 'index') {
					path.dirname += '/' + dirname;
				}

				path.basename = 'index';
				path.extname = '.html';
			}))

			.pipe($.cached('bemjson'))

			.pipe(gulp.dest(path.output));
	});

	gulp.task('html', gulp.series('html:bemhtml', 'html:bemjson'));
};