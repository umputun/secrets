block('header')(
	tag()('header'),
	content()(function() {
		return {
			elem: 'content',
			tag: 'h1',
			content: this.ctx.content
		};
	})
);