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
					content: 'SafeSecret.Info'
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
									title: 'Enter the PIN to get protected info',
									content: {
										block: 'input',
										mods: { type: 'pin' },
										id: 'pin',
										placeholder: 'PIN here'
									}
								}
							]
						},
						{
							block: 'result',
							content: [
								{
									elem: 'tip',
									attrs: {
										id: 'result__tip'
									}
								},
								{
									block: 'textarea',
									mods: { autoselect: true },
									mix: { block: 'result', elem: 'info' },
									attrs: {
										id: 'result__info'
									},
									placeholder: 'Loading..'
								},
								{
									block: 'description',
									content: 'This is the last time this information will be available for any access.'
								}
							]
						}
					]
				}
			]
		}
	]
})