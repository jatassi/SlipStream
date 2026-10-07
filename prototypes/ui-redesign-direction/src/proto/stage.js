// PROTOTYPE. The stage: mounts one variant in one shell, frames phone shells in a device,
// and draws the switcher. Never part of any design being evaluated.
(function () {
  const SS = window.SS;
  const VARIANTS = (SS.variants = {});
  const ORDER = [];
  SS.registerVariant = (def) => { VARIANTS[def.key] = def; ORDER.push(def.key); ORDER.sort(); };

  const SHELLS = [
    { id: 'wide', label: 'Admin', hint: 'Desk', short: 'Desk' },
    { id: 'phone', label: 'Admin', hint: 'Phone', short: 'Phone' },
    { id: 'portal', label: 'Portal', hint: 'Phone', short: 'Portal' },
    { id: 'portal-wide', label: 'Portal', hint: 'Desk', short: 'Portal web' },
  ];
  const DEVICE = { w: 393, h: 852, top: 54, bottom: 34 };
  const params = new URLSearchParams(location.search);
  const state = {
    variant: (params.get('variant') || 'A').toUpperCase(),
    shell: params.get('shell') || 'wide',
    view: params.get('view') || '',
    bar: params.get('bar') !== '0',
    still: params.get('still') === '1',
  };
  const narrow = () => Math.min(innerWidth, screen.width) < 700;
  const isPhoneShell = (s) => s === 'phone' || s === 'portal';

  let current = null;
  const stageEl = document.getElementById('stage');

  function writeURL() {
    const p = new URLSearchParams(location.search);
    p.set('variant', state.variant);
    p.set('shell', state.shell);
    state.view ? p.set('view', state.view) : p.delete('view');
    history.replaceState(null, '', `${location.pathname}?${p}`);
  }

  function enableCSS(key) {
    for (const k of ORDER) {
      const el = document.getElementById('css-' + k);
      if (el?.sheet) el.sheet.disabled = k !== key;
    }
  }

  function frameFor(shell) {
    stageEl.innerHTML = '';
    stageEl.className = 'stage';
    stageEl.removeAttribute('style');
    const safe = (top, bottom) => { stageEl.style.setProperty('--safe-top', top); stageEl.style.setProperty('--safe-bottom', bottom); };
    const host = document.createElement('div');
    host.className = 'ss-root';
    host.dataset.variant = state.variant;
    host.dataset.shell = shell;
    if (isPhoneShell(shell) && !narrow()) {
      stageEl.classList.add('stage--device');
      stageEl.innerHTML = `<div class="device"><div class="device__screen"></div><div class="device__island"></div><div class="device__status"><span class="device__time">21:42</span><span class="device__glyphs">${statusGlyphs()}</span></div><div class="device__home"></div></div>`;
      const screenEl = stageEl.querySelector('.device__screen');
      safe(DEVICE.top + 'px', DEVICE.bottom + 'px');
      screenEl.appendChild(host);
      fitDevice();
      return { host, frame: 'device', width: DEVICE.w, height: DEVICE.h };
    }
    if (!isPhoneShell(shell) && narrow()) {
      stageEl.classList.add('stage--scaled');
      safe('0px', '0px');
      const box = document.createElement('div');
      box.className = 'scaled';
      box.appendChild(host);
      stageEl.appendChild(box);
      fitScaled(box);
      return { host, frame: 'scaled', width: 1440, height: 900 };
    }
    stageEl.classList.add('stage--native');
    safe(isPhoneShell(shell) && !(narrow() && state.bar) ? 'env(safe-area-inset-top, 0px)' : '0px', isPhoneShell(shell) ? 'env(safe-area-inset-bottom, 0px)' : '0px');
    stageEl.appendChild(host);
    return { host, frame: 'native', width: innerWidth, height: innerHeight };
  }

  function statusGlyphs() {
    return `<svg width="18" height="12" viewBox="0 0 18 12"><rect x="0" y="8" width="3" height="4" rx="1"/><rect x="5" y="5.5" width="3" height="6.5" rx="1"/><rect x="10" y="3" width="3" height="9" rx="1"/><rect x="15" y="0" width="3" height="12" rx="1"/></svg>
      <svg width="16" height="12" viewBox="0 0 16 12"><path d="M8 11.5 5.6 9a3.4 3.4 0 0 1 4.8 0z"/><path d="M3.4 6.8a6.5 6.5 0 0 1 9.2 0l-1.3 1.3a4.6 4.6 0 0 0-6.6 0z" /><path d="M1.2 4.6a9.6 9.6 0 0 1 13.6 0l-1.3 1.3a7.7 7.7 0 0 0-11 0z"/></svg>
      <svg width="27" height="13" viewBox="0 0 27 13"><rect x="0.5" y="0.5" width="23" height="12" rx="3.5" fill="none" stroke="currentColor" opacity=".4"/><rect x="2" y="2" width="16" height="9" rx="2"/><path d="M25 4.5v4a2 2 0 0 0 0-4z" opacity=".45"/></svg>`;
  }

  function fitDevice() {
    const dev = stageEl.querySelector('.device');
    if (!dev) return;
    const s = Math.min(1, (innerHeight - (state.bar ? 120 : 48)) / (DEVICE.h + 28), (innerWidth - 48) / (DEVICE.w + 28));
    dev.style.transform = `scale(${s})`;
  }
  function fitScaled(box) {
    const s = innerWidth / 1440;
    box.style.transform = `scale(${s})`;
    box.parentElement.style.height = 900 * s + 'px';
  }

  async function mount() {
    if (current) {
      try { current.cleanups.forEach((fn) => fn()); current.api?.unmount?.(); } catch (e) { console.error(e); }
      current = null;
    }
    const def = VARIANTS[state.variant] || VARIANTS[ORDER[0]];
    state.variant = def.key;
    enableCSS(def.key);
    document.documentElement.dataset.variant = def.key;
    const { host, frame, width, height } = frameFor(state.shell);
    const cleanups = [];
    const ctx = {
      root: host,
      shell: state.shell,
      phone: isPhoneShell(state.shell),
      portal: state.shell.startsWith('portal'),
      frame, width, height,
      view: state.view,
      setView(v) { state.view = v || ''; ctx.view = state.view; writeURL(); },
      data: SS.data, sim: SS.sim, art: SS.art, icon: SS.icon, fmt: SS.fmt,
      on(ev, fn) { const off = SS.sim.on(ev, fn); cleanups.push(off); return off; },
      cleanup(fn) { cleanups.push(fn); },
      listen(target, ev, fn, opt) { target.addEventListener(ev, fn, opt); cleanups.push(() => target.removeEventListener(ev, fn, opt)); },
    };
    current = { def, cleanups, api: null };
    try { current.api = def.mount(ctx) || null; } catch (e) {
      console.error(e);
      host.innerHTML = `<pre style="color:#f88;padding:24px;font:12px ui-monospace">${SS.fmt.esc(e.stack || e)}</pre>`;
    }
    writeURL();
    renderBar();
  }

  function go(patch) {
    const changedShellOrVariant = (patch.variant && patch.variant !== state.variant) || (patch.shell && patch.shell !== state.shell);
    Object.assign(state, patch);
    if (changedShellOrVariant && !('view' in patch)) state.view = '';
    mount();
  }

  function cycle(dir) {
    const i = ORDER.indexOf(state.variant);
    go({ variant: ORDER[(i + dir + ORDER.length) % ORDER.length] });
  }

  const bar = document.getElementById('protobar');
  function renderBar() {
    bar.hidden = !state.bar;
    const def = VARIANTS[state.variant];
    bar.innerHTML = `
      <button class="pb-arrow" data-dir="-1" aria-label="Previous direction"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg></button>
      <div class="pb-label"><span class="pb-key">${def.key}</span><span class="pb-name">${def.name}</span></div>
      <button class="pb-arrow" data-dir="1" aria-label="Next direction"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg></button>
      <span class="pb-sep"></span>
      <div class="pb-shells" role="tablist">${SHELLS.map((s) => `<button role="tab" aria-selected="${s.id === state.shell}" data-shell="${s.id}"><span class="pb-long">${s.label}<small>${s.hint}</small></span><span class="pb-short">${s.short}</span></button>`).join('')}</div>
      <span class="pb-sep"></span>
      <span class="pb-note" title="Every title, poster, person, Node and number is invented">Prototype · synthetic data</span>`;
  }
  bar.addEventListener('click', (e) => {
    const a = e.target.closest('[data-dir]');
    if (a) return cycle(+a.dataset.dir);
    const s = e.target.closest('[data-shell]');
    if (s) go({ shell: s.dataset.shell });
  });
  addEventListener('keydown', (e) => {
    if (e.target.closest('input, textarea, select, [contenteditable="true"]') || e.metaKey || e.ctrlKey || e.altKey) return;
    if (e.defaultPrevented) return;
    if (e.key === 'ArrowLeft') return cycle(-1);
    if (e.key === 'ArrowRight') return cycle(1);
    if (e.key === '\\') { state.bar = !state.bar; renderBar(); fitDevice(); }
  });
  addEventListener('resize', () => {
    fitDevice();
    const box = stageEl.querySelector('.scaled');
    if (box) fitScaled(box);
  });

  async function boot() {
    const bootEl = document.getElementById('boot');
    const fill = bootEl.querySelector('.boot__fill');
    await document.fonts.ready;
    await SS.art.prewarm((p) => (fill.style.transform = `scaleX(${p})`));
    bootEl.classList.add('boot--done');
    setTimeout(() => bootEl.remove(), 500);
    if (!state.still) SS.sim.start();
    mount();
    window.__ss = {
      go: (variant, shell, view) => new Promise((res) => { go({ variant, shell, view: view || '' }); requestAnimationFrame(() => requestAnimationFrame(res)); }),
      state,
    };
    document.documentElement.dataset.ready = '1';
  }
  // Variant scripts load after this file; boot once they have all registered.
  if (document.readyState === 'loading') addEventListener('DOMContentLoaded', boot);
  else setTimeout(boot);
})();
