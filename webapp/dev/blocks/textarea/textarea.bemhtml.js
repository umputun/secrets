block('textarea')(
	tag()('textarea'),
	mix()({ block: 'animation', elem: 'lower' }),
	attrs()(function() {
		var ctx = this.ctx;

		return {
			id: ctx.id,
			placeholder: ctx.placeholder,
			required: true
		};
	})
);