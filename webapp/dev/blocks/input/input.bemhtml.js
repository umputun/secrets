block('input')(
	tag()('input'),
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

block('input').mod('type', 'number')(
	attrs()(function() {
		var attrs = applyNext();

		attrs.type = 'number';

		return attrs;
	})
);

block('input').mod('type', 'pin')(
	attrs()(function() {
		var attrs = applyNext();

		attrs.type = 'number';
		attrs.min = '10000';
		attrs.max = '99999';

		return attrs;
	})
);