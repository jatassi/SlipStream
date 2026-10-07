// PROTOTYPE. House Lights: shared model, live bindings, light and motion helpers.
(function () {
  const SS = window.SS;
  const VB = (window.VB = window.VB || {});
  const D = SS.data, F = SS.fmt, ART = SS.art;
  const esc = F.esc;

  VB.icon = (name, size = 18) => SS.icon(name, { size, stroke: 1.5 });
  VB.reduced = () => matchMedia('(prefers-reduced-motion: reduce)').matches;
  VB.href = (view) => {
    const p = new URLSearchParams(location.search);
    view ? p.set('view', view) : p.delete('view');
    return '?' + p.toString();
  };
  VB.name = (id) => D.A[id]?.name || id;
  VB.first = (id) => VB.name(id).split(' ')[0];
  VB.av = (id, cls = '') => `<span class="b-av ${cls}" aria-hidden="true">${esc(F.initials(VB.name(id)))}</span>`;

  // The poster face of a Work, as an inline style (single quotes so it survives an attribute).
  VB.face = (w) => {
    const f = ART.font(w);
    return `font-family:${f.family.replace(/"/g, "'")},serif;font-weight:${f.weight};letter-spacing:${f.tracking};${f.upper ? 'text-transform:uppercase;' : ''}`;
  };
  VB.glow = (w) => ART.colors(w).glow;
  VB.tint = (w) => {
    const c = ART.colors(w);
    return `--glow:${c.glow};--glow-a:${ART.rgba(c.glow, 0.55)};--glow-b:${ART.rgba(c.accent, 0.4)};--glow-c:${ART.rgba(c.glow, 0.16)}`;
  };

  const measure = document.createElement('canvas').getContext('2d');
  // A Work's title set as its logo, fitted to a width in one or two lines.
  VB.logo = (w, { maxW, max, min = 26, tag = 'h1', cls = '', text } = {}) => {
    const f = ART.font(w);
    const raw = text || w.title;
    const t = f.upper ? raw.toUpperCase() : raw;
    const track = parseFloat(f.tracking);
    const unit = (s) => {
      measure.font = `${f.weight} 100px ${f.family}`;
      return (measure.measureText(s).width + track * 100 * (s.length - 1)) / 100;
    };
    let size = Math.min(max, maxW / unit(t));
    let lines = [t];
    if (size < max * 0.66 && t.includes(' ')) {
      const words = t.split(' ');
      let best = null;
      for (let i = 1; i < words.length; i++) {
        const a = words.slice(0, i).join(' '), b = words.slice(i).join(' ');
        const m = Math.max(unit(a), unit(b));
        if (!best || m < best.m) best = { m, ls: [a, b] };
      }
      const s2 = Math.min(max * 0.84, maxW / best.m);
      if (s2 > size * 1.18) { size = s2; lines = best.ls; }
    }
    size = Math.max(min, Math.floor(size));
    return `<${tag} class="b-logo ${cls}" style="${VB.face(w)}font-size:${size}px" aria-label="${esc(raw)}">${lines.map((l) => `<span>${esc(l)}</span>`).join('')}</${tag}>`;
  };

  // ---------- Model ----------
  // Acquisition is one pipeline per Item: download, then Transfer, then import.
  // Each stage owns a slice of the Work's develop fraction so colour never runs backwards.
  const STAGE = { download: [0, 0.9], transfer: [0.9, 0.98], import: [0.98, 1] };
  const RANK = { running: 0, stalled: 1, failed: 2, queued: 3 };
  const epKey = (e) => (e ? `${e[0]}x${e[1]}` : '');
  VB.epKey = epKey;
  VB.job = (id) => D.jobs.find((j) => j.id === id);
  VB.isAcq = (j) => !!STAGE[j.kind];

  VB.acq = (workId, ep) => {
    let best = null;
    for (const j of D.jobs) {
      if (j.work !== workId || !STAGE[j.kind] || !(j.state in RANK) || VB.local.resolved.has(j.id)) continue;
      if (ep && epKey(j.episode) !== epKey(ep)) continue;
      if (!best || RANK[j.state] < RANK[best.state]) best = j;
    }
    return best;
  };
  VB.frac = (j) => {
    if (!j || j.state === 'done') return 1;
    if (j.state === 'queued' || j.state === 'failed') return 0;
    const r = STAGE[j.kind];
    return r ? r[0] + (r[1] - r[0]) * j.progress : j.progress;
  };
  // Seconds until the Item can be watched: this stage plus the stages still to come.
  VB.readyIn = (j) => {
    if (!j || j.state !== 'running' || j.eta == null) return null;
    const s = j.size || 0;
    if (j.kind === 'download') return j.eta + s / 96e6 + s / 700e6;
    if (j.kind === 'transfer') return j.eta + s / 700e6;
    return j.eta;
  };
  VB.arrived = (id) => D.history.some((h) => h.kind === 'imported' && h.work === id && !h.episode);

  // The six library states (P-030), derived from live jobs first and the stored status second.
  VB.status = (w) => {
    const j = VB.acq(w.id);
    if (w.type === 'series') return { s: w.counts.missing ? 'missing' : 'available', j };
    if (j) return { s: j.state === 'failed' ? 'failed' : 'downloading', j };
    if (VB.arrived(w.id)) return { s: 'available', j: null };
    if (w.status === 'downloading') return { s: 'missing', j: null };
    return { s: w.status, j: null };
  };
  VB.statusLabel = { available: 'Available', upgradable: 'Upgradable', downloading: 'Downloading', missing: 'Missing', failed: 'Failed', unreleased: 'Unreleased' };

  // One cell per Target: the stored coverage, corrected by whatever job is touching it now.
  VB.coverage = (w) => {
    const prof = D.profiles.find((p) => p.id === w.profile);
    return D.targets.map((t) => {
      const wanted = prof.targets.includes(t.id);
      let c = wanted ? w.coverage[t.id] : 'unwanted';
      const j = D.jobs.find((x) => x.work === w.id && x.target === t.id && !x.episode && x.state !== 'done' && !VB.local.resolved.has(x.id));
      if (wanted && j) {
        if (j.state === 'failed') c = 'failed';
        else if (j.state === 'stalled') c = 'stalled';
        else if (j.state === 'queued') c = 'queued';
        else c = j.kind === 'transcode' ? 'deriving' : 'downloading';
      } else if (c === 'downloading') c = 'missing';
      return { t, c, j: wanted ? j : null };
    });
  };

  // ---------- Local state: what the admin and the Member do in this session ----------
  VB.local = VB.local || { req: {}, mine: [], resolved: new Set(), monitored: {}, seen: new Set() };
  VB.reqs = () => D.requests.concat(VB.local.mine);
  VB.reqState = (r) => (r.state === 'available' || r.state === 'downloading' ? r.state : VB.local.req[r.id] || r.state);

  VB.aisle = () => {
    const out = [];
    const pend = VB.reqs().filter((r) => VB.reqState(r) === 'pending');
    if (pend.length) {
      const who = [...new Set(pend.map((r) => VB.first(r.by)))];
      out.push({ tone: 'amber', id: 'requests', title: `${F.plural(pend.length, 'request')} to review`, sub: `From ${F.list(who)}`, act: 'Review', go: 'requests' });
    }
    for (const j of D.jobs) {
      if (VB.local.resolved.has(j.id)) continue;
      if (j.state === 'stalled') out.push({ tone: 'amber', id: j.id, title: `${j.label} has stalled`, sub: `No seeders for 41 min on ${j.node}`, act: 'Find another release', go: `work/${j.work}`, job: j.id });
    }
    for (const j of D.jobs) {
      if (VB.local.resolved.has(j.id)) continue;
      if (j.state === 'failed') out.push({ tone: 'exit', id: j.id, title: `${j.label} failed to import`, sub: 'The release is a 48 MB sample', act: 'Blocklist and search', job: j.id, cmd: 'blocklist' });
    }
    for (const n of D.nodes) if (n.state === 'offline') out.push({ tone: 'exit', id: n.id, title: `${n.name} is offline`, sub: `Last seen ${F.time(n.lastSeen)}. forge covers Transcoding`, act: `Open ${n.name}`, go: 'nodes' });
    return out;
  };

  // ---------- Live bindings ----------
  // Elements opt in with data attributes; one tick pass moves develop fronts, one per-second pass
  // rewrites text. Nothing here reads layout.
  const shortRate = (j) => (j.kind === 'transcode' || j.kind === 'subtitles' ? `${(j.speed || 0).toFixed(1)}× realtime` : F.rate(j.rate));
  VB.line = (j) => {
    if (!j || j.state === 'done') return 'Arrived';
    if (j.state === 'failed') return 'Failed: sample release';
    if (j.state === 'stalled') return 'Stalled, no seeders';
    if (j.state === 'queued') return `Queued on ${j.node}`;
    const r = VB.readyIn(j);
    if (j.kind === 'download') return `${F.rate(j.rate)} · ${F.dur(r)}`;
    if (j.kind === 'transfer') return `To ${j.to} · ${F.dur(r)}`;
    if (j.kind === 'import') return `Importing · ${F.dur(r)}`;
    return `${shortRate(j)} · ${F.dur(j.eta)}`;
  };
  VB.memberLine = (j) => {
    if (!j || j.state === 'done') return 'Ready to watch';
    if (j.state !== 'running') return 'Waiting for a copy';
    const r = VB.readyIn(j);
    return r < 60 ? 'Ready in a moment' : `Ready in about ${F.dur(r)}`;
  };
  const pctOf = (f) => `${Math.floor(f * 100)}%`;
  const LV = {
    wline: (id, ep) => VB.line(VB.acq(id, ep ? ep.split('x').map(Number) : undefined)),
    wmember: (id, ep) => VB.memberLine(VB.acq(id, ep ? ep.split('x').map(Number) : undefined)),
    wpct: (id, ep) => pctOf(VB.frac(VB.acq(id, ep ? ep.split('x').map(Number) : undefined))),
    jpct: (id) => { const j = VB.job(id); return j.state === 'queued' ? '0%' : pctOf(j.progress); },
    jrate: (id) => { const j = VB.job(id); return j.state === 'running' ? shortRate(j) : j.state === 'done' ? 'Done' : '0 B/s'; },
    jeta: (id) => { const j = VB.job(id); return j.state === 'running' ? F.dur(j.eta) : ''; },
    jline: (id) => VB.line(VB.job(id)),
    active: () => String(D.jobs.filter((j) => j.state === 'running').length),
    down: () => F.rate(SS.sim.totals().down),
    clock: () => F.time(SS.sim.clock),
    ago: (ts) => F.ago(new Date(+ts)),
    ncpu: (id) => pctOf(D.N[id].cpu),
    nmem: (id) => pctOf(D.N[id].mem),
    ngpu: (id) => pctOf(D.N[id].gpu || 0),
    ndown: (id) => F.rate(D.N[id].down),
    nup: (id) => F.rate(D.N[id].up),
  };
  const LM = {
    ncpu: (id) => D.N[id].cpu,
    nmem: (id) => D.N[id].mem,
    ngpu: (id) => D.N[id].gpu || 0,
    wfrac: (id) => VB.frac(VB.acq(id)),
  };
  VB.LV = LV;

  VB.live = function (ctx) {
    let devs = [], texts = [], meters = [];
    const api = {
      scan(root) {
        for (const el of root.querySelectorAll('[data-dev]')) {
          if (el._bl) continue;
          el._bl = 1;
          devs.push({ el, c: el.querySelector('.b-dev__c'), w: el.dataset.w, ep: el.dataset.ep ? el.dataset.ep.split('x').map(Number) : undefined, job: el.dataset.job, f: NaN, st: el.dataset.st });
        }
        for (const el of root.querySelectorAll('[data-lv]')) {
          if (el._bl) continue;
          el._bl = 1;
          const [k, ...a] = el.dataset.lv.split('|');
          if (LV[k]) texts.push({ el, fn: () => LV[k](...a), v: el.textContent });
        }
        for (const el of root.querySelectorAll('[data-lm]')) {
          if (el._bl) continue;
          el._bl = 1;
          const [k, ...a] = el.dataset.lm.split('|');
          if (LM[k]) meters.push({ el, fn: () => LM[k](...a), v: -1 });
        }
        api.tick();
      },
      tick() {
        for (const d of devs) {
          if (!d.el.isConnected) continue;
          const j = d.job ? VB.job(d.job) : VB.acq(d.w, d.ep);
          const st = j && !VB.local.resolved.has(j.id) ? j.state : 'done';
          const f = st === 'done' ? 1 : VB.frac(j);
          if (st !== d.st) { d.el.dataset.st = st; d.st = st; }
          if (!(Math.abs(f - d.f) < 0.0004)) { d.c.style.clipPath = `inset(0 ${(100 - f * 100).toFixed(2)}% 0 0)`; d.f = f; }
        }
      },
      second() {
        devs = devs.filter((d) => d.el.isConnected);
        texts = texts.filter((t) => t.el.isConnected);
        meters = meters.filter((m) => m.el.isConnected);
        for (const t of texts) { const v = t.fn(); if (v !== t.v) { t.el.textContent = v; t.v = v; } }
        for (const m of meters) { const v = Math.round(m.fn() * 1000) / 1000; if (v !== m.v) { m.el.style.transform = `scaleX(${v})`; m.v = v; } }
      },
      reset() { devs = []; texts = []; meters = []; },
    };
    ctx.on('tick', () => api.tick());
    ctx.on('second', () => api.second());
    return api;
  };

  // ---------- Develop: monochrome poster revealing colour to its progress ----------
  const devMarkup = (src, attrs, st, f, extra) =>
    `<div class="b-dev" data-dev ${attrs} data-st="${st}"${extra}><img class="b-dev__m" src="${src}" alt="" draggable="false" decoding="async"><img class="b-dev__c" src="${src}" alt="" draggable="false" decoding="async" style="clip-path:inset(0 ${(100 - f * 100).toFixed(2)}% 0 0)"></div>`;
  VB.dev = (w, { job, ep, vt, still } = {}) => {
    const j = job || VB.acq(w.id, ep);
    const st = j ? j.state : 'done';
    const src = still ? ART.still(w, still) : ART.poster(w);
    const attrs = job ? `data-job="${job.id}"` : `data-w="${w.id}"${ep ? ` data-ep="${epKey(ep)}"` : ''}`;
    return devMarkup(src, attrs, st, VB.frac(j), vt ? ` style="view-transition-name:${vt}"` : '');
  };
  VB.poster = (w, { vt, cls = '' } = {}) => `<img class="b-poster ${cls}" src="${ART.poster(w)}" alt="" draggable="false" decoding="async"${vt ? ` style="view-transition-name:${vt}"` : ''}>`;
  // A Work's art for any tile: develops while acquisition runs, a plain poster otherwise.
  VB.art = (w, opt = {}) => {
    if (w.type === 'movie' && VB.acq(w.id)) return VB.dev(w, opt);
    return VB.poster(w, opt);
  };

  // ---------- Spill: the light a Work throws on the house ----------
  VB.spill = function (host) {
    host.classList.add('b-spill');
    host.setAttribute('aria-hidden', 'true');
    host.innerHTML = '<div class="b-spill__l"></div><div class="b-spill__l"></div>';
    const L = host.children;
    let i = 0, id;
    return {
      set(w) {
        const nid = w ? w.id : null;
        if (nid === id) return;
        id = nid;
        const next = L[1 - i], prev = L[i];
        if (w) {
          const c = ART.colors(w);
          next.style.backgroundImage = `radial-gradient(58% 46% at 30% 62%, ${ART.rgba(c.glow, 0.8)}, transparent 72%), radial-gradient(46% 40% at 76% 56%, ${ART.rgba(c.accent, 0.42)}, transparent 70%), url(${ART.backdrop(w, { w: 320, h: 180 })})`;
        }
        next.classList.toggle('is-on', !!w);
        prev.classList.remove('is-on');
        i = 1 - i;
      },
      get id() { return id; },
      dim(v) { host.style.opacity = String(v); },
    };
  };

  // Now showing is the latest arrival. A new arrival waits for the current reel to run at least
  // `dwell` ms, so a burst of imports changes the picture once, not three times.
  // The reel is shared by every shell, so switching between desk, phone and portal keeps the picture.
  const reel = { cur: null, since: 0 };
  VB.nowShowing = function (ctx, onChange, dwell = 20000) {
    const latest = () => D.history.find((h) => h.kind === 'imported');
    if (!reel.cur || performance.now() - reel.since > dwell) { reel.cur = latest(); reel.since = performance.now(); }
    let timer = 0;
    const go = () => { const n = latest(); if (!n || n === reel.cur) return; reel.cur = n; reel.since = performance.now(); onChange(n); };
    const schedule = () => {
      clearTimeout(timer);
      const wait = dwell - (performance.now() - reel.since);
      if (wait <= 0) go(); else timer = setTimeout(go, wait);
    };
    ctx.on('history:new', (h) => { if (h.kind === 'imported') schedule(); });
    if (latest() !== reel.cur) schedule();
    ctx.cleanup(() => clearTimeout(timer));
    return { get: () => reel.cur };
  };

  // ---------- Motion ----------
  // A spring sampled into a CSS linear() easing, so gestures hand their velocity to the compositor.
  VB.spring = ({ response = 0.42, damping = 1, v0 = 0 } = {}) => {
    const w = (2 * Math.PI) / response, z = damping;
    const pos = (t) => {
      if (z >= 1) return 1 + (-1 + (v0 - w) * t) * Math.exp(-w * t);
      const wd = w * Math.sqrt(1 - z * z);
      const B = (v0 - z * w) / wd;
      return 1 + Math.exp(-z * w * t) * (-Math.cos(wd * t) + B * Math.sin(wd * t));
    };
    let T = 0;
    for (let t = 0; t < 2.5; t += 1 / 240) if (Math.abs(1 - pos(t)) > 0.002) T = t;
    T = Math.max(0.12, T + 1 / 60);
    const n = Math.min(90, Math.max(12, Math.ceil(T * 70)));
    const pts = [];
    for (let k = 0; k <= n; k++) pts.push(k === n ? 1 : +pos((k / n) * T).toFixed(4));
    return { easing: `linear(${pts.join(', ')})`, duration: Math.round(T * 1000) };
  };
  // Animate transform between two values with a spring, then commit the end value.
  VB.springTo = (el, from, to, opt) => {
    el.getAnimations().forEach((a) => a.cancel());
    if (VB.reduced()) { el.style.transform = to; return Promise.resolve(); }
    const s = VB.spring(opt);
    const a = el.animate([{ transform: from }, { transform: to }], { duration: s.duration, easing: s.easing });
    el.style.transform = to;
    return a.finished.catch(() => {});
  };
  // FLIP a keyed list across a re-render: leavers fade first, survivors glide, newcomers fade in.
  VB.flip = (box, render) => {
    const s = box.getBoundingClientRect().width / box.offsetWidth || 1;
    const before = new Map([...box.children].map((el) => [el.dataset.key, el.getBoundingClientRect()]));
    render();
    if (VB.reduced()) return;
    for (const el of box.children) {
      const b = before.get(el.dataset.key);
      if (!b) { el.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 320, easing: 'ease-out' }); continue; }
      const a = el.getBoundingClientRect();
      const dx = (b.left - a.left) / s, dy = (b.top - a.top) / s;
      if (dx || dy) el.animate([{ transform: `translate(${dx}px, ${dy}px)` }, { transform: 'none' }], { duration: 480, easing: 'cubic-bezier(0.23, 1, 0.32, 1)' });
    }
  };
  VB.leave = (els) => Promise.all([...els].map((el) => (VB.reduced() ? Promise.resolve() : el.animate([{ opacity: 1, transform: 'none' }, { opacity: 0, transform: 'scale(0.96)' }], { duration: 260, easing: 'ease-out', fill: 'forwards' }).finished)));

  VB.tx = (el) => new DOMMatrixReadOnly(getComputedStyle(el).transform).m41;
  VB.ty = (el) => new DOMMatrixReadOnly(getComputedStyle(el).transform).m42;

  // Same-document view transition, skipped for keyboard-initiated navigation and reduced motion.
  VB.vt = (update, { kbd = false } = {}) => {
    if (kbd || VB.reduced() || !document.startViewTransition) { update(); return null; }
    try { return document.startViewTransition(update); } catch { update(); return null; }
  };

  // ---------- Overlays ----------
  VB.toast = (root, text) => {
    let host = root.querySelector('.b-toasts');
    if (!host) { host = document.createElement('div'); host.className = 'b-toasts'; host.setAttribute('role', 'status'); root.querySelector('.b-app').append(host); }
    const t = document.createElement('div');
    t.className = 'b-toast';
    t.textContent = text;
    host.append(t);
    requestAnimationFrame(() => requestAnimationFrame(() => t.classList.add('is-in')));
    setTimeout(() => { t.classList.remove('is-in'); setTimeout(() => t.remove(), 320); }, 3800);
  };

  // Rect of `el` in the root's own (unscaled) coordinates, so overlays line up inside the device frame.
  VB.rel = (root, el) => {
    const R = root.getBoundingClientRect(), r = el.getBoundingClientRect();
    const s = R.width / root.offsetWidth || 1;
    return { left: (r.left - R.left) / s, top: (r.top - R.top) / s, width: r.width / s, height: r.height / s, right: (r.right - R.left) / s, bottom: (r.bottom - R.top) / s, W: root.offsetWidth, H: root.offsetHeight };
  };
  VB.popover = function (ctx, root) {
    let open = null;
    const close = (focus) => {
      if (!open) return;
      open.el.remove();
      open.btn.setAttribute('aria-expanded', 'false');
      if (focus) open.btn.focus();
      open = null;
    };
    const show = (btn, html, { cls = '', width = 340 } = {}) => {
      if (open && open.btn === btn) return close();
      close();
      const host = root.querySelector('.b-layer');
      const el = document.createElement('div');
      el.className = `b-pop ${cls}`;
      el.innerHTML = html;
      el.style.width = width + 'px';
      const r = VB.rel(root, btn);
      el.style.top = r.bottom + 8 + 'px';
      el.style.left = Math.max(12, Math.min(r.W - width - 12, r.right - width)) + 'px';
      el.style.transformOrigin = `${r.right - parseFloat(el.style.left) - r.width / 2}px 0`;
      host.append(el);
      btn.setAttribute('aria-expanded', 'true');
      open = { el, btn };
      requestAnimationFrame(() => el.classList.add('is-in'));
      return el;
    };
    ctx.listen(root, 'pointerdown', (e) => { if (open && !open.el.contains(e.target) && !open.btn.contains(e.target)) close(); });
    ctx.listen(window, 'keydown', (e) => { if (e.key === 'Escape' && open) { close(true); e.preventDefault(); } });
    return { show, close, get open() { return open; } };
  };

  // Scheduled tasks, shown from the header (P-007). Synthetic, like everything else here.
  VB.tasks = [
    { name: 'RSS sync', state: 'running', note: 'Kestrel, Aperture and Lumen Torznab. Halide resumes at 22:10' },
    { name: 'Refresh series metadata', state: 'running', note: '7 of 12 series' },
    { name: 'Library scan on vault', state: 'next', note: 'In 38 min' },
    { name: 'Back up the library database', state: 'next', note: 'At 03:00' },
  ];
})();
