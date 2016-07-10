({
	block: 'document',
	title: 'SafeSecret.Info',
	styles: ['css/main.css'],
	scripts: ['js/main.js'],
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
									id: 'text',
									title: 'What information do you want to share?',
									content: {
										block: 'textarea',
										id: 'text',
										placeholder: 'Enter here'
									}
								},
								{
									block: 'field',
									id: 'time',
									title: 'How long to keep?',
									content: [
										{
											block: 'input',
											mods: { type: 'number' },
											id: 'time',
											placeholder: '10'
										},
										{
											elem: 'desc',
											content: 'How many minutes?'
										}
									]
								},
								{
									block: 'field',
									id: 'pin',
									title: 'Enter the PIN to protect your information',
									content: [
										{
											block: 'input',
											mods: { type: 'pin' },
											id: 'pin',
											placeholder: (function() {
												var text = '';
											    var possible = '0123456789';

											    for (var i = 0; i < 5; i++)
											        text += possible.charAt(Math.floor(Math.random() * possible.length));

											    return text;
											})()
										},
										{
											elem: 'desc',
											content: 'PIN is a 5-digit password.'
										}
									]
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
									content: 'Here is your link and don\'t forget your PIN!'
								},
								{
									block: 'textarea',
									mods: { result: true, size: 'm' },
									mix: { block: 'result', elem: 'info' },
									attrs: {
										id: 'result__info'
									}
								},
								{
									block: 'description',
									mix: { block: 'result', elem: 'desc' },
									content: [
										'Use this link to access to saving information. ',
										{
											block: 'link',
											url: 'javascript:location.reload()',
											content: 'Save yet another one?'
										}
									]
								},
								{
									block: 'button',
									mods: { content: 'copy', shown: true },
									mix: { block: 'result', elem: 'button' },
									attrs: {
										'data-clipboard-target': '#result__info'
									},
									content: 'Copy'
								}
							]
						}
					]
				}
			]
		}
	]
})