(function () {
  const SS = window.SS;
  const VA = window.VA;
  SS.registerVariant({
    key: 'A',
    name: 'Slipstream',
    mount(ctx) {
      if (ctx.shell === 'wide') return VA.mountWide(ctx);
      if (ctx.shell === 'phone') return VA.mountPhone(ctx);
      return VA.mountPortal(ctx);
    },
  });
})();
