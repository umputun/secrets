module.exports = function(gulp, $, path) {
	'use strict';

	if (process.env.NODE_ENV == 'dev') {
		var bs = $.browserSync.create();
		var isOnline = false;
		var errorHandler = function() {
			return $.plumber({
				errorHandler: $.notify.onError(function(err) {
					return {
						title: title  + ' (' + err.plugin + ')',
						message: err.message
					};
				})
			});
		};

		gulp.task('dev:html:bemhtml', function() {
			return gulp.src(path.inputBEMHTML)
				.pipe(errorHandler('BEMHTML'))
				
				.pipe($.concat(path.BEMHTML))
				
				.pipe(gulp.dest(path.tmp));
		});

		gulp.task('dev:html:bemjson', function() {
			return gulp.src(path.inputBEMJSON)
				.pipe(errorHandler('BEMJSON'))
				
				.pipe($.bemjson2html({ template: path.tmp + '/' + path.BEMHTML }))
				.pipe($.rename(function(path) {
					var dirname = path.basename.replace(/.bemjson/, '');

					if (dirname != 'index') {
						path.dirname += '/' + dirname;
					}

					path.basename = 'index';
					path.extname = '.html';
				}))

				.pipe($.cached('BEMJSON'))
				
				.pipe(gulp.dest(path.output));
		});

		gulp.task('dev:html', gulp.series('dev:html:bemhtml', 'dev:html:bemjson'));

		gulp.task('dev:styles', function() {
			return gulp.src(path.inputStyles)
				.pipe(errorHandler('Styles'))

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

		gulp.task('dev:js', function() {
			return gulp.src(path.inputJS)
				.pipe(errorHandler('JS'))

				.pipe($.order())
				.pipe($.concat('main.js'))
				.pipe($.uglify())
				
				.pipe(gulp.dest(path.outputJS))
		});

		gulp.task('dev:files', function() {
			return gulp.src(path.inputFiles)
				.pipe(errorHandler('Files'))

				.pipe(gulp.dest(path.output))
		});

		gulp.task('dev:build', gulp.parallel('dev:html', 'dev:styles', 'dev:js', 'dev:files'));

		gulp.task('dev:watch', function() {
			gulp.watch([path.inputBEMHTML, path.inputBEMJSON], gulp.series('dev:html'));
			gulp.watch(path.inputStyles, gulp.series('dev:styles'));
			gulp.watch(path.inputJS, gulp.series('dev:js'));
			gulp.watch(path.inputFiles, gulp.series('dev:files'));
		});

		gulp.task('dev:server', function() {
			bs.init({
				server: {
					baseDir: path.output,
					middleware: [
						$.connectModrewrite([
							'^/show/(.*)$ /show/?$1 [L]'
						])
					]
				},
				open: false,
				browser: 'browser',
				reloadOnRestart: true,
				online: isOnline,
				https: true
			});

			bs.watch(path.output + '/**/*.*').on('change', bs.reload);
		});

		gulp.task('dev', gulp.series('dev:build', gulp.parallel('dev:watch', 'dev:server')));

		gulp.task('dev:online', function(cb) {
			isOnline = true;
			cb();
		});

		gulp.task('devOnline', gulp.series('dev:online', 'dev'));
	}
};