const gulp = require('gulp');
const Fiber = require('fibers');
const concat = require('gulp-concat');
const order = require('gulp-order');
const sass = require('gulp-sass');
const bemjson2html = require('gulp-bemjson2html');
const rename = require('gulp-rename');
const combineMq = require('gulp-combine-mq');
const postcss = require('gulp-postcss');
const autoprefixer = require('autoprefixer');
const csso = require('gulp-csso');
const uglify = require('gulp-uglify');

sass.compiler = require('sass');

module.exports = path => {
	gulp.task('build:html:bemhtml', () => {
		return gulp.src(path.inputBEMHTML)
			.pipe(concat(path.BEMHTML))
			.pipe(gulp.dest(path.tmp));
	});

	gulp.task('build:html:bemjson', () => {
		return gulp.src(path.inputBEMJSON)
			.pipe(bemjson2html({ template: path.tmp + '/' + path.BEMHTML }))
			.pipe(rename(path => {
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

	gulp.task('build:styles', () => {
		return gulp.src(path.inputStyles)
			.pipe(order())
			.pipe(concat('main.scss'))
			.pipe(sass({ fiber: Fiber }))
			
			.pipe(postcss([
				autoprefixer({ browsers: '> 0.3%, not ie < 10' })
			]))
			
			.pipe(combineMq({
				beautify: false
			}))
			.pipe(csso())
			.pipe(gulp.dest(path.outputStyles))
	});

	gulp.task('build:js', () => {
		return gulp.src(path.inputJS)
			.pipe(order())
			.pipe(concat('main.js'))
			.pipe(uglify())
			.pipe(gulp.dest(path.outputJS))
	});

	gulp.task('build:files', () => {
		return gulp.src(path.inputFiles)
			.pipe(gulp.dest(path.output))
	});

	gulp.task('build', gulp.parallel('build:html', 'build:styles', 'build:js', 'build:files'));
};
