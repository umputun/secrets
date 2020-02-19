block('link')(
	tag()('a'),
	attrs()(function() {
		var ctx = this.ctx;

		return {
			href: ctx.url
		};
	})
);