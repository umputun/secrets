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

block('textarea').mod('autoselect', true)(
	attrs()(function() {
		var ctx = this.ctx;

		return {
			id: ctx.id,
			onclick: 'this.focus(); this.select()',
			readonly: true
		};
	})
);