block('header')(
	tag()('header'),
	content()(function() {
		return [
			{
				elem: 'content',
				tag: 'h1',
				content: this.ctx.content
			},
			{
				block: 'button',
				mods: { shown: true },
				mix: { block: 'header', elem: 'info' },
				attrs: {
					id: 'info'
				},
				content: 'i'
			}
		];
	})
);