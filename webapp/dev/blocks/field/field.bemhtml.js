block('field')(
	tag()('li'),
	content()(function() {
		var ctx = this.ctx;

		return [
			{
				elem: 'title',
				id: ctx.id,
				content: ctx.title
			},
			ctx.content
		]
	})
);

block('field').elem('title')(
	tag()('label'),
	mix()({ block: 'animation', elem: 'upper' }),
	attrs()(function() {
		return {
			for: this.ctx.id
		};
	})
);

block('field').elem('desc')(
	mix()([
		{ block: 'animation', elem: 'upper' },
		{ block: 'description'}
	])
);