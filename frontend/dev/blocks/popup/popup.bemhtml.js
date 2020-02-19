block('popup')(
	content()(function() {
		var ctx = this.ctx;

		return [
			{
				elem: 'content',
				content: ctx.content
			},
			{
				block: 'button',
				mods: { content: 'welcome', shown: true },
				mix: { block: 'popup', elem: 'button' },
				content: 'I\'ve got it'
			}
		]
	})
);