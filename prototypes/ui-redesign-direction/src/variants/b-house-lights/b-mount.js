// PROTOTYPE. House Lights: registers the direction with the stage.
(function () {
  const SS = window.SS, VB = window.VB;
  SS.registerVariant({
    key: 'B',
    name: 'House Lights',
    mount(ctx) {
      if (ctx.portal) return VB.portal(ctx);
      if (ctx.phone) return VB.phoneShell(ctx);
      return VB.wide(ctx);
    },
  });
})();
