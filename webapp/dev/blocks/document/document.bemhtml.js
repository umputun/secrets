block('document').replace()(function() {
	var ctx = this.ctx;

	return [
	    '<!DOCTYPE html>',
	    {
	        tag: 'html',
	        block: 'document no-js',
	        attrs: {
	            lang: 'ru'
	        },
	        content: [
	        	{
		            tag: 'head',
		            content: [
		                {
		                    tag: 'meta',
		                    attrs: {
		                        charset: 'utf-8'
		                    }
		                },
		                {
		                    tag: 'meta',
		                    attrs: {
		                        'http-equiv': 'X-UA-Compatible',
		                        content: 'IE=edge'
		                    }
		                },
		                {
		                    tag: 'meta',
		                    attrs: {
		                        name: 'viewport',
		                        content: 'width=device-width, initial-scale=1'
		                    }
		                },
		                {
		                    tag: 'title',
		                    content: ctx.title
		                },
		                {
		                	tag: 'link',
		                	attrs: {
		                		rel: 'apple-touch-icon',
		                		sizes: '180x180',
		                		href: '/apple-touch-icon.png'
		                	}
		                },
		                {
		                	tag: 'link',
		                	attrs: {
		                		rel: 'icon',
		                		type: 'image/png',
		                		href: '/favicon-32x32.png',
		                		sizes: '32x32'
		                	}
		                },
		                {
		                	tag: 'link',
		                	attrs: {
		                		rel: 'icon',
		                		type: 'image/png',
		                		href: '/favicon-16x16.png',
		                		sizes: '16x16'
		                	}
		                },
		                {
		                	tag: 'link',
		                	attrs: {
		                		rel: 'manifest',
		                		href: '/manifest.json' 
		                	}
		                },
		                {
		                	tag: 'link',
		                	attrs: {
		                		rel: 'mask-icon',
		                		href: '/safari-pinned-tab.svg',
		                		color: '#5bbad5'
		                	}
		                },
		                {
		                	tag: 'meta',
		                	attrs: {
		                		name: 'apple-mobile-web-app-title',
		                		content: 'SafeSecret'
		                	}
		                },
		                {
		                	tag: 'meta',
		                	attrs: {
		                		name: 'application-name',
		                		content: 'SafeSecret'
		                	}
		                },
		                {
		                	tag: 'meta',
		                	attrs: {
		                		name: 'theme-color',
		                		content: '#ffffff'
		                	}
		                },
		                ctx.styles.map(function(link) {
		                	return {
		                		tag: 'link',
		                		attrs: {
		                			rel: 'stylesheet',
		                			href: link
		                		}
		                	};
		                }),
		                ctx.scripts.map(function(script) {
		                	return {
		                		tag: 'script',
		                		attrs: {
		                			src: script
		                		}
		                	};
		                })
		            ]
		        },
		        {
		        	tag: 'body',
		        	cls: 'document__page',
		        	content: ctx.content
		        }
			]
	    }
	];
});