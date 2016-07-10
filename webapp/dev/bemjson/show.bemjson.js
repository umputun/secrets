({
	block: 'document',
	title: 'SafeSecret.Info',
	styles: ['../css/main.css'],
	scripts: ['../js/main.js'],
	content: [
		{
			block: 'fs-form',
			attrs: {
				id: 'fs-form'
			},
			content: [
				{
					block: 'header',
					content: {
						block: 'link',
						url: '/',
						content: 'SafeSecret.Info'
					}
				},
				{
					elem: 'content',
					elemMods: { full: true },
					tag: 'form',
					attrs: {
						autocomplete: 'off'
					},
					content: [
						{
							block: 'fields',
							content: [
								{
									block: 'field',
									id: 'pin',
									title: 'Enter the secret PIN to see content',
									content: {
										block: 'input',
										mods: { type: 'pin' },
										id: 'pin',
										placeholder: '5-diget PIN'
									}
								}
							]
						},
						{
							block: 'result',
							content: [
								{
									elem: 'throbber',
									content: 'Loading..'
								},
								{
									elem: 'tip',
									attrs: {
										id: 'result__tip'
									}
								},
								{
									block: 'link',
									mix: { block: 'result', elem: 'again' },
									url: 'javascript:location.reload()',
									content: 'Try again'
								},
								{
									block: 'link',
									mix: { block: 'result', elem: 'to-main' },
									url: '/',
									content: 'Back to main page'
								},
								{
									block: 'textarea',
									mods: { result: true },
									mix: { block: 'result', elem: 'info' },
									attrs: {
										id: 'result__info'
									}
								},
								{
									block: 'description',
									mix: { block: 'result', elem: 'desc' },
									content: 'This information will self-destruct after you close this page.'
								},
								{
									block: 'button',
									mods: { content: 'copy', shown: true },
									mix: { block: 'result', elem: 'button' },
									attrs: {
										'data-clipboard-target': '#result__info'
									},
									content: 'Copy'
								},
							]
						}
					]
				}
			]
		}
	]
})
