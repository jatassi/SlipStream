(function () {
  'use strict';
  const VC = window.VC;
  window.SS.registerVariant({
    key: 'C',
    name: 'Night Line',
    mount(ctx) {
      ctx.root.classList.add('c-root');
      const app = ctx.portal ? VC.portal(ctx) : ctx.phone ? VC.phone(ctx) : VC.wide(ctx);
      ctx.cleanup(() => VC.act.resumeAll());
      return { unmount() { app?.destroy?.(); } };
    },
  });
})();
