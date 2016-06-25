module.exports = function(gulp, $, path, options) {
	'use strict';

	options = options || {
		isOnline: false,
		bs: require('browser-sync').create()
	};

	const bs = options.bs;

	gulp.task('server', function() {
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
			online: options.isOnline
		});

		bs.watch(path.output + '/**/*.*').on('change', bs.reload);
	});
};