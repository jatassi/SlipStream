(function () {
  const SS = window.SS;
  const d = SS.data;
  const fmt = SS.fmt;
  const VA = window.VA;
  const esc = fmt.esc;
  const icon = VA.icon;

  const TABS = [['home', 'Flow'], ['library', 'Library'], ['activity', 'Activity'], ['search', 'Search'], ['more', 'More']];
  const tabIcon = (id) => ({ home: icon('wind', { size: 24 }), library: icon('library', { size: 24 }), activity: VA.flowlines(24), search: icon('search', { size: 24 }), more: icon('ellipsis', { size: 24 }) }[id]);
  VA.phLib = VA.phLib || { type: 'movie', status: 'all' };

  const chevron = () => icon('chevron-right', { size: 16 });

  function screen({ title, large = true, back = '', right = '', hero = '', body = '' }) {
    const el = document.createElement('section');
    el.className = `a-scr ${hero ? 'a-scr--hero' : ''}`;
    el.innerHTML = `
      <header class="a-nav"><div class="a-nav__bg"></div><div class="a-nav__row">
        ${back ? `<button class="a-nav__back" data-back aria-label="Back to ${esc(back)}">${icon('chevron-left', { size: 22 })}<span>${esc(back)}</span></button>` : '<span></span>'}
        <span class="a-nav__title">${esc(title)}</span>
        <span class="a-nav__right">${right}</span>
      </div></header>
      <div class="a-scr__scroll" data-scroll>
        ${hero}
        ${large ? `<h1 class="a-large">${esc(title)}</h1>` : ''}
        <div class="a-scr__body">${body}</div>
      </div>
      <div class="a-scr__dim"></div>`;
    const sc = el.querySelector('[data-scroll]');
    const onScroll = () => {
      const lt = el.querySelector('.a-large');
      const limit = lt ? lt.offsetTop + lt.offsetHeight - (hero ? 96 : 62) : 8;
      el.classList.toggle('is-collapsed', sc.scrollTop > limit);
      el.classList.toggle('is-scrolled', sc.scrollTop > 2);
    };
    sc.addEventListener('scroll', onScroll, { passive: true });
    requestAnimationFrame(onScroll);
    return el;
  }

  VA.mountPhone = (ctx) => {
    const root = ctx.root;
    root.classList.add('a-app', 'a-phone');
    root.innerHTML = `<div class="a-stack" data-stack></div>
      <nav class="a-tabbar" aria-label="Tabs">${TABS.map(([id, label]) => `<a class="a-tab" href="#${id}" data-tab="${id}">${tabIcon(id)}${id === 'activity' ? '<span class="a-tab__badge a-num" data-badge></span>' : ''}<span>${label}</span></a>`).join('')}</nav>
      <div class="a-sheetlayer" data-sheetlayer></div>`;
    const stackEl = root.querySelector('[data-stack]');
    const sheetLayer = root.querySelector('[data-sheetlayer]');
    let tab = 'home';
    const stack = [];
    const api = { root, ctx, toast: (t) => VA.toast(root, t), go: (v) => go(v) };

    const scale = () => root.getBoundingClientRect().width / root.offsetWidth || 1;
    const top = () => stack[stack.length - 1];
    const dimOf = (s) => s.el.querySelector('.a-scr__dim');

    function build(view) {
      const [name, arg] = view.split('/');
      const make = SCREENS[name] || SCREENS.home;
      const s = make(arg, api, stack.length ? titleOf(stack[stack.length - 1].view) : '');
      s.view = view;
      return s;
    }
    function setTab(t, view) {
      stack.forEach((s) => { s.destroy?.(); s.el.remove(); });
      stack.length = 0;
      tab = t;
      const s = build(view || t);
      stackEl.appendChild(s.el);
      stack.push(s);
      root.querySelectorAll('[data-tab]').forEach((a) => a.setAttribute('aria-current', a.dataset.tab === t ? 'page' : 'false'));
    }
    const EASE = 'cubic-bezier(0.32, 0.72, 0, 1)';
    function push(view, animate = true) {
      const under = top();
      const s = build(view);
      s.el.classList.add('a-scr--pushed');
      stackEl.appendChild(s.el);
      stack.push(s);
      under.el.classList.add('is-under');
      if (animate && !VA.reduced()) {
        s.el.animate([{ transform: 'translateX(100%)' }, { transform: 'translateX(0)' }], { duration: 440, easing: EASE });
        under.el.animate([{ transform: 'translateX(0)' }, { transform: 'translateX(-28%)' }], { duration: 440, easing: EASE });
        dimOf(under).animate([{ opacity: 0 }, { opacity: 1 }], { duration: 440, easing: EASE });
      } else if (animate) s.el.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 200 });
      ctx.setView(view);
      return s;
    }
    function pop({ from = 0, velocity = 0, animate = true } = {}) {
      if (stack.length < 2) return;
      const s = stack.pop();
      const under = top();
      const W = root.offsetWidth;
      const finish = () => {
        s.destroy?.();
        s.el.remove();
        under.el.classList.remove('is-under');
        under.el.style.transform = '';
        dimOf(under).style.opacity = '';
        ctx.setView(under.view);
      };
      if (!animate) return finish();
      if (VA.reduced()) { s.el.animate([{ opacity: 1 }, { opacity: 0 }], { duration: 180, fill: 'forwards' }).finished.then(finish); return; }
      const dist = W - from;
      const sp = VA.spring({ response: 0.36, v0: dist > 1 ? velocity / dist : 0 });
      const opts = { duration: velocity ? sp.duration : 340, easing: velocity ? sp.easing : EASE, fill: 'forwards' };
      const k0 = from / W;
      s.el.style.transform = '';
      under.el.classList.remove('is-under');
      under.el.style.transform = '';
      s.el.animate([{ transform: `translateX(${from}px)` }, { transform: `translateX(${W}px)` }], opts).finished.then(finish);
      under.el.animate([{ transform: `translateX(${-28 * (1 - k0)}%)` }, { transform: 'translateX(0)' }], { ...opts, fill: 'none' });
      dimOf(under).style.opacity = '0';
      dimOf(under).animate([{ opacity: 1 - k0 }, { opacity: 0 }], { ...opts, fill: 'none' });
    }

    function go(v, first = false) {
      v = v || 'home';
      const [name, id, extra] = v.split('/');
      closeSheet(false);
      if (TABS.some(([t]) => t === name)) { setTab(name); ctx.setView(v); return; }
      if (name === 'series') { VA.phLib.type = 'series'; setTab('library'); ctx.setView(v); return; }
      if (first || !stack.length) setTab(name === 'work' ? 'library' : 'more');
      push(name === 'work' ? `work/${id}` : name, !first);
      if (name === 'work' && extra === 'actions') openSheet(d.W[id], !first);
    }

    // Edge swipe back: tracks the finger 1:1, projects the release velocity, settles on a spring.
    let drag = null;
    ctx.listen(root, 'pointerdown', (e) => {
      if (stack.length < 2 || sheetLayer.firstChild) return;
      const r = root.getBoundingClientRect();
      if ((e.clientX - r.left) / scale() > 24) return;
      if (e.pointerType === 'mouse') e.preventDefault();
      drag = { id: e.pointerId, x0: e.clientX, y0: e.clientY, on: false, hist: [], x: 0 };
    });
    ctx.listen(root, 'dragstart', (e) => e.preventDefault());
    ctx.listen(root, 'pointermove', (e) => {
      if (!drag || e.pointerId !== drag.id) return;
      const dx = (e.clientX - drag.x0) / scale();
      const dy = (e.clientY - drag.y0) / scale();
      if (!drag.on) {
        if (Math.abs(dy) > 10 && Math.abs(dy) > Math.abs(dx)) { drag = null; return; }
        if (dx < 8) return;
        drag.on = true;
        root.setPointerCapture(e.pointerId);
      }
      const W = root.offsetWidth;
      const x = Math.max(0, dx);
      drag.x = x;
      drag.hist.push([performance.now(), x]);
      if (drag.hist.length > 6) drag.hist.shift();
      const s = top(), under = stack[stack.length - 2];
      s.el.style.transform = `translateX(${x}px)`;
      under.el.style.transform = `translateX(${-28 * (1 - x / W)}%)`;
      dimOf(under).style.opacity = String(1 - x / W);
    });
    const endDrag = (e) => {
      if (!drag || e.pointerId !== drag.id) return;
      const g = drag;
      drag = null;
      if (!g.on) return;
      const s = top(), under = stack[stack.length - 2];
      const W = root.offsetWidth;
      const [t0, x0] = g.hist[0], [t1, x1] = g.hist[g.hist.length - 1];
      const v = t1 > t0 ? ((x1 - x0) / (t1 - t0)) * 1000 : 0;
      if (g.x + VA.project(v, 0.994) > W / 2) return pop({ from: g.x, velocity: Math.max(v, 1) });
      const sp = VA.spring({ response: 0.34, v0: g.x > 1 ? -v / -g.x : 0 });
      const opts = { duration: sp.duration, easing: sp.easing };
      const k0 = g.x / W;
      s.el.style.transform = '';
      under.el.style.transform = '';
      dimOf(under).style.opacity = '';
      s.el.animate([{ transform: `translateX(${g.x}px)` }, { transform: 'translateX(0)' }], opts);
      under.el.animate([{ transform: `translateX(${-28 * (1 - k0)}%)` }, { transform: 'translateX(-28%)' }], opts);
      dimOf(under).animate([{ opacity: 1 - k0 }, { opacity: 1 }], opts);
    };
    ctx.listen(root, 'pointerup', endDrag);
    ctx.listen(root, 'pointercancel', endDrag);

    function openSheet(w, animate = true) {
      const live = d.jobs.find((j) => j.work === w.id && VA.isLive(j));
      sheetLayer.innerHTML = `<div class="a-sheet__scrim" data-sheet-close></div>
        <div class="a-sheet" role="dialog" aria-label="Actions for ${esc(w.title)}">
          <div class="a-sheet__grab" aria-hidden="true"></div>
          <div class="a-sheet__head">${VA.posterImg(w, 'a-poster--sm')}<span><b>${esc(w.title)}</b><span>${esc(live ? VA.jobLine(live) : VA.statusOf(w).text)}</span></span></div>
          <div class="a-sheet__group">
            <button class="a-sheet__item" data-sheet-do="search">${icon('search', { size: 20 })}Search for a release</button>
            <button class="a-sheet__item" data-sheet-do="manual">${icon('list', { size: 20 })}Manual search</button>
            <button class="a-sheet__item" data-sheet-do="monitor">${icon('eye', { size: 20 })}${w.unmonitored ? 'Monitor' : 'Stop monitoring'}</button>
            <button class="a-sheet__item" data-sheet-do="refresh">${icon('refresh-cw', { size: 20 })}Refresh metadata</button>
            <button class="a-sheet__item" data-sheet-do="profile">${icon('sliders-horizontal', { size: 20 })}Change Profile</button>
          </div>
          <div class="a-sheet__group"><button class="a-sheet__item a-sheet__item--quiet" data-sheet-do="delete">${icon('trash-2', { size: 20 })}Delete Work and files</button></div>
          <button class="a-sheet__cancel" data-sheet-close>Cancel</button>
        </div>`;
      const sh = sheetLayer.querySelector('.a-sheet');
      const scrim = sheetLayer.querySelector('.a-sheet__scrim');
      sheetLayer.dataset.work = w.id;
      if (animate) {
        sh.animate([{ transform: 'translateY(100%)' }, { transform: 'translateY(0)' }], { duration: VA.reduced() ? 1 : 460, easing: EASE });
        scrim.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 300, easing: 'ease-out' });
      }
      ctx.setView(`work/${w.id}/actions`);
      let g = null;
      sh.addEventListener('pointerdown', (e) => { g = { id: e.pointerId, y0: e.clientY, on: false, hist: [], y: 0 }; });
      sh.addEventListener('pointermove', (e) => {
        if (!g || e.pointerId !== g.id) return;
        const dy = (e.clientY - g.y0) / scale();
        if (!g.on) { if (Math.abs(dy) < 6) return; g.on = true; sh.setPointerCapture(e.pointerId); }
        const h = sh.offsetHeight;
        const y = dy < 0 ? -((-dy * h * 0.55) / (h + 0.55 * -dy)) * 0.35 : dy;
        g.y = y;
        g.hist.push([performance.now(), dy]);
        if (g.hist.length > 6) g.hist.shift();
        sh.style.transform = `translateY(${y}px)`;
        scrim.style.opacity = String(Math.max(0, 1 - Math.max(0, y) / h));
      });
      const up = (e) => {
        if (!g || e.pointerId !== g.id) return;
        const s = g;
        g = null;
        if (!s.on) return;
        sh.addEventListener('click', (ev) => { ev.stopPropagation(); ev.preventDefault(); }, { capture: true, once: true });
        const h = sh.offsetHeight;
        const [t0, y0] = s.hist[0], [t1, y1] = s.hist[s.hist.length - 1];
        const v = t1 > t0 ? ((y1 - y0) / (t1 - t0)) * 1000 : 0;
        if (s.y > 0 && s.y + VA.project(v, 0.994) > h * 0.45) return closeSheet(true, s.y, Math.max(v, 1));
        const sp = VA.spring({ response: 0.32, v0: s.y ? -v / -s.y : 0 });
        sh.style.transform = '';
        sh.animate([{ transform: `translateY(${s.y}px)` }, { transform: 'translateY(0)' }], { duration: sp.duration, easing: sp.easing });
        const o = +(scrim.style.opacity || 1);
        scrim.style.opacity = '';
        scrim.animate([{ opacity: o }, { opacity: 1 }], { duration: 200 });
      };
      sh.addEventListener('pointerup', up);
      sh.addEventListener('pointercancel', up);
    }
    function closeSheet(animate = true, from = 0, v = 0) {
      const sh = sheetLayer.querySelector('.a-sheet');
      if (!sh) return;
      const id = sheetLayer.dataset.work;
      const done = () => { sheetLayer.innerHTML = ''; delete sheetLayer.dataset.work; };
      ctx.setView(`work/${id}`);
      if (!animate) return done();
      const h = sh.offsetHeight;
      const sp = VA.spring({ response: 0.3, v0: h - from > 1 ? v / (h - from) : 0 });
      const opts = v > 0 ? { duration: sp.duration, easing: sp.easing, fill: 'forwards' } : { duration: VA.reduced() ? 1 : 280, easing: EASE, fill: 'forwards' };
      sh.animate([{ transform: `translateY(${from}px)` }, { transform: `translateY(${h + 20}px)` }], opts).finished.then(done);
      const scrim = sheetLayer.querySelector('.a-sheet__scrim');
      scrim.animate([{ opacity: +(scrim.style.opacity || 1) }, { opacity: 0 }], { duration: 240, fill: 'forwards' });
    }

    const searchWork = (w) => {
      const j = d.jobs.find((x) => x.work === w.id && VA.isLive(x));
      if (j?.state === 'failed') { VA.act.blocklist(j); return 'Blocklisted. Searching again.'; }
      if (j) return `${w.title} is already ${VA.jobVerb(j).toLowerCase()}.`;
      if (w.type === 'series') { const n = VA.act.searchMissingEpisodes(); return n ? `Searching for ${fmt.plural(n, 'missing episode')}.` : 'Every aired episode is here.'; }
      if (w.status === 'available') return `${w.title} meets its Targets.`;
      VA.act.search(w);
      return `Searching for ${w.title}.`;
    };
    api.searchWork = searchWork;

    ctx.listen(root, 'click', (e) => {
      const t = e.target;
      let el;
      if (t.closest('[data-sheet-close]')) return closeSheet();
      if ((el = t.closest('[data-sheet-do]'))) {
        const w = d.W[sheetLayer.dataset.work];
        const a = el.dataset.sheetDo;
        closeSheet();
        if (a === 'search') api.toast(searchWork(w));
        else if (a === 'monitor') { w.unmonitored = !w.unmonitored; api.toast(w.unmonitored ? `${w.title} is no longer monitored.` : `${w.title} is monitored again.`); }
        else if (a === 'refresh') api.toast(`Refreshing metadata for ${w.title}.`);
        else if (a === 'manual') api.toast('Manual search lists releases from 4 indexers.');
        else if (a === 'profile') api.toast(`${w.title} uses the ${d.profiles.find((p) => p.id === w.profile).name} Profile.`);
        else api.toast('Deleting is off in the prototype.');
        return;
      }
      if (t.closest('[data-back]')) return pop();
      if ((el = t.closest('[data-actions]'))) return openSheet(d.W[el.dataset.actions]);
      if (top()?.click?.(e) === true) return;
      if ((el = t.closest('[data-need]'))) {
        const n = VA.needs().find((x) => x.id === el.dataset.need);
        if (n?.job?.state === 'failed') { el.classList.add('is-loading'); VA.act.blocklist(n.job); api.toast('Blocklisted. Searching again.'); }
        else if (n?.job?.state === 'stalled') { VA.act.findAnother(n.job); api.toast('Looking for a release with seeders.'); }
        else if (n?.node) push('nodes');
        else if (n) push('requests');
        return;
      }
      if ((el = t.closest('[data-approve]'))) { const r = d.requests.find((x) => x.id === el.dataset.approve); VA.act.approve(r); api.toast(`Approved ${d.W[r.work].title}.`); return; }
      if ((el = t.closest('[data-deny]'))) { const r = d.requests.find((x) => x.id === el.dataset.deny); VA.act.deny(r, ''); api.toast(`Denied ${d.W[r.work].title}.`); return; }
      const a = t.closest('a[href^="#"]');
      if (!a) return;
      e.preventDefault();
      const v = a.getAttribute('href').slice(1);
      if (a.dataset.tab) {
        if (a.dataset.tab === tab && stack.length > 1) { while (stack.length > 2) { const s = stack.pop(); s.destroy?.(); s.el.remove(); } pop(); }
        else go(v);
        return;
      }
      if (v.startsWith('work/') || ['nodes', 'requests', 'system', 'settings', 'discover'].includes(v)) push(v);
      else go(v);
    });

    const counts = () => {
      const run = d.jobs.filter((j) => j.state === 'running').length;
      const b = root.querySelector('[data-badge]');
      b.textContent = run ? String(run) : '';
      b.hidden = !run;
      root.querySelector('.a-tabbar .a-flowlines').classList.toggle('is-running', !!run && !VA.reduced());
    };
    ctx.on('second', () => { counts(); stack.forEach((s) => s.live?.()); });
    for (const ev of ['job:new', 'job:done', 'job:update', 'request:update', 'work:update', 'history:new']) ctx.on(ev, (p) => { counts(); stack.forEach((s) => s.refresh?.(ev, p)); });

    go(ctx.view || 'home', true);
    counts();
    return { unmount() { stack.forEach((s) => s.destroy?.()); } };
  };

  const titleOf = (view) => {
    const [n, id] = view.split('/');
    if (n === 'work') return d.W[id]?.title || 'Back';
    return { home: 'Flow', library: 'Library', activity: 'Activity', search: 'Search', more: 'More', nodes: 'Nodes', requests: 'Requests' }[n] || 'Back';
  };

  const phJob = (j) => {
    const w = d.W[j.work];
    const st = j.state;
    const eta = VA.jobEta(j) ? VA.jobEta(j) + ' left' : st === 'queued' ? 'Waiting' : st === 'stalled' ? 'No seeders' : '';
    return `<a class="a-pjob" href="#work/${w.id}" data-key="${j.id}" data-state="${st}">
      <span class="a-pjob__art">${VA.posterImg(w, 'a-poster--sm')}</span>
      <span class="a-pjob__t"><b>${esc(VA.jobTitle(j))}</b><span class="${st === 'failed' ? 'is-sep' : ''}" data-flip="line">${esc(VA.jobLine(j))} · ${esc(j.node)}</span>
        ${st === 'failed' ? `<span class="a-pjob__msg">${esc(j.message)}</span>` : `<span class="a-pjob__p">${VA.prog(j.progress, st)}<i class="a-num" data-live="pct">${fmt.pct(j.progress)}</i></span>`}
        <span class="a-pjob__r a-num"><i data-live="rate">${esc(VA.jobRate(j))}</i><i data-live="eta">${esc(eta)}</i></span>
      </span>
    </a>`;
  };
  const livePJobs = (box) => box.querySelectorAll('.a-pjob').forEach((row) => {
    const j = d.jobs.find((x) => x.id === row.dataset.key);
    if (!j) return;
    const set = (k, v) => { const x = row.querySelector(`[data-live="${k}"]`); if (x && x.textContent !== v) x.textContent = v; };
    set('pct', fmt.pct(j.progress));
    set('rate', VA.jobRate(j));
    set('eta', VA.jobEta(j) ? VA.jobEta(j) + ' left' : j.state === 'queued' ? 'Waiting' : j.state === 'stalled' ? 'No seeders' : '');
    row.querySelector('.a-prog')?.style.setProperty('--a-p', j.progress.toFixed(4));
  });
  const phNode = (n) => `<a class="a-pnode" href="#nodes" data-node="${n.id}"><span class="a-pnode__b">${VA.nodeBody(n, { w: 120, h: 44 })}</span><span class="a-pnode__t"><b>${esc(n.name)}${n.controller ? '<span class="a-tag">Controller</span>' : ''}</b><span data-live="state" class="${n.state === 'offline' ? 'is-off' : ''}">${esc(VA.nodeState(n))}</span></span><span class="a-pnode__r a-num" data-live="rates">${esc(VA.nodeRates(n))}</span></a>`;

  const SCREENS = {};

  SCREENS.home = (_, api) => {
    const el = screen({ title: 'Flow', right: `<a class="a-nav__icon" href="#requests" aria-label="Requests">${icon('inbox', { size: 22 })}</a>`, body: `
      <p class="a-psent"><span class="a-psent__l1"><span data-s="lead"></span><span data-s="at"> at <span class="a-num" data-s="rate"></span></span>.</span><button class="a-sentence__needs" data-s="needs" type="button"></button></p>
      <div class="a-pband" data-tunnel></div>
      <section class="a-psec"><h2 class="a-psec__h">Needs you</h2><div class="a-plist" data-list="needs"></div></section>
      <section class="a-psec"><h2 class="a-psec__h">Arrived tonight</h2><div class="a-plist" data-list="arr"></div></section>
      <section class="a-psec"><h2 class="a-psec__h">Nodes</h2><div class="a-plist a-plist--inset" data-list="nodes">${d.nodes.map(phNode).join('')}</div></section>` });
    const q = (s) => el.querySelector(s);
    const tun = VA.Tunnel(q('[data-tunnel]'), { compact: true, onOpen: (F) => api.go(`work/${F.work.id}`), onOpenWork: (w) => api.go(`work/${w.id}`), onResolve: (m) => api.toast(m) });
    const needsBtn = q('[data-s="needs"]');
    const sentence = (first) => {
      const s = VA.sentence();
      if (first) q('[data-s="lead"]').textContent = s.lead; else VA.flip(q('[data-s="lead"]'), s.lead);
      q('[data-s="at"]').hidden = !s.rate;
      q('[data-s="rate"]').textContent = s.rate;
      if (first) needsBtn.textContent = s.needs; else VA.flip(needsBtn, s.needs);
      needsBtn.disabled = !s.sep;
    };
    let ids = '';
    const lists = (force) => {
      const n = VA.needs();
      const a = VA.arrivals(12).slice(0, 4);
      const k = n.map((x) => x.id + x.title).join() + a.map(VA.arrivalKey).join();
      if (!force && k === ids) return;
      ids = k;
      VA.patch(q('[data-list="needs"]'), n, (x) => x.id, VA.needRow);
      VA.emptyState(q('[data-list="needs"]'), !n.length, 'Nothing needs you.', 'Failures and stalls appear here the moment they happen.');
      VA.patch(q('[data-list="arr"]'), a, VA.arrivalKey, VA.arrRow);
    };
    sentence(true);
    lists(true);
    return {
      el,
      click(e) { if (e.target.closest('[data-s="needs"]')) { tun.pinSeparation(); return true; } return false; },
      live() { sentence(); lists(); VA.liveNodes(q('[data-list="nodes"]')); },
      refresh() { sentence(); lists(true); },
      destroy() { tun.destroy(); },
    };
  };

  SCREENS.library = (_, api) => {
    const st = VA.phLib;
    const el = screen({ title: 'Library', body: `<div class="a-pseg" data-slot="type"></div><div class="a-chips" data-slot="chips"></div><div class="a-pgrid" data-grid></div>` });
    const q = (s) => el.querySelector(s);
    const bucket = (w) => (w.type === 'series' ? (w.counts.downloading ? 'downloading' : w.counts.missing ? 'missing' : 'available') : d.jobs.some((j) => j.work === w.id && j.state === 'failed') ? 'failed' : w.status);
    const render = () => {
      const all = d.works.filter((w) => w.type === st.type).sort((a, b) => b.added - a.added);
      q('[data-slot="type"]').innerHTML = VA.seg('ptype', [['movie', 'Movies'], ['series', 'Series']], st.type);
      const opts = [['all', 'All'], ['downloading', 'Downloading'], ['missing', 'Missing'], ['upgradable', 'Upgradable'], ['unreleased', 'Unreleased'], ['failed', 'Failed']].map(([k, l]) => [k, l, k === 'all' ? all.length : all.filter((w) => bucket(w) === k).length]).filter(([k, , n]) => k === 'all' || n);
      if (!opts.some(([k]) => k === st.status)) st.status = 'all';
      q('[data-slot="chips"]').innerHTML = opts.map(([k, l, n]) => `<button class="a-chipbtn" aria-pressed="${st.status === k}" data-chip="${k}">${l}<span class="a-num">${n}</span></button>`).join('');
      q('[data-grid]').innerHTML = all.filter((w) => st.status === 'all' || bucket(w) === st.status).map(VA.tile).join('');
    };
    render();
    return {
      el,
      click(e) {
        let b;
        if ((b = e.target.closest('[data-seg="ptype"] [data-v]'))) { st.type = b.dataset.v; st.status = 'all'; render(); api.ctx.setView(st.type === 'series' ? 'series' : 'library'); return true; }
        if ((b = e.target.closest('[data-chip]'))) { st.status = b.dataset.chip; render(); return true; }
        return false;
      },
      live() {
        el.querySelectorAll('.a-tile').forEach((t) => {
          const w = d.W[t.dataset.key];
          const s = VA.statusOf(w);
          if (s.job) t.querySelector('.a-prog')?.style.setProperty('--a-p', s.job.progress.toFixed(4));
          const m = t.querySelector('[data-flip="st"]');
          if (s.tone === 'live' && m.textContent !== s.text) m.textContent = s.text;
        });
      },
      refresh(ev) { if (ev !== 'job:update') render(); },
    };
  };

  SCREENS.work = (id, api, back) => {
    const w = d.W[id];
    const prof = d.profiles.find((p) => p.id === w.profile);
    let season = w.type === 'series' ? (w.seasons.findIndex((s) => s.episodes.some((e) => e.status === 'downloading' || e.status === 'missing' || e.status === 'failed')) + 1 || w.seasons.length) : 0;
    const el = screen({
      title: w.title, back: back || 'Library',
      hero: `<div class="a-ph-hero"><img class="a-ph-hero__bd" src="${SS.art.backdrop(w, { w: 800, h: 640 })}" alt=""><span class="a-ph-hero__poster">${VA.posterImg(w)}</span></div>`,
      right: `<button class="a-nav__icon" data-actions="${w.id}" aria-label="Actions">${icon('ellipsis', { size: 22 })}</button>`,
      body: `<p class="a-pwork__meta">${VA.workMeta(w, true)}</p><p class="a-pwork__genres">${esc(w.genres.join(', '))}${w.rating ? ` · Rated ${w.rating.toFixed(1)}` : ''}</p>
        <div class="a-pwork__status" data-wstatus></div>
        <div class="a-pwork__acts">${VA.btn('Search', { kind: 'primary', ic: icon('search', { size: 16 }), attrs: 'data-psearch' })}${VA.btn('Actions', { ic: icon('ellipsis', { size: 16 }), attrs: `data-actions="${w.id}"` })}</div>
        <p class="a-pwork__ov">${esc(w.overview)}</p>
        <section class="a-psec"><h2 class="a-psec__h">Path</h2><div class="a-pstrip" data-path>${VA.pathStrip(w)}</div></section>
        <section class="a-psec"><h2 class="a-psec__h">Targets <span>${esc(prof.name)} Profile</span></h2><div class="a-plist a-plist--inset" data-cov></div></section>
        ${w.type === 'series' ? `<section class="a-psec"><h2 class="a-psec__h">Seasons</h2><div class="a-pseg a-pseg--scroll" data-slot="season"></div><div class="a-plist a-plist--inset" data-eps></div></section>` : ''}
        <section class="a-psec"><h2 class="a-psec__h">Files</h2><div class="a-plist a-plist--inset" data-files></div></section>`,
    });
    el.classList.add('a-scr--work');
    const q = (s) => el.querySelector(s);
    const cov = () => {
      const live = d.jobs.filter((j) => j.work === w.id && VA.isLive(j));
      q('[data-cov]').innerHTML = d.targets.filter((t) => prof.targets.includes(t.id)).map((t) => {
        const j = live.find((x) => x.target === t.id && !x.episode);
        const c = w.coverage[t.id];
        const right = j ? (j.state === 'failed' ? `<span class="is-sep">Import failed</span>` : `<span class="a-num">${esc(VA.jobVerb(j))} ${fmt.pct(j.progress)}</span>`) : `<span class="is-muted">${esc(fmt.coverageLabel[c])}</span>`;
        return `<div class="a-prow"><span class="a-cov a-cov--${j && j.state !== 'failed' ? 'downloading' : c}"></span><span class="a-prow__t"><b>${esc(t.name)}</b><small>${t.kind === 'derived' ? `Derived from ${esc(VA.targetName(t.from))}` : 'Downloaded'}</small></span>${right}</div>`;
      }).join('');
    };
    const files = () => {
      const fs = w.type === 'series' ? [] : w.files || [];
      q('[data-files]').innerHTML = fs.length ? fs.map((f) => `<div class="a-prow a-prow--file"><span class="a-prow__t"><span class="a-mono">${esc(f.name)}</span><small>${esc(VA.targetName(f.target))} · ${fmt.bytes(f.size)} · ${esc(f.video)}</small></span></div>`).join('') : `<div class="a-prow"><span class="a-prow__t"><small>${w.type === 'series' ? `${w.counts.available} episode files on vault.` : 'No files yet. They appear once an import finishes on vault.'}</small></span></div>`;
    };
    const eps = () => {
      if (w.type !== 'series') return;
      q('[data-slot="season"]').innerHTML = VA.seg('pseason', w.seasons.map((s) => [String(s.n), `Season ${s.n}`]), String(season));
      const s = w.seasons[season - 1];
      q('[data-eps]').innerHTML = s.episodes.map((e) => {
        const j = d.jobs.find((x) => x.work === w.id && x.episode && x.episode[0] === s.n && x.episode[1] === e.n && VA.isLive(x));
        const st = j ? `<span class="a-num">${esc(VA.jobVerb(j))} ${fmt.pct(j.progress)}</span>` : e.status === 'unreleased' ? `<span class="is-muted">${esc(fmt.until(e.air))}</span>` : e.status === 'missing' ? `<span class="is-muted">Missing</span>` : e.status === 'failed' ? `<span class="is-sep">Failed</span>` : `<span class="is-muted">1080p</span>`;
        return `<div class="a-prow a-prow--ep ${j ? 'is-live' : ''}"><span class="a-num a-prow__n">${e.n}</span><span class="a-prow__t"><b>${esc(e.title)}</b><small>${e.air.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })}</small></span>${st}</div>`;
      }).join('');
    };
    const status = () => {
      const j = d.jobs.find((x) => x.work === w.id && VA.isLive(x));
      const box = q('[data-wstatus]');
      if (!j) { box.innerHTML = `<span>${esc(w.type === 'series' ? `${w.counts.available} of ${w.counts.total} episodes${w.nextAir ? ` · next ${fmt.until(w.nextAir)}` : ''}` : VA.statusOf(w).text)}</span>`; return; }
      if (j.state === 'failed') { box.innerHTML = `<span class="is-sep">${VA.vortex(16)}${esc(j.message)}</span>`; return; }
      box.innerHTML = `<span>${esc(VA.jobLine(j))}</span><span class="a-num">${fmt.pct(j.progress)}${j.eta ? ` · ${esc(VA.jobEta(j))} left` : ''}</span>${VA.prog(j.progress, j.state)}`;
    };
    cov(); files(); eps(); status();
    return {
      el,
      click(e) {
        const b = e.target.closest('[data-seg="pseason"] [data-v]');
        if (b) { season = +b.dataset.v; eps(); return true; }
        if (e.target.closest('[data-psearch]')) { api.toast(api.searchWork(w)); return true; }
        return false;
      },
      live() { status(); if (d.jobs.some((x) => x.work === w.id && VA.isLive(x))) { q('[data-path]').innerHTML = VA.pathStrip(w); cov(); eps(); } },
      refresh(ev, p) { if (p?.work === w.id) { cov(); eps(); files(); status(); q('[data-path]').innerHTML = VA.pathStrip(w); } },
    };
  };

  SCREENS.activity = () => {
    const el = screen({ title: 'Activity', body: `<section class="a-psec" data-g="needs"><h2 class="a-psec__h">Needs you</h2><div class="a-plist"></div></section><section class="a-psec" data-g="running"><h2 class="a-psec__h">Moving</h2><div class="a-plist"></div></section><section class="a-psec" data-g="waiting"><h2 class="a-psec__h">Waiting</h2><div class="a-plist"></div></section>` });
    const groupOf = (j) => (j.state === 'failed' || j.state === 'stalled' ? 'needs' : j.state === 'queued' ? 'waiting' : 'running');
    const render = () => {
      const jobs = d.jobs.filter(VA.isLive);
      for (const g of ['needs', 'running', 'waiting']) {
        const sec = el.querySelector(`[data-g="${g}"]`);
        const list = jobs.filter((j) => groupOf(j) === g);
        sec.hidden = !list.length;
        VA.patch(sec.querySelector('.a-plist'), list, (j) => j.id, phJob);
      }
    };
    render();
    return { el, live() { livePJobs(el); }, refresh() { render(); } };
  };

  SCREENS.search = () => {
    const el = screen({ title: 'Search', body: `<label class="a-psearch">${icon('search', { size: 17 })}<input type="search" placeholder="Works, people, releases" aria-label="Search the library" enterkeyhint="search" autocomplete="off" data-q></label><div data-results></div>` });
    const input = el.querySelector('[data-q]');
    const out = el.querySelector('[data-results]');
    const row = (w) => { const s = VA.statusOf(w); return `<a class="a-prow a-prow--work" href="#work/${w.id}">${VA.posterImg(w, 'a-poster--sm')}<span class="a-prow__t"><b>${esc(w.title)}</b><small>${w.type === 'series' ? 'Series' : 'Movie'} · ${w.year}</small></span><span class="${s.tone === 'sep' ? 'is-sep' : 'is-muted'}">${esc(s.tone === 'live' ? s.text.split(' · ')[0] : s.tone ? s.text : '')}</span></a>`; };
    const render = () => {
      const qs = input.value.trim().toLowerCase();
      if (!qs) {
        const missing = d.works.filter((w) => w.status === 'missing');
        out.innerHTML = `<section class="a-psec"><h2 class="a-psec__h">Missing</h2><div class="a-plist a-plist--inset">${missing.map(row).join('')}</div></section><section class="a-psec"><h2 class="a-psec__h">Recently added</h2><div class="a-plist a-plist--inset">${d.works.slice().sort((a, b) => b.added - a.added).slice(0, 6).map(row).join('')}</div></section>`;
        return;
      }
      const res = d.works.filter((w) => w.title.toLowerCase().includes(qs) || w.cast.some((c) => c.toLowerCase().includes(qs)) || (w.director || w.creator || '').toLowerCase().includes(qs));
      out.innerHTML = res.length ? `<div class="a-plist a-plist--inset">${res.map(row).join('')}</div>` : `<div class="a-empty"><p>Nothing in the library matches “${esc(input.value)}”.</p><p class="a-empty__sub">Try a title, an actor or a director.</p></div>`;
    };
    input.addEventListener('input', render);
    render();
    return { el };
  };

  SCREENS.more = () => {
    const missingEps = d.works.filter((w) => w.type === 'series').reduce((s, w) => s + w.counts.missing, 0);
    const missingMovies = d.works.filter((w) => w.type === 'movie' && w.status === 'missing').length;
    const upg = d.works.filter((w) => w.status === 'upgradable').length;
    const pend = d.requests.filter((r) => r.state === 'pending').length;
    const off = d.nodes.filter((n) => n.state === 'offline').length;
    const item = (href, ic, label, value = '') => `<a class="a-prow a-prow--link" href="#${href}"><span class="a-prow__ic">${ic}</span><span class="a-prow__t"><b>${label}</b></span><span class="is-muted a-num">${value}</span>${chevron()}</a>`;
    const el = screen({ title: 'More', body: `
      <section class="a-psec"><h2 class="a-psec__h">Discover</h2><div class="a-plist a-plist--inset">${item('discover', icon('compass', { size: 20 }), 'Coming up', String(d.calendar.length))}${item('discover', icon('calendar', { size: 20 }), 'Not yet released', String(d.works.filter((w) => w.status === 'unreleased').length))}</div></section>
      <section class="a-psec"><h2 class="a-psec__h">Library</h2><div class="a-plist a-plist--inset">${item('library', icon('film', { size: 20 }), 'Missing movies', String(missingMovies))}${item('series', icon('tv', { size: 20 }), 'Missing episodes', String(missingEps))}${item('library', icon('arrow-up', { size: 20 }), 'Upgradable', String(upg))}</div></section>
      <section class="a-psec"><h2 class="a-psec__h">Run</h2><div class="a-plist a-plist--inset">${item('requests', icon('inbox', { size: 20 }), 'Requests', pend ? `${pend} pending` : '')}${item('nodes', icon('server', { size: 20 }), 'Nodes', `${d.nodes.length}${off ? `, ${off} offline` : ''}`)}</div></section>
      <section class="a-psec"><h2 class="a-psec__h">Settings</h2><div class="a-plist a-plist--inset">${item('settings', icon('sliders-horizontal', { size: 20 }), 'Targets and Profiles')}${item('settings', icon('user', { size: 20 }), 'Accounts')}${item('system', icon('gauge', { size: 20 }), 'System', '2.3.1 ready')}</div></section>
      <p class="a-pfoot a-num">SlipStream 2.3.0 · atlas is the Controller<br>Signed in as Alex Rowan</p>` });
    return { el };
  };

  SCREENS.nodes = (_, __, back) => {
    const el = screen({ title: 'Nodes', back: back || 'More', body: `<div class="a-plist a-plist--inset">${d.nodes.map(phNode).join('')}</div>${d.nodes.filter((n) => n.stateNote).map((n) => `<p class="a-pnote"><b>${esc(n.name)}</b> ${esc(n.stateNote)}.</p>`).join('')}` });
    return { el, live() { VA.liveNodes(el); } };
  };
  SCREENS.requests = (_, __, back) => {
    const el = screen({ title: 'Requests', back: back || 'More', body: `<div class="a-plist a-plist--inset" data-list></div>` });
    const render = () => {
      const list = d.requests.slice().sort((a, b) => (a.state === 'pending' ? -1 : 0) - (b.state === 'pending' ? -1 : 0) || b.at - a.at);
      VA.patch(el.querySelector('[data-list]'), list, (r) => r.id, (r) => {
        const w = d.W[r.work];
        const J = VA.journey(r);
        return `<div class="a-preq" data-state="${r.state}">${VA.posterImg(w, 'a-poster--sm')}<span class="a-prow__t"><b>${esc(w.title)}</b><small>${esc(d.A[r.by].name)} · ${esc(fmt.ago(r.at))}</small><span class="a-preq__st" data-flip="st">${esc(r.state === 'pending' ? 'Waiting for you' : r.state === 'denied' ? 'Denied' : J.label)}</span>${r.state === 'pending' ? `<span class="a-preq__acts"><button class="a-btn a-btn--sm a-btn--primary" data-approve="${r.id}">Approve</button><button class="a-btn a-btn--sm" data-deny="${r.id}">Deny</button></span>` : ''}</span></div>`;
      });
    };
    render();
    return { el, refresh() { render(); } };
  };
  SCREENS.discover = (_, __, back) => {
    const el = screen({ title: 'Coming up', back: back || 'More', body: `<div class="a-plist a-plist--inset">${d.calendar.slice().sort((a, b) => a.at - b.at).map((c) => { const w = d.W[c.work]; return `<a class="a-prow a-prow--work" href="#work/${w.id}">${VA.posterImg(w, 'a-poster--sm')}<span class="a-prow__t"><b>${esc(w.title)}</b><small>${c.episode ? `${fmt.ep(c.episode)} · ${esc(c.title)}` : esc(c.title)}</small></span><span class="is-muted">${esc(fmt.until(c.at))}</span></a>`; }).join('')}</div>` });
    return { el };
  };
  SCREENS.settings = (_, __, back) => {
    const el = screen({ title: 'Targets and Profiles', back: back || 'More', body: `<section class="a-psec"><h2 class="a-psec__h">Targets</h2><div class="a-plist a-plist--inset">${d.targets.map((t) => `<div class="a-prow"><span class="a-prow__t"><b>${esc(t.name)}</b><small>${esc(t.note)}</small></span></div>`).join('')}</div></section><section class="a-psec"><h2 class="a-psec__h">Profiles</h2><div class="a-plist a-plist--inset">${d.profiles.map((p) => `<div class="a-prow"><span class="a-prow__t"><b>${esc(p.name)}</b><small>${esc(fmt.list(p.targets.map(VA.targetName)))}</small></span></div>`).join('')}</div></section>` });
    return { el };
  };
  SCREENS.system = (_, __, back) => {
    const el = screen({ title: 'System', back: back || 'More', body: `<section class="a-psec"><h2 class="a-psec__h">Health</h2><div class="a-plist a-plist--inset">${d.alerts.map((a) => `<div class="a-prow"><span class="a-prow__ic">${a.level === 'error' ? VA.vortex(18) : a.level === 'warning' ? VA.thinning(18) : VA.laminar(18)}</span><span class="a-prow__t"><b>${esc(a.title)}</b><small>${esc(a.detail)}</small></span></div>`).join('')}</div></section><p class="a-pfoot">SlipStream 2.3.1 is ready. atlas updates first, then each Worker in turn.</p>` });
    return { el };
  };
})();
