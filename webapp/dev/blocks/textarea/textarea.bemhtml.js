block('textarea')(
	tag()('textarea'),
	mix()({ block: 'animation', elem: 'lower' }),
	attrs()(function() {
		var ctx = this.ctx;

		return {
			id: ctx.id,
			placeholder: ctx.placeholder,
			required: true,
			autofocus: true
		};
	})
);

block('textarea').mod('autoselect', true)(
	attrs()(function() {
		var ctx = this.ctx;

		return {
			id: ctx.id,
			placeholder: ctx.placeholder,
			onclick: 'this.focus(); this.select()',
			readonly: true
		};
	})
);