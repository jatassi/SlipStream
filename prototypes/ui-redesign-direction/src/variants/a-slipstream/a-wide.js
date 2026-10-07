(function () {
  const SS = window.SS;
  const d = SS.data;
  const fmt = SS.fmt;
  const VA = window.VA;
  const esc = fmt.esc;
  const icon = VA.icon;

  const NAV = [['home', 'Flow'], ['library', 'Movies'], ['series', 'Series'], ['activity', 'Activity'], ['requests', 'Requests'], ['nodes', 'Nodes'], ['discover', 'Discover']];
  const NAV_END = [['settings', 'Settings'], ['system', 'System']];
  const navIcon = (id) => ({
    home: icon('wind', { size: 20 }), library: icon('film', { size: 20 }), series: icon('tv', { size: 20 }), activity: VA.flowlines(20),
    requests: icon('inbox', { size: 20 }), nodes: icon('server', { size: 20 }), discover: icon('compass', { size: 20 }), settings: icon('settings', { size: 20 }), system: icon('gauge', { size: 20 }),
  }[id]);
  const TITLES = { home: 'Flow', library: 'Movies', series: 'Series', activity: 'Activity', requests: 'Requests', nodes: 'Nodes', discover: 'Discover', settings: 'Settings', system: 'System' };

  VA.lib = VA.lib || { movie: { view: 'grid', size: 'm', sort: 'added', dir: 1, status: 'all' }, series: { view: 'grid', size: 'm', sort: 'added', dir: 1, status: 'all' } };
  VA.actState = VA.actState || { filter: 'all', node: 'all' };
  VA.reqState = VA.reqState || { filter: 'pending' };

  const bucket = (w) => {
    if (w.type === 'series') return w.counts.downloading ? 'downloading' : w.counts.missing ? 'missing' : 'available';
    if (d.jobs.some((j) => j.work === w.id && j.state === 'failed')) return 'failed';
    return w.status;
  };
  const sizeOf = (w) => (w.type === 'series' ? w.size || 0 : (w.files || []).reduce((s, f) => s + f.size, 0));
  const SORTS = {
    added: ['Recently added', (a, b) => b.added - a.added],
    title: ['Title', (a, b) => a.title.localeCompare(b.title)],
    year: ['Year', (a, b) => b.year - a.year || a.title.localeCompare(b.title)],
    rating: ['Rating', (a, b) => (b.rating || 0) - (a.rating || 0)],
    size: ['Size on disk', (a, b) => sizeOf(b) - sizeOf(a)],
  };
  const STATUS = [['all', 'All'], ['available', 'Available'], ['downloading', 'Downloading'], ['missing', 'Missing'], ['upgradable', 'Upgradable'], ['unreleased', 'Unreleased'], ['failed', 'Failed']];

  const releasesFor = (w) => {
    const base = w.title.replace(/[^A-Za-z0-9 ]/g, '').replace(/ +/g, '.');
    const s = w.type === 'series' ? `.S${String(w.seasons.length).padStart(2, '0')}` : `.${w.year}`;
    return [
      { name: `${base}${s}.2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC.TrueHD.Atmos.7.1-FRAMESTOR`, target: 'uhd', size: 61.8e9, seeds: 212, indexer: 'Halide', age: '3 days' },
      { name: `${base}${s}.2160p.WEB-DL.DV.HDR.DDP5.1.Atmos.H.265-BYNDR`, target: 'uhd', size: 18.2e9, seeds: 640, indexer: 'Aperture', age: '6 days' },
      { name: `${base}${s}.1080p.BluRay.x264.DTS-HD.MA.5.1-HONE`, target: 'hd', size: 12.4e9, seeds: 1180, indexer: 'Kestrel', age: '3 days' },
      { name: `${base}${s}.1080p.WEB-DL.DDP5.1.H.264-NTb`, target: 'hd', size: 5.1e9, seeds: 2304, indexer: 'Kestrel', age: '9 days' },
      { name: `${base}${s}.720p.WEBRip.x264-SPiRiT`, target: null, size: 1.9e9, seeds: 88, indexer: 'Lumen Torznab', age: '9 days', reject: 'Below every Target in the Profile' },
      { name: `${base}${s}.1080p.WEBRip.x264-SPiRiT.sample`, target: null, size: 0.048e9, seeds: 4, indexer: 'Lumen Torznab', age: '12 days', reject: 'On the Blocklist' },
    ];
  };

  const VIEWS = {};

  VA.mountWide = (ctx) => {
    const root = ctx.root;
    root.classList.add('a-app', 'a-wide');
    root.innerHTML = `
      <nav class="a-rail" aria-label="Main">
        <a class="a-rail__mark" href="#home" aria-label="SlipStream, Flow">${VA.mark(26)}</a>
        <div class="a-rail__group">${NAV.map(([id, label]) => railItem(id, label)).join('')}</div>
        <div class="a-rail__group a-rail__group--end">${NAV_END.map(([id, label]) => railItem(id, label)).join('')}
          <button class="a-rail__item" data-pop="account" aria-label="Account, Alex Rowan"><span class="a-rail__ic">${VA.avatar('alex', 24)}</span><span class="a-rail__label">Account</span></button>
        </div>
      </nav>
      <div class="a-main">
        <header class="a-top">
          <div class="a-top__title"><a class="a-top__back" data-back hidden href="#library">${icon('chevron-left', { size: 16 })}<span data-back-label>Movies</span></a><p class="a-top__name" data-title>Flow</p></div>
          <button class="a-cmdk" data-cmdk type="button">${icon('search', { size: 15 })}<span>Search or run a command</span><kbd>⌘K</kbd></button>
          <div class="a-top__right">
            <a class="a-tput a-num" href="#activity" title="Live throughput across Nodes"><span>${icon('arrow-down', { size: 13 })}<b data-tput="down"></b></span><span>${icon('arrow-up', { size: 13 })}<b data-tput="up"></b></span></a>
            <button class="a-ibtn" data-pop="tasks" aria-label="Running tasks">${icon('list-checks', { size: 18 })}<span class="a-dot" data-dot="tasks"></span></button>
            <button class="a-ibtn" data-pop="bell" aria-label="Notifications">${icon('bell', { size: 18 })}<span class="a-dot a-dot--n a-num" data-dot="bell"></span></button>
          </div>
        </header>
        <main class="a-view" data-view tabindex="-1"></main>
      </div>
      <div class="a-layer" data-layer></div>`;

    const viewEl = root.querySelector('[data-view]');
    const layer = root.querySelector('[data-layer]');
    const q = (s) => root.querySelector(s);
    let cur = null, curName = '', curArg = '';
    const scrollMem = {};
    const fils = VA.Filaments(root);
    // The top bar names the page only once its display head has scrolled away.
    const topEl = q('.a-top');
    const headIO = new IntersectionObserver(([e]) => topEl.classList.toggle('is-headed', e.isIntersecting), { root: viewEl });
    const watchHead = () => {
      headIO.disconnect();
      const h = viewEl.querySelector('[data-head]');
      topEl.classList.toggle('is-headed', !!h);
      if (h) headIO.observe(h);
    };

    const api = {
      root, ctx, layer,
      go: (v, o) => go(v, o),
      toast: (t) => VA.toast(root, t),
      setView: (v) => ctx.setView(v),
    };

    function go(v, { vt = null, restore = false } = {}) {
      v = v || 'home';
      const [name, ...rest] = v.split('/');
      const fn = VIEWS[name] ? name : 'home';
      const arg = rest.join('/');
      closeLayer();
      if (curName) scrollMem[curName] = viewEl.scrollTop;
      ctx.setView(v);
      const render = () => {
        cur?.destroy?.();
        viewEl.innerHTML = '';
        viewEl.className = `a-view a-view--${fn}`;
        cur = VIEWS[fn](viewEl, arg, api) || {};
        curName = fn; curArg = arg;
        viewEl.scrollTop = restore ? scrollMem[fn] || 0 : 0;
        chrome();
        watchHead();
        vt?.after?.();
      };
      if (vt && document.startViewTransition && !VA.reduced()) {
        vt.before?.();
        const tr = document.startViewTransition(render);
        tr.finished.finally(() => root.querySelectorAll('img[style*="view-transition-name"]').forEach((x) => { x.style.viewTransitionName = ''; }));
      } else render();
    }

    function chrome() {
      const w = curName === 'work' ? d.W[curArg] : null;
      const section = w ? (w.type === 'series' ? 'series' : 'library') : curName;
      root.querySelectorAll('.a-rail__item[data-nav]').forEach((a) => {
        if (a.dataset.nav === section) a.setAttribute('aria-current', 'page'); else a.removeAttribute('aria-current');
      });
      const back = q('[data-back]');
      back.hidden = !w;
      if (w) { back.setAttribute('href', '#' + section); q('[data-back-label]').textContent = TITLES[section]; }
      q('[data-title]').textContent = w ? w.title : curName === 'activity' && curArg ? 'Activity' : TITLES[curName] || 'Flow';
      counts();
    }

    function counts() {
      const t = SS.sim.totals();
      q('[data-tput="down"]').textContent = fmt.rate(t.down);
      q('[data-tput="up"]').textContent = fmt.rate(t.up);
      const running = d.jobs.some((j) => j.state === 'running');
      root.querySelectorAll('.a-rail .a-flowlines').forEach((s) => s.classList.toggle('is-running', running && !VA.reduced()));
      const pend = d.requests.filter((r) => r.state === 'pending').length;
      const rb = q('[data-badge="requests"]');
      if (rb) { rb.textContent = pend ? String(pend) : ''; rb.hidden = !pend; }
      const bell = d.alerts.filter((a) => !a.read).length;
      const bd = q('[data-dot="bell"]');
      bd.textContent = bell ? String(bell) : '';
      bd.hidden = !bell;
      q('[data-dot="tasks"]').hidden = false;
    }

    ctx.on('second', () => { counts(); cur?.live?.(); });
    for (const ev of ['job:new', 'job:done', 'job:update', 'request:update', 'work:update', 'history:new']) ctx.on(ev, (p) => { cur?.refresh?.(ev, p); counts(); });

    const posterVT = (img) => ({
      before() { img.style.viewTransitionName = 'a-poster'; },
      after() { const p = root.querySelector('.a-work__poster'); if (p) p.style.viewTransitionName = 'a-poster'; },
    });
    const backVT = (id) => ({
      before() { const p = root.querySelector('.a-work__poster'); if (p) p.style.viewTransitionName = 'a-poster'; },
      after() { const img = root.querySelector(`img[data-vt="${CSS.escape(id)}"]`); if (img) img.style.viewTransitionName = 'a-poster'; },
    });

    ctx.listen(root, 'click', (e) => {
      const t = e.target;
      if (t.closest('[data-cmdk]')) return openPalette(false);
      const pop = t.closest('[data-pop]');
      if (pop) return togglePop(pop.dataset.pop, pop);
      if (cur?.click?.(e) === true) return;
      if (handleCommon(e, api)) return;
      const a = t.closest('a[href^="#"]');
      if (a && root.contains(a)) {
        e.preventDefault();
        const v = a.getAttribute('href').slice(1);
        if (a.closest('.a-layer')) closeLayer();
        if (a.matches('[data-back]') && curName === 'work') return go(v, { vt: backVT(curArg), restore: true });
        const img = a.querySelector('img[data-vt]');
        if (v.startsWith('work/') && img) return go(v, { vt: posterVT(img) });
        go(v);
      }
    });
    ctx.listen(root, 'submit', (e) => { e.preventDefault(); handleSubmit(e, api); });
    ctx.listen(window, 'keydown', (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); return layer.dataset.kind === 'palette' ? closeLayer() : openPalette(true); }
      if (e.key === 'Escape' && layer.dataset.kind) { e.preventDefault(); closeLayer(); }
      if (e.key === '/' && !e.target.closest('input, textarea, select') && !layer.dataset.kind) { e.preventDefault(); openPalette(true); }
    });

    function closeLayer() {
      if (!layer.dataset.kind) return;
      layer.innerHTML = '';
      delete layer.dataset.kind;
      root.querySelectorAll('[data-pop][aria-expanded]').forEach((b) => b.removeAttribute('aria-expanded'));
    }
    api.closeLayer = closeLayer;
    function anchorTo(trigger, el, align = 'right') {
      const rr = root.getBoundingClientRect();
      const s = rr.width / root.offsetWidth || 1;
      const r = trigger.getBoundingClientRect();
      const x = (align === 'right' ? r.right : r.left) - rr.left;
      el.style.top = (r.bottom - rr.top) / s + 8 + 'px';
      if (align === 'right') { el.style.right = (rr.width - x) / s + 'px'; el.style.transformOrigin = 'top right'; }
      else if (align === 'rail') { el.style.left = (r.right - rr.left) / s + 10 + 'px'; el.style.top = 'auto'; el.style.bottom = (rr.bottom - r.bottom) / s + 'px'; el.style.transformOrigin = 'bottom left'; }
      else { el.style.left = x / s + 'px'; el.style.transformOrigin = 'top left'; }
    }
    function togglePop(kind, trigger) {
      if (layer.dataset.kind === kind) return closeLayer();
      closeLayer();
      const html = POPS[kind]?.(api, trigger);
      if (!html) return;
      layer.dataset.kind = kind;
      layer.innerHTML = `<div class="a-scrim" data-close></div><div class="a-pop a-pop--${kind}" role="dialog">${html}</div>`;
      const el = layer.querySelector('.a-pop');
      anchorTo(trigger, el, kind === 'account' ? 'rail' : trigger.dataset.align || 'right');
      trigger.setAttribute('aria-expanded', 'true');
      layer.querySelector('[data-close]').addEventListener('pointerdown', closeLayer);
    }
    api.togglePop = togglePop;

    // Command palette: no animation, the keyboard is faster than any motion.
    function openPalette() {
      closeLayer();
      layer.dataset.kind = 'palette';
      layer.innerHTML = `<div class="a-scrim a-scrim--palette" data-close></div>
        <div class="a-palette" role="dialog" aria-label="Command palette">
          <div class="a-palette__field">${icon('search', { size: 17 })}<input class="a-palette__input" placeholder="Search Works, go to a view, or run a command" aria-label="Search Works, views and commands" autocomplete="off" spellcheck="false" role="combobox" aria-expanded="true" aria-controls="a-pal-list"><kbd>esc</kbd></div>
          <div class="a-palette__list" id="a-pal-list" role="listbox"></div>
          <div class="a-palette__foot"><span><kbd>↑</kbd><kbd>↓</kbd> to move</span><span><kbd>↵</kbd> to open</span></div>
        </div>`;
      const input = layer.querySelector('input');
      const list = layer.querySelector('.a-palette__list');
      let items = [], sel = 0;
      const render = () => {
        items = paletteItems(input.value.trim());
        sel = Math.min(sel, Math.max(0, items.length - 1));
        let html = '', group = '';
        items.forEach((it, i) => {
          if (it.group !== group) { group = it.group; html += `<div class="a-palette__group" role="presentation">${group}</div>`; }
          html += `<div class="a-palette__item ${i === sel ? 'is-sel' : ''}" role="option" aria-selected="${i === sel}" data-i="${i}">${it.art || `<span class="a-palette__ic">${it.ic}</span>`}<span class="a-palette__label">${esc(it.label)}</span><span class="a-palette__hint">${esc(it.hint || '')}</span></div>`;
        });
        list.innerHTML = html || `<div class="a-palette__empty">No Works, views or commands match “${esc(input.value)}”.</div>`;
      };
      const run = (i) => { const it = items[i]; if (!it) return; closeLayer(); it.run(); };
      input.addEventListener('input', () => { sel = 0; render(); });
      input.addEventListener('keydown', (e) => {
        if (e.key === 'ArrowDown') { e.preventDefault(); sel = (sel + 1) % Math.max(1, items.length); render(); list.querySelector('.is-sel')?.scrollIntoView({ block: 'nearest' }); }
        else if (e.key === 'ArrowUp') { e.preventDefault(); sel = (sel - 1 + items.length) % Math.max(1, items.length); render(); list.querySelector('.is-sel')?.scrollIntoView({ block: 'nearest' }); }
        else if (e.key === 'Enter') { e.preventDefault(); run(sel); }
      });
      list.addEventListener('pointermove', (e) => { const o = e.target.closest('[data-i]'); if (o && +o.dataset.i !== sel) { sel = +o.dataset.i; list.querySelectorAll('.a-palette__item').forEach((x) => { const on = +x.dataset.i === sel; x.classList.toggle('is-sel', on); x.setAttribute('aria-selected', on); }); } });
      list.addEventListener('click', (e) => { const o = e.target.closest('[data-i]'); if (o) run(+o.dataset.i); });
      layer.querySelector('[data-close]').addEventListener('pointerdown', closeLayer);
      render();
      input.focus();
    }
    function paletteItems(qs) {
      const ql = qs.toLowerCase();
      const out = [];
      const match = (s) => !ql || s.toLowerCase().includes(ql);
      const works = d.works.filter((w) => match(w.title)).sort((a, b) => (ql ? (a.title.toLowerCase().startsWith(ql) ? -1 : 0) - (b.title.toLowerCase().startsWith(ql) ? -1 : 0) : b.added - a.added)).slice(0, ql ? 6 : 4);
      const cmds = [];
      const failed = d.jobs.find((j) => j.state === 'failed');
      if (failed) cmds.push({ label: `Blocklist ${d.W[failed.work].title} and search`, hint: 'Separation', ic: VA.vortex(16), run: () => { VA.act.blocklist(failed); api.toast('Blocklisted the sample. Searching Kestrel for another release.'); } });
      const stalled = d.jobs.find((j) => j.state === 'stalled');
      if (stalled) cmds.push({ label: `Find another release for ${d.W[stalled.work].title}`, hint: 'Stalled', ic: VA.stall(16), run: () => { VA.act.findAnother(stalled); api.toast('Looking for a release with seeders.'); } });
      cmds.push({ label: 'Search for missing episodes', hint: 'Series', ic: icon('search', { size: 16 }), run: () => { const n = VA.act.searchMissingEpisodes(); api.toast(n ? `Searching for ${fmt.plural(n, 'missing episode')}.` : 'No episodes are missing.'); } });
      const ams = d.N['ams-1'];
      cmds.push(ams.paused ? { label: 'Resume ams-1', hint: 'Node', ic: icon('play', { size: 16 }), run: () => { VA.act.pauseNode('ams-1', false); api.toast('ams-1 is downloading again.'); } } : { label: 'Pause ams-1', hint: 'Node', ic: icon('pause', { size: 16 }), run: () => { VA.act.pauseNode('ams-1', true); api.toast('ams-1 is paused. Its downloads hold where they are.'); } });
      const pend = d.requests.filter((r) => r.state === 'pending').length;
      if (pend) cmds.push({ label: `Review ${fmt.plural(pend, 'pending request')}`, hint: 'Requests', ic: icon('inbox', { size: 16 }), run: () => go('requests') });
      cmds.push({ label: 'Refresh metadata for the whole library', hint: 'Library', ic: icon('refresh-cw', { size: 16 }), run: () => api.toast('Refreshing metadata for 58 Works.') });
      const views = [...NAV, ...NAV_END].map(([id, label]) => ({ label: `Go to ${label}`, hint: '', ic: navIcon(id), run: () => go(id) }));
      if (ql) {
        works.forEach((w) => out.push({ group: 'Works', label: w.title, hint: `${w.type === 'series' ? 'Series' : 'Movie'} · ${w.year}`, art: `<span class="a-palette__art">${VA.posterImg(w, 'a-poster--xs')}</span>`, run: () => go('work/' + w.id) }));
        cmds.filter((c) => match(c.label)).forEach((c) => out.push({ ...c, group: 'Commands' }));
        views.filter((c) => match(c.label)).forEach((c) => out.push({ ...c, group: 'Go to' }));
      } else {
        cmds.forEach((c) => out.push({ ...c, group: 'Commands' }));
        views.forEach((c) => out.push({ ...c, group: 'Go to' }));
        works.forEach((w) => out.push({ group: 'Recently added', label: w.title, hint: `${w.type === 'series' ? 'Series' : 'Movie'} · ${w.year}`, art: `<span class="a-palette__art">${VA.posterImg(w, 'a-poster--xs')}</span>`, run: () => go('work/' + w.id) }));
      }
      return out;
    }

    go(ctx.view || 'home');
    return { unmount() { cur?.destroy?.(); fils.destroy(); headIO.disconnect(); } };
  };
  const head = (title, sub = '') => `<header class="a-dhead"><h1 class="a-dhead__t" data-head>${esc(title)}</h1>${sub ? `<p class="a-dhead__sub">${sub}</p>` : ''}</header>`;

  function railItem(id, label) {
    const badge = id === 'requests' ? `<span class="a-rail__badge a-num" data-badge="requests" hidden></span>` : '';
    return `<a class="a-rail__item" href="#${id}" data-nav="${id}"><span class="a-rail__ic">${navIcon(id)}${badge}</span><span class="a-rail__label">${label}</span></a>`;
  }

  function handleCommon(e, api) {
    const t = e.target;
    const b = (sel) => t.closest(sel);
    let el;
    if ((el = b('[data-need]'))) {
      const id = el.dataset.need;
      const n = VA.needs().find((x) => x.id === id);
      if (!n) return true;
      if (n.job && n.job.state === 'failed') { el.classList.add('is-loading'); VA.act.blocklist(n.job); api.toast('Blocklisted the sample. Searching Kestrel for another release.'); }
      else if (n.job && n.job.state === 'stalled') { el.classList.add('is-loading'); VA.act.findAnother(n.job); api.toast('Looking for a release with seeders.'); }
      else if (n.node) api.go('nodes/' + n.node.id);
      else if (id === 'requests') api.go('requests');
      return true;
    }
    if ((el = b('[data-blocklist]'))) { const j = d.jobs.find((x) => x.id === el.dataset.blocklist); if (j) { el.classList.add('is-loading'); VA.act.blocklist(j); api.toast('Blocklisted the sample. Searching Kestrel for another release.'); } return true; }
    if ((el = b('[data-another]'))) { const j = d.jobs.find((x) => x.id === el.dataset.another); if (j) { VA.act.findAnother(j); api.toast('Looking for a release with seeders.'); } return true; }
    if ((el = b('[data-open-work]'))) { api.go('work/' + el.dataset.openWork); return true; }
    if ((el = b('[data-toggle]'))) { const j = d.jobs.find((x) => x.id === el.dataset.toggle); if (j) VA.act.toggleJob(j); return true; }
    if ((el = b('[data-approve]'))) {
      const r = d.requests.find((x) => x.id === el.dataset.approve);
      el.classList.add('is-loading');
      setTimeout(() => { VA.act.approve(r); api.toast(`Approved ${d.W[r.work].title} for ${VA.firstName(r.by)}.`); }, 380);
      return true;
    }
    if ((el = b('[data-deny]'))) {
      const row = el.closest('.a-req');
      const f = row.querySelector('[data-deny-form]');
      f.hidden = false;
      row.querySelector('.a-req__acts').hidden = true;
      f.querySelector('input').focus();
      return true;
    }
    if ((el = b('[data-deny-cancel]'))) {
      const row = el.closest('.a-req');
      row.querySelector('[data-deny-form]').hidden = true;
      row.querySelector('.a-req__acts').hidden = false;
      return true;
    }
    if ((el = b('[data-pop-cmd]'))) {
      const c = el.dataset.popCmd;
      api.closeLayer();
      if (c === 'read') { d.alerts.forEach((a) => { a.read = true; }); api.toast('Marked every notification as read.'); }
      else if (c === 'run') api.toast(`${el.dataset.task} is running now.`);
      else if (c === 'signout') api.toast('Signing out is off in the prototype.');
      else if (c === 'update') api.go('system');
      else if (c.startsWith('go:')) api.go(c.slice(3));
      else api.toast(el.dataset.msg || 'Done.');
      return true;
    }
    return false;
  }
  function handleSubmit(e, api) {
    const f = e.target.closest('[data-deny-form]');
    if (f) {
      const r = d.requests.find((x) => x.id === f.dataset.denyForm);
      VA.act.deny(r, f.reason.value.trim());
      api.toast(`Denied ${d.W[r.work].title}. ${VA.firstName(r.by)} sees your reason.`);
    }
  }

  const POPS = {
    bell() {
      const list = d.alerts.slice().sort((a, b) => b.at - a.at);
      return `<div class="a-pop__head"><b>Notifications</b><button class="a-link" data-pop-cmd="read">Mark all as read</button></div>
        <div class="a-pop__list">${list.map((a) => `<button class="a-note ${a.read ? 'is-read' : ''}" data-pop-cmd="${a.node ? 'go:nodes/' + a.node : a.work ? 'go:work/' + a.work : a.title.startsWith('SlipStream') ? 'update' : 'go:system'}"><span class="a-note__g">${a.level === 'error' ? VA.vortex(16) : a.level === 'warning' ? VA.watchGlyph(a.node ? 'behind' : 'stall', 16) : VA.laminar(16)}</span><span class="a-note__t"><b>${esc(a.title)}</b><span>${esc(a.detail)}</span></span><span class="a-note__time a-num">${esc(fmt.ago(a.at))}</span></button>`).join('')}</div>`;
    },
    tasks() {
      const clock = SS.sim.clock;
      const next = new Date(Math.ceil(clock / (15 * 60e3)) * 15 * 60e3);
      const rows = [
        ['RSS sync', `Every 15 min · next at ${fmt.time(next)}`, 'Run now'],
        ['Refresh metadata', 'Running on atlas · 41 of 58 Works checked', ''],
        ['Library scan', 'vault · nightly at 03:00', 'Run now'],
        ['Backup', 'atlas · nightly at 04:00 · last 4.1 GB', 'Run now'],
      ];
      return `<div class="a-pop__head"><b>Running tasks</b><span class="a-pop__sub">1 running</span></div>
        <div class="a-pop__list">${rows.map(([n, s, a]) => `<div class="a-task"><span class="a-task__t"><b>${n}</b><span>${esc(s)}</span>${a ? '' : VA.prog(0.71)}</span>${a ? VA.btn(a, { sm: true, attrs: `data-pop-cmd="run" data-task="${n}"` }) : '<span class="a-task__live a-num">71%</span>'}</div>`).join('')}</div>`;
    },
    account() {
      const me = d.A[d.me];
      return `<div class="a-acct">${VA.avatar('alex', 36)}<span><b>${esc(me.name)}</b><span>${esc(me.email)}</span><span class="a-tag">Admin</span></span></div>
        <div class="a-pop__list a-pop__menu"><button class="a-menu" data-pop-cmd="go:settings">${icon('settings', { size: 16 })}Account and passkeys</button><button class="a-menu" data-pop-cmd="go:settings">${icon('user', { size: 16 })}Accounts and Members</button><button class="a-menu" data-pop-cmd="signout">${icon('log-out', { size: 16 })}Sign out</button></div>`;
    },
    more(api, trigger) {
      const w = d.W[trigger.dataset.work];
      return `<div class="a-pop__list a-pop__menu">
        <button class="a-menu" data-pop-cmd="msg" data-msg="Refreshing metadata for ${esc(w.title)}.">${icon('refresh-cw', { size: 16 })}Refresh metadata</button>
        <button class="a-menu" data-pop-cmd="msg" data-msg="Profile editing opens in Settings in the full build.">${icon('sliders-horizontal', { size: 16 })}Change Profile</button>
        <button class="a-menu" data-pop-cmd="msg" data-msg="${esc(w.title)} is wanted in ${esc(fmt.list(w.editions))}.">${icon('layers', { size: 16 })}Editions</button>
        <button class="a-menu" data-pop-cmd="go:activity">${icon('history', { size: 16 })}History</button>
        <div class="a-menu__sep"></div>
        <button class="a-menu a-menu--quiet" data-pop-cmd="msg" data-msg="Deleting is off in the prototype.">${icon('trash-2', { size: 16 })}Delete Work and files</button>
      </div>`;
    },
  };

  VIEWS.home = (el, _, api) => {
    const day = fmt.day(SS.sim.clock);
    el.innerHTML = `<div class="a-home">
      <div class="a-lede">
        <h1 class="a-sentence"><span data-s="lead"></span><span data-s="at"> at <span class="a-num" data-s="rate"></span></span>.<br><button class="a-sentence__needs" type="button" data-s="needs"></button></h1>
        <p class="a-lede__when a-num">${esc(day)} · <span data-s="clock"></span></p>
      </div>
      <div class="a-home__tunnel" data-tunnel aria-label="The Tunnel"></div>
      <div class="a-home__grid">
        <section aria-labelledby="a-h-needs">
          <div class="a-section-head"><h2 class="a-h2" id="a-h-needs">Needs you</h2><a class="a-link" href="#activity">Activity ${icon('chevron-right', { size: 14 })}</a></div>
          <div class="a-list" data-list="needs"></div>
          <div class="a-watch" data-watch-box><h3 class="a-watch__h">Watching</h3><div class="a-watch__list" data-list="watch"></div></div>
        </section>
        <section aria-labelledby="a-h-arr">
          <div class="a-section-head"><h2 class="a-h2" id="a-h-arr">Arrived tonight <span class="a-fresh a-num" data-s="fresh"></span></h2><a class="a-link" href="#library">Movies ${icon('chevron-right', { size: 14 })}</a></div>
          <div class="a-list" data-list="arr"></div>
        </section>
      </div>
      <section class="a-home__nodes" aria-labelledby="a-h-nodes">
        <div class="a-section-head"><h2 class="a-h2" id="a-h-nodes">Nodes <span class="a-count" data-s="nodes"></span></h2><a class="a-link" href="#nodes">All Nodes ${icon('chevron-right', { size: 14 })}</a></div>
        <div class="a-nstrip" data-list="nodes">${d.nodes.map(VA.nodeCell).join('')}</div>
      </section>
    </div>`;
    const q = (s) => el.querySelector(s);
    const needsEl = q('[data-list="needs"]');
    const watchEl = q('[data-list="watch"]');
    const arrEl = q('[data-list="arr"]');
    const nodesEl = q('[data-list="nodes"]');
    const needsBtn = q('[data-s="needs"]');
    const tun = VA.Tunnel(q('[data-tunnel]'), {
      onOpen: (F) => api.go(F.job ? 'activity/' + F.job.id : 'work/' + F.work.id),
      onOpenWork: (w) => api.go('work/' + w.id),
      onResolve: (msg) => api.toast(msg),
    });

    const sentence = (first) => {
      const s = VA.sentence();
      if (first) q('[data-s="lead"]').textContent = s.lead; else VA.flip(q('[data-s="lead"]'), s.lead);
      q('[data-s="at"]').hidden = !s.rate;
      q('[data-s="rate"]').textContent = s.rate;
      if (first) needsBtn.textContent = s.needs; else VA.flip(needsBtn, s.needs);
      needsBtn.disabled = !s.sep;
      q('[data-s="clock"]').textContent = fmt.time(SS.sim.clock);
      const off = d.nodes.filter((n) => n.state === 'offline').length;
      q('[data-s="nodes"]').textContent = `${d.nodes.length}${off ? `, ${off} offline` : ''}`;
    };
    let needIds = '', arrIds = '';
    const needs = (force) => {
      const list = VA.needs();
      const watch = VA.watching();
      const ids = list.map((n) => n.id + n.title).join() + '|' + watch.map((n) => n.id + n.title).join();
      if (force || ids !== needIds) {
        needIds = ids;
        VA.patch(needsEl, list, (n) => n.id, VA.needRow);
        emptyState(needsEl, !list.length, 'Nothing needs you.', 'Failures and offline Nodes appear here the moment they happen.');
        VA.patch(watchEl, watch, (n) => n.id, VA.watchItem);
        q('[data-watch-box]').hidden = !watch.length;
      } else for (const n of list) { const t = needsEl.querySelector(`[data-key="${CSS.escape(n.id)}"] .a-need__time`); if (t && n.at) t.textContent = fmt.ago(n.at); }
    };
    const arrived = (force) => {
      const all = VA.arrivals(12);
      const list = all.slice(0, 3);
      const ids = list.map(VA.arrivalKey).join() + [...VA.seen].length;
      const fresh = all.filter((h) => !VA.seen.has(VA.arrivalKey(h))).length;
      q('[data-s="fresh"]').textContent = fresh ? `${fresh} new` : '';
      if (force || ids !== arrIds) {
        arrIds = ids;
        VA.patch(arrEl, list, VA.arrivalKey, VA.arrRow);
        emptyState(arrEl, !list.length, 'Nothing has arrived tonight.', 'Imports land here with the time they arrived, marked until you have seen them.');
      } else arrEl.querySelectorAll('.a-arr').forEach((r) => { const h = list.find((x) => VA.arrivalKey(x) === r.dataset.arr); if (h) r.querySelector('.a-arr__time').textContent = fmt.ago(h.at); });
    };
    sentence(true);
    needs(true);
    arrived(true);
    // Only an arrival that lands while the page is open flashes its dye.
    const arm = setTimeout(() => { arrEl.querySelectorAll('.a-enter').forEach((r) => r.classList.remove('a-enter')); arrEl.classList.add('is-live'); }, 320);

    let litRow = null;
    const over = (e) => {
      const row = e.target.closest('.a-need, .a-witem, .a-arr.is-new');
      if (row?.classList.contains('a-arr')) { VA.seen.add(row.dataset.arr); row.classList.remove('is-new'); arrived(true); return; }
      if (row === litRow) return;
      litRow = row;
      tun.focus(row?.dataset.flow || null);
    };
    const leave = () => { litRow = null; tun.focus(null); };
    el.addEventListener('pointerover', over);
    el.addEventListener('focusin', over);
    needsEl.addEventListener('pointerleave', leave);
    watchEl.addEventListener('pointerleave', leave);
    needsBtn.addEventListener('pointerenter', () => { tun.focusSeparations(true); needsEl.querySelectorAll('.a-need').forEach((r) => r.classList.add('is-lit')); });
    needsBtn.addEventListener('pointerleave', () => { tun.focusSeparations(false); needsEl.querySelectorAll('.is-lit').forEach((r) => r.classList.remove('is-lit')); });

    return {
      click(e) {
        if (e.target.closest('[data-s="needs"]')) { tun.focusSeparations(false); if (!tun.pinSeparation()) api.go('activity'); return true; }
        const w = e.target.closest('[data-watch]');
        if (w) {
          const n = VA.watching().find((x) => x.id === w.dataset.watch);
          if (!n) return true;
          if (n.flow && tun.pin(n.flow)) return true;
          api.go(n.node ? 'nodes/' + n.node.id : n.job ? 'activity/' + n.job.id : 'requests');
          return true;
        }
        return false;
      },
      live() { sentence(); needs(); arrived(); VA.liveNodes(nodesEl); },
      refresh() { sentence(); needs(true); arrived(true); },
      destroy() { tun.destroy(); clearTimeout(arm); },
    };
  };

  function emptyState(box, on, title, sub) {
    let e = box.querySelector(':scope > .a-empty');
    if (on && !e) { box.insertAdjacentHTML('beforeend', `<div class="a-empty"><p>${esc(title)}</p><p class="a-empty__sub">${esc(sub)}</p></div>`); }
    else if (!on && e) e.remove();
  }
  VA.emptyState = emptyState;

  VIEWS.library = (el, _, api) => worksView(el, 'movie', api);
  VIEWS.series = (el, _, api) => worksView(el, 'series', api);
  function worksView(el, type, api) {
    const st = VA.lib[type];
    const all = d.works.filter((w) => w.type === type);
    el.innerHTML = `<div class="a-page a-lib">
      ${head(type === 'series' ? 'Series' : 'Movies')}
      <div class="a-lib__bar">
        <div data-slot="status"></div>
        <div class="a-lib__tools">
          <label class="a-lib__sort"><span>Sort</span><select class="a-select" data-sort>${Object.entries(SORTS).map(([k, [label]]) => `<option value="${k}" ${k === st.sort ? 'selected' : ''}>${label}</option>`).join('')}</select></label>
          <div class="a-seg a-seg--icons" role="tablist" aria-label="Poster size" data-slot="size"></div>
          <div class="a-seg a-seg--icons" role="tablist" aria-label="Layout" data-slot="mode"></div>
        </div>
      </div>
      <div class="a-lib__body" data-body></div>
    </div>`;
    const q = (s) => el.querySelector(s);
    const body = q('[data-body]');
    const bar = () => {
      const counts = Object.fromEntries(STATUS.map(([k]) => [k, k === 'all' ? all.length : all.filter((w) => bucket(w) === k).length]));
      q('[data-slot="status"]').innerHTML = VA.seg('status', STATUS.filter(([k]) => k === 'all' || counts[k]).map(([k, l]) => [k, l, counts[k]]), st.status);
      q('[data-slot="size"]').innerHTML = [['s', 'Small posters', icon('grid-3x3', { size: 15 })], ['m', 'Medium posters', icon('grid-2x2', { size: 15 })], ['l', 'Large posters', icon('square', { size: 15 })]].map(([k, l, ic]) => `<button role="tab" aria-selected="${st.size === k}" data-size="${k}" aria-label="${l}" title="${l}" ${st.view === 'table' ? 'disabled' : ''}>${ic}</button>`).join('');
      q('[data-slot="mode"]').innerHTML = [['grid', 'Posters', icon('image', { size: 15 })], ['table', 'Table', icon('table-2', { size: 15 })]].map(([k, l, ic]) => `<button role="tab" aria-selected="${st.view === k}" data-mode="${k}" aria-label="${l}" title="${l}">${ic}</button>`).join('');
    };
    // A table row already shows the year, so an available Movie says so instead of repeating it.
    const stText = (w, s, tile) => (tile && w.type === 'series' && !s.tone ? `${w.seasons.length} ${w.seasons.length === 1 ? 'season' : 'seasons'} · ${s.text}` : !tile && s.available ? 'Available' : s.text);
    const items = () => all.filter((w) => st.status === 'all' || bucket(w) === st.status).sort((a, b) => SORTS[st.sort][1](a, b) * st.dir);
    const table = (list) => {
      const th = (k, label, cls = '') => `<button class="a-wtable__th ${cls}" data-th="${k}" aria-sort="${st.sort === k ? (st.dir > 0 ? 'descending' : 'ascending') : 'none'}">${label}${st.sort !== k ? '' : st.dir > 0 ? icon('chevron-down', { size: 13 }) : icon('chevron-up', { size: 13 })}</button>`;
      return `<div class="a-wtable a-wtable--${type}" role="table">
        <div class="a-wtable__head" role="row">${th('title', 'Title')}${th('year', 'Year')}<span>Status</span><span>Profile</span><span>Targets</span>${th('size', 'Size', 'is-num')}${th('added', 'Added', 'is-num')}</div>
        ${list.map((w) => {
          const s = VA.statusOf(w);
          return `<a class="a-wtable__row" role="row" href="#work/${w.id}" data-key="${w.id}"><span class="a-wtable__t" role="cell">${VA.posterImg(w, 'a-poster--xs')}<span>${esc(w.title)}</span></span><span class="a-num" role="cell">${w.year}</span><span role="cell" class="a-wtable__st ${s.tone ? 'is-' + s.tone : ''}" data-flip="st">${esc(stText(w, s, false))}</span><span role="cell">${esc(d.profiles.find((p) => p.id === w.profile).name)}</span><span role="cell" class="a-wtable__cov">${VA.covMarks(w)}</span><span role="cell" class="is-num a-num">${sizeOf(w) ? fmt.bytes(sizeOf(w)) : ''}</span><span role="cell" class="is-num a-num">${w.added.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })}</span></a>`;
        }).join('')}
      </div>`;
    };
    const render = () => {
      bar();
      const list = items();
      if (!list.length) body.innerHTML = `<div class="a-empty"><p>No ${type === 'series' ? 'Series' : 'Movies'} are ${esc(STATUS.find(([k]) => k === st.status)[1].toLowerCase())}.</p><p class="a-empty__sub">Choose All to see the whole library.</p></div>`;
      else if (st.view === 'grid') body.innerHTML = `<div class="a-grid a-grid--${st.size}">${list.map(VA.tile).join('')}</div>`;
      else body.innerHTML = table(list);
    };
    render();
    q('[data-sort]').addEventListener('change', (e) => { st.sort = e.target.value; st.dir = 1; render(); });
    return {
      click(e) {
        const t = e.target;
        let b;
        if ((b = t.closest('[data-seg="status"] [data-v]'))) { st.status = b.dataset.v; render(); return true; }
        if ((b = t.closest('[data-size]'))) { st.size = b.dataset.size; render(); return true; }
        if ((b = t.closest('[data-mode]'))) { st.view = b.dataset.mode; render(); return true; }
        if ((b = t.closest('[data-th]'))) { const k = b.dataset.th; if (st.sort === k) st.dir = -st.dir; else { st.sort = k; st.dir = 1; } q('[data-sort]').value = k; render(); return true; }
        return false;
      },
      live() {
        body.querySelectorAll('.a-tile, .a-wtable__row').forEach((t) => {
          const w = d.W[t.dataset.key];
          const s = VA.statusOf(w);
          const m = t.querySelector('[data-flip="st"]');
          const text = stText(w, s, t.classList.contains('a-tile'));
          if (m.textContent !== text) {
            if (m.textContent.split('·')[0] === text.split('·')[0]) m.textContent = text; else VA.flip(m, text);
            m.className = m.className.replace(/ ?is-\w+/g, '') + (s.tone ? ' is-' + s.tone : '');
          }
        });
      },
      refresh(ev) { if (ev === 'job:new' || ev === 'job:done' || ev === 'work:update') { const y = el.scrollTop; render(); el.scrollTop = y; } },
    };
  }

  VIEWS.work = (el, id, api) => {
    const w = d.W[id];
    if (!w) { el.innerHTML = `<div class="a-page"><div class="a-empty"><p>This Work is not in the library.</p><p class="a-empty__sub">Search with ⌘K to find it, or add it from Discover.</p></div></div>`; return {}; }
    const prof = d.profiles.find((p) => p.id === w.profile);
    const live = () => d.jobs.find((j) => j.work === w.id && VA.isLive(j));
    let season = w.type === 'series' ? (w.seasons.findIndex((s) => s.episodes.some((e) => e.status === 'downloading' || e.status === 'missing' || e.status === 'failed')) + 1 || w.seasons.length) : 0;
    const size = sizeOf(w);
    const hist = d.history.filter((h) => h.work === w.id);
    el.innerHTML = `<article class="a-work">
      <div class="a-work__hero">
        <img class="a-work__backdrop" src="${SS.art.backdrop(w, { w: 1600, h: 900 })}" alt="">
        <div class="a-work__heroin">
          <img class="a-poster a-work__poster" src="${SS.art.poster(w)}" alt="${esc(w.title)} poster">
          <div class="a-work__head">
            <h1 class="a-work__title" data-head>${esc(w.title)}</h1>
            <p class="a-work__meta">${VA.workMeta(w)}</p>
            <p class="a-work__overview">${esc(w.overview)}</p>
            <div class="a-work__status" data-wstatus></div>
            <div class="a-work__actions">
              <span class="a-work__monitor"><button class="a-switch" role="switch" aria-checked="${!w.unmonitored}" aria-labelledby="a-mon" data-monitor></button><span id="a-mon">Monitored</span></span>
              ${VA.btn('Search', { kind: 'primary', ic: icon('search', { size: 15 }), attrs: 'data-search' })}
              ${VA.btn('Manual search', { ic: icon('list', { size: 15 }), attrs: 'data-manual aria-expanded="false"' })}
              ${VA.iconBtn(icon('ellipsis', { size: 18 }), 'More actions', `data-pop="more" data-work="${w.id}" data-align="left"`)}
            </div>
          </div>
        </div>
      </div>
      <div class="a-work__manual" data-manual-panel hidden></div>
      <div class="a-work__body">
        <div class="a-work__main">
          <section><div class="a-section-head"><h3 class="a-h2">Path</h3></div><div data-path>${VA.pathStrip(w)}</div></section>
          <section><div class="a-section-head"><h3 class="a-h2">Targets <span class="a-count">${esc(prof.name)} Profile</span></h3></div><div data-cov>${VA.coverage(w)}</div></section>
          ${w.type === 'series' ? `<section><div class="a-section-head"><h3 class="a-h2">Seasons</h3><div data-slot="season"></div></div><div data-eps></div></section>` : ''}
          <section><div class="a-section-head"><h3 class="a-h2">Files <span class="a-count">${w.files?.length || ''}</span></h3></div><div data-files>${w.type === 'series' ? seriesFiles(w) : VA.files(w)}</div></section>
        </div>
        <aside class="a-work__side">
          <dl class="a-side">
            <div><dt>Profile</dt><dd>${esc(prof.name)}<small>${esc(fmt.list(prof.targets.map(VA.targetName)))}</small></dd></div>
            <div><dt>Editions</dt><dd>${esc(fmt.list(w.editions))}</dd></div>
            <div><dt>Added</dt><dd>${w.added.toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric' })}${w.addedBy ? `<small>Requested by ${esc(d.A[w.addedBy].name)}</small>` : ''}</dd></div>
            <div><dt>Size on disk</dt><dd class="a-num">${size ? fmt.bytes(size) : 'Nothing yet'}</dd></div>
            <div><dt>Media Server</dt><dd>Plex on vault</dd></div>
            <div><dt>Folder</dt><dd class="a-mono">/mnt/media/${w.type === 'series' ? 'Series' : 'Movies'}/${esc(w.title)} (${w.year})</dd></div>
          </dl>
          <div class="a-section-head"><h3 class="a-h2">History</h3></div>
          <ol class="a-hist">${hist.length ? hist.map((h) => `<li><span class="a-hist__t">${esc(h.text)}</span><span class="a-hist__w a-num">${esc(fmt.ago(h.at))} · ${esc(h.node)}</span></li>`).join('') : `<li class="a-hist__none">Nothing has happened to this Work since it was added.</li>`}</ol>
        </aside>
      </div>
    </article>`;
    const q = (s) => el.querySelector(s);
    const status = () => {
      const j = live();
      const box = q('[data-wstatus]');
      if (!j) {
        const s = VA.statusOf(w);
        box.innerHTML = w.type === 'series' ? `<span>${esc(`${w.counts.available} of ${w.counts.total} episodes`)}${w.nextAir ? ` · next ${esc(fmt.until(w.nextAir))}` : ''}</span>` : `<span class="${s.tone === 'sep' ? 'is-sep' : ''}">${esc(w.status === 'available' ? `In the library in ${fmt.list(d.targets.filter((t) => w.coverage[t.id] === 'met').map((t) => t.name))}` : s.text)}</span>`;
        return;
      }
      const html = j.state === 'failed'
        ? `<span class="is-sep">${VA.vortex(16)}${esc(VA.jobLine(j))}: ${esc(j.message)}</span>`
        : `<span>${j.episode ? `${esc(fmt.ep(j.episode))} · ` : ''}${esc(VA.jobLine(j))} on ${esc(j.node)}</span><span class="a-num" data-l="pct">${fmt.pct(j.progress)}</span>${VA.jobRate(j) ? `<span class="a-num" data-l="rate">${esc(VA.jobRate(j))}</span>` : ''}${j.eta ? `<span class="a-num" data-l="eta">${esc(VA.jobEta(j))} left</span>` : ''}${VA.fil('job:' + j.id, { y: 0.55, pad: 4 })}`;
      if (box.dataset.job !== j.id + j.state) { box.dataset.job = j.id + j.state; box.innerHTML = html; }
      else {
        const set = (k, v) => { const x = box.querySelector(`[data-l="${k}"]`); if (x) x.textContent = v; };
        set('pct', fmt.pct(j.progress)); set('rate', VA.jobRate(j)); set('eta', `${VA.jobEta(j)} left`);
      }
    };
    const eps = () => {
      if (w.type !== 'series') return;
      q('[data-slot="season"]').innerHTML = VA.seg('season', w.seasons.map((s) => [String(s.n), `Season ${s.n}`]), String(season));
      q('[data-eps]').innerHTML = VA.episodes(w, season);
    };
    status();
    eps();
    const manual = () => {
      const p = q('[data-manual-panel]');
      const rels = releasesFor(w);
      p.innerHTML = `<div class="a-manual">
        <div class="a-section-head"><h3 class="a-h2">Manual search <span class="a-count">${rels.length} releases from 4 indexers</span></h3>${VA.iconBtn(icon('x', { size: 16 }), 'Close manual search', 'data-manual')}</div>
        <div class="a-table a-rels" role="table">
          <div class="a-table__head" role="row"><span>Release</span><span>Meets</span><span class="is-num">Size</span><span class="is-num">Seeds</span><span>Indexer</span><span>Age</span><span></span></div>
          ${rels.map((r, i) => `<div class="a-table__row ${r.reject ? 'is-off' : ''}" role="row"><span class="a-mono a-rels__name" title="${esc(r.name)}">${esc(r.name)}</span><span>${r.target ? esc(VA.targetName(r.target)) : `<small>${esc(r.reject)}</small>`}</span><span class="is-num a-num">${fmt.bytes(r.size)}</span><span class="is-num a-num">${r.seeds}</span><span>${esc(r.indexer)}</span><span>${esc(r.age)}</span><span>${r.reject ? '' : VA.btn('Grab', { sm: true, attrs: `data-grab="${i}"` })}</span></div>`).join('')}
        </div></div>`;
    };
    const resync = () => {
      status();
      q('[data-path]').innerHTML = VA.pathStrip(w);
      q('[data-cov]').innerHTML = VA.coverage(w);
      eps();
    };
    return {
      click(e) {
        const t = e.target;
        let b;
        if ((b = t.closest('[data-monitor]'))) {
          w.unmonitored = !w.unmonitored;
          b.setAttribute('aria-checked', String(!w.unmonitored));
          api.toast(w.unmonitored ? `${w.title} is no longer monitored. Nothing new is searched for it.` : `${w.title} is monitored again.`);
          return true;
        }
        if ((b = t.closest('[data-search]'))) {
          const j = live();
          if (j && j.state !== 'failed') { api.toast(`${w.title} is already ${VA.jobVerb(j).toLowerCase()} on ${j.node}.`); return true; }
          if (j?.state === 'failed') { VA.act.blocklist(j); api.toast('Blocklisted the sample. Searching Kestrel for another release.'); return true; }
          if (w.type === 'series') { const n = VA.act.searchMissingEpisodes(); api.toast(n ? `Searching for ${fmt.plural(n, 'missing episode')}.` : `Every aired episode of ${w.title} is in the library.`); return true; }
          if (w.status === 'available') { api.toast(`${w.title} meets its Targets. Upgrades are searched automatically.`); return true; }
          b.classList.add('is-loading');
          setTimeout(() => { VA.act.search(w); api.toast(`Searching 4 indexers for ${w.title}.`); }, 400);
          return true;
        }
        if ((b = t.closest('[data-manual]'))) {
          const p = q('[data-manual-panel]');
          p.hidden = !p.hidden;
          el.querySelectorAll('[data-manual][aria-expanded]').forEach((x) => x.setAttribute('aria-expanded', String(!p.hidden)));
          if (!p.hidden) { manual(); p.scrollIntoView({ block: 'nearest', behavior: VA.reduced() ? 'auto' : 'smooth' }); }
          return true;
        }
        if ((b = t.closest('[data-grab]'))) {
          const r = releasesFor(w)[+b.dataset.grab];
          if (d.jobs.some((j) => j.work === w.id && j.target === r.target && VA.isLive(j) && j.state !== 'failed')) { api.toast(`${VA.targetName(r.target)} is already on its way.`); return true; }
          const n = { id: 'vg' + Date.now(), kind: 'download', work: w.id, label: w.title, target: r.target, release: r.name, node: 'ams-1', size: r.size, progress: 0, rate: 24e6, seeds: r.seeds, peers: Math.round(r.seeds / 6), state: 'running', indexer: r.indexer };
          if (w.status === 'missing') w.status = 'downloading';
          d.jobs.push(n);
          SS.sim.emit('job:new', n);
          b.disabled = true;
          b.querySelector('span').textContent = 'Grabbed';
          api.toast(`Grabbed from ${r.indexer}. Downloading on ams-1.`);
          return true;
        }
        if ((b = t.closest('[data-seg="season"] [data-v]'))) { season = +b.dataset.v; eps(); return true; }
        if ((b = t.closest('[data-ep-search]'))) {
          const [s, n] = b.dataset.epSearch.split('.').map(Number);
          w.seasons[s - 1].episodes[n - 1].status = 'missing';
          VA.act.search(w, { episode: [s, n] });
          api.toast(`Searching for ${w.title} ${fmt.ep([s, n])}.`);
          return true;
        }
        return false;
      },
      live() {
        status();
        const j = live();
        if (j) {
          q('[data-path]').innerHTML = VA.pathStrip(w);
          const c = el.querySelector(`[data-live="cov-${j.target}"]`);
          if (c) c.textContent = fmt.pct(j.progress);
          if (j.episode) { const x = el.querySelector(`[data-live="ep-${j.episode[1]}"]`); if (x) x.textContent = fmt.pct(j.progress); }
        }
      },
      refresh(ev, p) { if (p?.work === w.id || ev === 'work:update') resync(); },
    };
  };
  function seriesFiles(w) {
    const recent = w.seasons[w.seasons.length - 1].episodes.filter((e) => e.status === 'available').slice(-4).reverse();
    if (!recent.length) return VA.files({ ...w, files: [] });
    const base = w.title.replace(/[^A-Za-z0-9 ]/g, '').replace(/ +/g, '.');
    const sn = w.seasons.length;
    return VA.files({ ...w, files: recent.map((e) => ({ target: 'hd', size: 2.4e9 + (e.n % 3) * 0.6e9, name: `${base}.S${String(sn).padStart(2, '0')}E${String(e.n).padStart(2, '0')}.${e.title.replace(/ /g, '.')}.1080p.WEB-DL.DDP5.1.H.264-NTb`, video: 'AVC · SDR', audio: 'DDP 5.1', resolution: '1920×1080', node: 'vault' })) });
  }

  VIEWS.activity = (el, openArg, api) => {
    const st = VA.actState;
    let open = openArg || null;
    el.innerHTML = `<div class="a-page a-act">
      ${head('Activity')}
      <div class="a-act__bar"><div data-slot="filter"></div><label class="a-lib__sort"><span>Node</span><select class="a-select" data-node-filter><option value="all">All Nodes</option>${d.nodes.map((n) => `<option value="${n.id}" ${st.node === n.id ? 'selected' : ''}>${n.name}</option>`).join('')}</select></label></div>
      <div class="a-job a-job--head" aria-hidden="true"><div class="a-job__main"><span></span><span>Work</span><span>Release</span><span>Node</span><span>Progress</span><span></span><span class="is-num">Rate</span><span class="is-num">Left</span></div><span class="a-job__actions"></span></div>
      <section data-group="needs"><div class="a-section-head"><h3 class="a-h2">Needs you <span class="a-count" data-n></span></h3></div><div class="a-jobs"></div></section>
      <section data-group="watch"><div class="a-section-head"><h3 class="a-h2">Watching <span class="a-count" data-n></span></h3></div><div class="a-jobs"></div></section>
      <section data-group="running"><div class="a-section-head"><h3 class="a-h2">Moving <span class="a-count" data-n></span></h3></div><div class="a-jobs"></div></section>
      <section data-group="waiting"><div class="a-section-head"><h3 class="a-h2">Waiting <span class="a-count" data-n></span></h3></div><div class="a-jobs"></div></section>
      <section class="a-act__done"><div class="a-section-head"><h3 class="a-h2">Finished tonight</h3></div><ol class="a-hist a-hist--wide" data-hist></ol></section>
    </div>`;
    const q = (s) => el.querySelector(s);
    const GROUPS = ['needs', 'watch', 'running', 'waiting'];
    const render = () => {
      const jobs = d.jobs.filter((j) => VA.isLive(j) && (st.node === 'all' || j.node === st.node));
      const n = { needs: 0, watch: 0, running: 0, waiting: 0 };
      jobs.forEach((j) => n[VA.jobGroup(j)]++);
      q('[data-slot="filter"]').innerHTML = VA.seg('filter', [['all', 'All', jobs.length], ['needs', 'Needs you', n.needs], ['watch', 'Watching', n.watch], ['running', 'Moving', n.running], ['waiting', 'Waiting', n.waiting]], st.filter);
      for (const g of GROUPS) {
        const sec = q(`[data-group="${g}"]`);
        const list = jobs.filter((j) => VA.jobGroup(j) === g);
        sec.hidden = !list.length || (st.filter !== 'all' && st.filter !== g);
        sec.querySelector('[data-n]').textContent = list.length;
        VA.patch(sec.querySelector('.a-jobs'), list, (j) => j.id, (j) => VA.jobRow(j, j.id === open));
      }
      const anyShown = GROUPS.some((g) => !q(`[data-group="${g}"]`).hidden);
      VA.emptyState(q('.a-act'), !anyShown, 'Nothing is moving on this Node.', 'Pick All Nodes, or start a search from a Work to set something in motion.');
      const since = +SS.sim.clock - 12 * 3600e3;
      q('[data-hist]').innerHTML = d.history.filter((h) => h.at >= since).map((h) => `<li><span class="a-hist__g">${h.kind === 'failed' ? VA.vortex(14) : VA.laminar(14)}</span><span class="a-hist__t"><b>${esc(d.W[h.work].title)}${h.episode ? ' · ' + fmt.ep(h.episode) : ''}</b> ${esc(h.text)}</span><span class="a-hist__w a-num">${esc(fmt.time(h.at))} · ${esc(h.node)}</span></li>`).join('');
    };
    render();
    if (open) requestAnimationFrame(() => el.querySelector(`.a-job[data-key="${CSS.escape(open)}"]`)?.scrollIntoView({ block: 'center' }));
    el.querySelector('[data-node-filter]').addEventListener('change', (e) => { st.node = e.target.value; render(); });
    return {
      click(e) {
        let b;
        if ((b = e.target.closest('[data-seg="filter"] [data-v]'))) { st.filter = b.dataset.v; render(); return true; }
        if ((b = e.target.closest('[data-more]'))) { open = b.dataset.more; api.setView('activity/' + open); render(); return true; }
        if ((b = e.target.closest('[data-job]'))) { open = open === b.dataset.job ? null : b.dataset.job; api.setView('activity' + (open ? '/' + open : '')); render(); return true; }
        return false;
      },
      live() { VA.liveJobs(el); },
      refresh() { render(); },
    };
  };

  VIEWS.nodes = (el, focus, api) => {
    const meter = (k, label, v, extra = '') => `<div class="a-meter"><span class="a-meter__k">${label}</span><span class="a-meter__v a-num" data-live="${k}">${v}</span>${extra}<span class="a-meter__bar" data-meter="${k}"><i></i></span></div>`;
    el.innerHTML = `<div class="a-page a-nodes">
      ${head('Nodes', `${d.nodes.length} Nodes under one Controller. atlas holds the library and serves this page.`)}
      ${d.nodes.map((n) => `<section class="a-node a-node--${n.state} ${focus === n.id ? 'is-focus' : ''}" data-node="${n.id}" data-key="${n.id}" id="a-node-${n.id}">
        <div class="a-node__body">${VA.nodeBody(n, { w: 240, h: 74 })}</div>
        <div class="a-node__id">
          <h3>${esc(n.name)}${n.controller ? '<span class="a-tag">Controller</span>' : '<span class="a-tag">Worker</span>'}</h3>
          <p>${esc(n.hardware)}</p>
          <p class="a-node__sub">${esc(n.os)} · ${esc(n.place)}</p>
          <p class="a-node__sub a-mono">${esc(n.address)}</p>
          <p class="a-node__state ${n.state === 'offline' ? 'is-sep' : ''}" data-live="state">${esc(VA.nodeState(n))}</p>
          ${n.stateNote ? `<p class="a-node__note">${esc(n.stateNote)}</p>` : ''}
        </div>
        <div class="a-node__roles"><span class="a-node__k">Roles</span><div>${n.roles.map((r) => `<span class="a-chip">${esc(r)}</span>`).join('')}</div></div>
        <div class="a-node__load">
          ${n.state === 'offline' ? `<p class="a-node__off">No load reported since ${fmt.time(n.lastSeen)}.</p>` : `${meter('cpu', 'CPU', fmt.pct(n.cpu))}${meter('mem', 'Memory', fmt.pct(n.mem))}${n.gpu != null ? meter('gpu', 'GPU', fmt.pct(n.gpu)) : ''}<p class="a-node__rates a-num" data-live="rates">${esc(VA.nodeRates(n))}</p>`}
        </div>
        <div class="a-node__disks">${n.disks.map((k) => `<div class="a-meter"><span class="a-meter__k ${k.path.startsWith('/') ? 'a-mono' : ''}">${esc(k.path)}</span><span class="a-meter__v a-num">${fmt.bytes(k.used)} of ${fmt.bytes(k.total)}</span><span class="a-meter__bar" style="--a-p:${(k.used / k.total).toFixed(3)}"><i></i></span></div>`).join('')}
          <p class="a-node__sub a-num">${n.state === 'offline' ? 'Offline' : `Up ${fmt.dur(n.uptime / 1000)}`} · ${esc(n.version)}${n.temp ? ` · ${n.temp} °C` : ''}</p></div>
      </section>`).join('')}
    </div>`;
    const set = () => el.querySelectorAll('[data-meter]').forEach((m) => { const n = d.N[m.closest('[data-node]').dataset.node]; m.style.setProperty('--a-p', String(n[m.dataset.meter] || 0)); });
    set();
    if (focus) requestAnimationFrame(() => el.querySelector('#a-node-' + CSS.escape(focus))?.scrollIntoView({ block: 'center' }));
    return { live() { VA.liveNodes(el); }, title: 'Nodes' };
  };

  VIEWS.requests = (el, _, api) => {
    const st = VA.reqState;
    el.innerHTML = `<div class="a-page a-reqs">${head('Requests')}<div class="a-act__bar"><div data-slot="filter"></div></div><div class="a-list" data-list></div></div>`;
    const q = (s) => el.querySelector(s);
    const groups = { pending: ['pending'], approved: ['approved', 'downloading'], available: ['available'], denied: ['denied'], all: ['pending', 'approved', 'downloading', 'available', 'denied'] };
    const render = () => {
      const n = (g) => d.requests.filter((r) => groups[g].includes(r.state)).length;
      q('[data-slot="filter"]').innerHTML = VA.seg('rfilter', [['pending', 'Pending', n('pending')], ['approved', 'Approved', n('approved')], ['available', 'Available', n('available')], ['denied', 'Denied', n('denied')], ['all', 'All', n('all')]], st.filter);
      const list = d.requests.filter((r) => groups[st.filter].includes(r.state)).sort((a, b) => b.at - a.at);
      VA.patch(q('[data-list]'), list, (r) => r.id, VA.reqRow);
      VA.emptyState(q('[data-list]'), !list.length, st.filter === 'pending' ? 'No requests are waiting.' : 'No requests here.', st.filter === 'pending' ? 'When a Member asks for a Work it lands here for you to approve or deny.' : 'Requests move between these lists as they are approved and arrive.');
    };
    render();
    return {
      click(e) { const b = e.target.closest('[data-seg="rfilter"] [data-v]'); if (b) { st.filter = b.dataset.v; render(); return true; } return false; },
      refresh() { render(); },
      live() { el.querySelectorAll('.a-req [data-live="sub"]').forEach((s) => { const r = d.requests.find((x) => x.id === s.closest('.a-req').dataset.key); const t = VA.reqStatus(r); if (s.textContent !== t) s.textContent = t; }); },
    };
  };

  VIEWS.discover = (el) => {
    const soon = d.calendar.slice().sort((a, b) => a.at - b.at);
    const unreleased = d.works.filter((w) => w.status === 'unreleased');
    el.innerHTML = `<div class="a-page a-disc">
      ${head('Discover')}
      <section><div class="a-section-head"><h3 class="a-h2">Coming up</h3></div>
        <ol class="a-cal">${soon.map((c) => { const w = d.W[c.work]; return `<li><a class="a-cal__row" href="#work/${w.id}"><span class="a-cal__when a-num">${esc(fmt.until(c.at))}</span>${VA.posterImg(w, 'a-poster--xs')}<span class="a-cal__t"><b>${esc(w.title)}</b><span>${c.episode ? `${fmt.ep(c.episode)} · ${esc(c.title)}` : esc(c.title)}</span></span><span class="a-cal__k">${c.kind === 'airs' ? 'Airs' : 'Releases'}</span></a></li>`; }).join('')}</ol>
      </section>
      <section><div class="a-section-head"><h3 class="a-h2">Not yet released</h3></div><div class="a-grid a-grid--s">${unreleased.map(VA.tile).join('')}</div></section>
    </div>`;
    return {};
  };

  VIEWS.settings = (el) => {
    el.innerHTML = `<div class="a-page a-set">
      ${head('Settings')}
      <section><div class="a-section-head"><h3 class="a-h2">Targets</h3></div>
        <div class="a-table a-set__t" role="table"><div class="a-table__head" role="row"><span>Target</span><span>Sourced</span><span>Rules</span></div>${d.targets.map((t) => `<div class="a-table__row" role="row"><span><b>${esc(t.name)}</b></span><span>${t.kind === 'derived' ? `Derived from ${esc(VA.targetName(t.from))}` : 'Downloaded'}</span><span class="a-ink3">${esc(t.note)}</span></div>`).join('')}</div></section>
      <section><div class="a-section-head"><h3 class="a-h2">Profiles</h3></div>
        <div class="a-table a-set__p" role="table"><div class="a-table__head" role="row"><span>Profile</span><span>Targets</span><span class="is-num">Works</span></div>${d.profiles.map((p) => `<div class="a-table__row" role="row"><span><b>${esc(p.name)}</b></span><span>${esc(fmt.list(p.targets.map(VA.targetName)))}</span><span class="is-num a-num">${d.works.filter((w) => w.profile === p.id).length}</span></div>`).join('')}</div></section>
      <section><div class="a-section-head"><h3 class="a-h2">Accounts</h3></div>
        <div class="a-table a-set__a" role="table"><div class="a-table__head" role="row"><span>Account</span><span>Email</span><span>Permission</span><span>Since</span></div>${d.accounts.map((a) => `<div class="a-table__row" role="row"><span class="a-set__who">${VA.avatar(a.id, 22)}<b>${esc(a.name)}</b></span><span class="a-ink3">${esc(a.email)}</span><span>${a.admin ? 'Admin' : 'Member'}</span><span class="a-num a-ink3">${a.since.toLocaleDateString('en-GB', { month: 'short', year: 'numeric' })}</span></div>`).join('')}</div></section>
    </div>`;
    return {};
  };

  VIEWS.system = (el) => {
    el.innerHTML = `<div class="a-page a-sys">
      ${head('System')}
      <section class="a-sys__update"><div><h3 class="a-h2">SlipStream 2.3.1 is ready</h3><p>atlas updates first, then each Worker in turn. Jobs pause for about a minute on each Node.</p></div>${VA.btn('Review update', { kind: 'primary', attrs: 'data-pop-cmd="msg" data-msg="Release notes open in the full build."' })}</section>
      <section><div class="a-section-head"><h3 class="a-h2">Health</h3></div><div class="a-list">${d.alerts.map((a) => `<div class="a-need a-need--health"><span class="a-need__glyph">${a.level === 'error' ? VA.vortex(18) : a.level === 'warning' ? VA.watchGlyph(a.node ? 'behind' : 'stall', 18) : VA.laminar(18)}</span><span class="a-need__text"><span class="a-need__title">${esc(a.title)}</span><span class="a-need__detail">${esc(a.detail)}</span></span><span class="a-need__time a-num">${esc(fmt.ago(a.at))}</span><span></span></div>`).join('')}</div></section>
      <section><div class="a-section-head"><h3 class="a-h2">Indexers</h3></div><div class="a-table a-set__i" role="table"><div class="a-table__head" role="row"><span>Indexer</span><span>Kind</span><span>State</span></div>${d.indexers.map((i) => `<div class="a-table__row" role="row"><span><b>${esc(i.name)}</b></span><span class="a-ink3">${esc(i.kind)}</span><span>${i.state === 'ok' ? 'Healthy' : esc(i.note)}</span></div>`).join('')}</div></section>
      <section><div class="a-section-head"><h3 class="a-h2">Media Servers</h3></div><div class="a-table a-set__i" role="table"><div class="a-table__head" role="row"><span>Server</span><span>Node</span><span>State</span></div>${d.mediaServers.map((m) => `<div class="a-table__row" role="row"><span><b>${esc(m.name)}</b></span><span class="a-ink3">${esc(m.node)}</span><span>Online · ${m.streams} streams</span></div>`).join('')}</div></section>
      <p class="a-sys__foot a-num">SlipStream 2.3.0 · Controller atlas · up ${fmt.dur(d.N.atlas.uptime / 1000)}</p>
    </div>`;
    return {};
  };
})();
