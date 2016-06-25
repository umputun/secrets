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