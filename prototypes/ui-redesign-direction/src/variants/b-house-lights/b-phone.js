// PROTOTYPE. House Lights: the admin phone. Tabs, push navigation, edge-swipe back, action sheet.
(function () {
  const SS = window.SS, VB = window.VB;
  const D = SS.data, F = SS.fmt, ART = SS.art;
  const esc = F.esc;
  const icon = VB.icon;

  const TABS = [['home', 'Tonight'], ['library', 'Library'], ['activity', 'Activity'], ['search', 'Search'], ['more', 'More']];
  const TAB_ICON = { home: icon('clapperboard', 24), library: icon('library-big', 24), activity: icon('activity', 24), search: icon('search', 24), more: icon('ellipsis', 24) };
  const ROOT_TITLE = Object.fromEntries(TABS);
  const project = (v, d = 0.998) => ((v / 1000) * d) / (1 - d);

  VB.phoneShell = function (ctx) {
    const { root } = ctx;
    root.innerHTML = `<div class="b-app b-ph">
      ${TABS.map(([k]) => `<div class="b-tabstack" data-tab="${k}" hidden></div>`).join('')}
      <nav class="b-tabs" aria-label="Sections">${TABS.map(([k, l]) => `<button class="b-tab" type="button" data-tab-go="${k}" aria-label="${l}">${TAB_ICON[k]}<span>${l}</span>${k === 'activity' ? `<i class="b-tab__badge" data-lv="active">${VB.LV.active()}</i>` : ''}${k === 'more' ? '<i class="b-tab__dot" data-moredot hidden></i>' : ''}</button>`).join('')}</nav>
      <div class="b-layer"></div>
    </div>`;
    const live = VB.live(ctx);
    const stacks = Object.fromEntries(TABS.map(([k]) => [k, []]));
    let tab = 'home';
    let anim = null;
    const now = VB.nowShowing(ctx, () => { const p = stacks.home[0]; if (p && tab === 'home' && stacks.home.length === 1) rebuild(p); });
    const box = (k) => root.querySelector(`.b-tabstack[data-tab="${k}"]`);
    const top = () => stacks[tab][stacks[tab].length - 1];
    const scale = () => root.getBoundingClientRect().width / root.offsetWidth || 1;

    // ---------- Pages ----------
    function pageFor(view) {
      const [a, b] = view.split('/');
      if (a === 'work' && D.W[b]) return workPage(D.W[b]);
      if (a === 'requests') return requestsPage();
      if (a === 'nodes') return nodesPage();
      return { home: homePage, library: libraryPage, activity: activityPage, search: searchPage, more: morePage }[a]();
    }
    function mountPage(k, view, { push = false } = {}) {
      const p = pageFor(view);
      const el = document.createElement('section');
      el.className = `b-pg b-pg--${p.kind}`;
      el.dataset.view = view;
      el.innerHTML = `<div class="b-spillhost"></div>
        <header class="b-nb"><div class="b-nb__bg"></div>
          ${push ? `<button class="b-nb__back" type="button" data-back aria-label="Back to ${esc(p.back)}">${icon('chevron-left', 26)}<span>${esc(p.back)}</span></button>` : '<span></span>'}
          <p class="b-nb__t">${esc(p.title)}</p>
          <div class="b-nb__r">${p.right || ''}</div>
        </header>
        <div class="b-scroll">${p.html}</div>`;
      box(k).append(el);
      const spill = VB.spill(el.querySelector('.b-spillhost'));
      spill.set(p.light || null);
      const entry = { view, el, p, spill };
      wirePage(entry);
      live.scan(el);
      VB.fillStills(el);
      return entry;
    }
    function rebuild(entry) {
      const fresh = pageFor(entry.view);
      const sc = entry.el.querySelector('.b-scroll');
      const y = sc.scrollTop;
      sc.innerHTML = fresh.html;
      entry.p = fresh;
      entry.spill.set(fresh.light || null);
      live.scan(entry.el);
      VB.fillStills(entry.el);
      sc.scrollTop = y;
      collapse(entry);
    }
    function wirePage(entry) {
      const sc = entry.el.querySelector('.b-scroll');
      sc.addEventListener('scroll', () => collapse(entry), { passive: true });
      requestAnimationFrame(() => collapse(entry));
    }
    // Large title or Work logo slides under the bar; the bar gains its material and the small title.
    function collapse(entry) {
      const sc = entry.el.querySelector('.b-scroll');
      const nb = entry.el.querySelector('.b-nb');
      const mark = sc.querySelector('[data-collapse]');
      if (!nb.offsetHeight) return;
      const y = sc.scrollTop;
      const start = mark ? mark.offsetTop + mark.offsetHeight - nb.offsetHeight - 8 : 8;
      const p = Math.max(0, Math.min(1, (y - start) / 18));
      if (p !== entry.cp) { nb.style.setProperty('--p', p.toFixed(3)); entry.cp = p; }
    }

    function homePage() {
      const h = now.get();
      const w = D.W[h.work];
      const t = D.targets.find((x) => x.id === h.target);
      const sub = h.episode ? `${F.ep(h.episode)} · ${w.seasons[h.episode[0] - 1].episodes[h.episode[1] - 1].title}` : `${w.year} · ${w.genres[0]} · ${F.runtime(w.runtime)}`;
      const dev = VB.developing();
      return {
        kind: 'root', title: 'Tonight', light: w,
        html: `<section class="b-phero" style="${VB.tint(w)}" data-collapse>
            <img class="b-phero__img" src="${ART.backdrop(w, { w: 786, h: 940 })}" alt="">
            <div class="b-phero__shade"></div>
            <div class="b-phero__body">
              ${VB.logo(w, { maxW: 330, max: 64, min: 30 })}
              <p class="b-meta"><span>${esc(sub)}</span></p>
              <p class="b-phero__arr">Now showing. Arrived <span data-lv="ago|${h.at.getTime()}">${esc(F.ago(h.at))}</span> in ${esc(t.name)}.</p>
              <div class="b-phero__act"><button class="b-btn b-btn--primary" type="button" data-cmd="play" data-work="${w.id}">${icon('play', 18)}Play</button><button class="b-btn b-btn--veil" type="button" data-push="work/${w.id}">Details</button></div>
            </div>
          </section>
          <section class="b-psec"><h2 class="b-ph2">Needs you</h2>${VB.aisleList(VB.aisle(), { cls: 'b-aisle--ph' })}</section>
          <section class="b-psec"><div class="b-psec__h"><h2 class="b-ph2">Developing</h2><span class="b-dim" data-lv="down">${F.rate(SS.sim.totals().down)}</span></div>
            <div class="b-prow">${dev.length ? dev.map(VB.devTile).join('') : '<p class="b-empty">Nothing is developing.</p>'}</div></section>
          <section class="b-psec"><h2 class="b-ph2">Coming soon</h2><div class="b-plist">${VB.comingSoon(4).map(VB.comingCard).join('')}</div></section>
          <p class="b-pnodes">${VB.nodesLine()}</p>`,
      };
    }
    const LIBP = { type: 'movie', filter: 'all' };
    function libraryPage() {
      const all = D.works.filter((w) => w.type === LIBP.type);
      const fl = LIBP.type === 'movie' ? [['all', 'All'], ['downloading', 'Downloading'], ['missing', 'Missing'], ['upgradable', 'Upgradable'], ['unreleased', 'Unreleased'], ['failed', 'Failed']] : [['all', 'All'], ['downloading', 'Downloading'], ['missing', 'Missing episodes']];
      const keep = (w) => LIBP.filter === 'all' || (w.type === 'series' ? (LIBP.filter === 'missing' ? w.counts.missing > 0 : !!VB.acq(w.id)) : VB.status(w).s === LIBP.filter);
      const list = all.filter(keep).sort((a, b) => b.added - a.added);
      return {
        kind: 'root', title: 'Library',
        html: `<h1 class="b-lt" data-collapse>Library</h1>
          <div class="b-pseg" role="tablist" aria-label="Content Type"><button type="button" role="tab" aria-selected="${LIBP.type === 'movie'}" data-libtype="movie">Movies</button><button type="button" role="tab" aria-selected="${LIBP.type === 'series'}" data-libtype="series">Series</button></div>
          <div class="b-chips" role="radiogroup" aria-label="Status">${fl.map(([k, l]) => `<button class="b-chipbtn" type="button" role="radio" aria-checked="${LIBP.filter === k}" data-libfilter="${k}">${l}</button>`).join('')}</div>
          ${list.length ? `<div class="b-pgrid">${list.map((w) => VB.tile(w)).join('')}</div>` : '<p class="b-empty b-empty--page">Nothing matches this filter.</p>'}`,
      };
    }
    function workPage(w) {
      const st = VB.status(w);
      const j = st.j;
      const prof = D.profiles.find((p) => p.id === w.profile);
      const meta = w.type === 'movie' ? [String(w.year), F.runtime(w.runtime), w.cert] : [w.network, F.plural(w.seasons.length, 'season'), w.cert];
      const stateLine = w.type === 'movie'
        ? (j ? `<div class="b-pstate" data-st="${j.state}"><div class="b-pstate__art">${VB.dev(w)}</div><div><p class="b-pstate__k">${j.state === 'failed' ? 'Import failed' : j.state === 'stalled' ? 'Stalled' : j.kind === 'download' ? `Downloading on ${esc(j.node)}` : j.kind === 'transfer' ? 'Transferring to vault' : 'Importing'}</p><p class="b-pstate__v" data-lv="wline|${w.id}">${esc(VB.line(j))}</p>${j.state === 'failed' ? `<p class="b-pstate__v">${esc(j.message)}</p>` : ''}</div></div>` : '')
        : (j ? `<div class="b-pstate"><div><p class="b-pstate__k">${F.ep(j.episode)} downloading</p><p class="b-pstate__v" data-lv="wline|${w.id}|${VB.epKey(j.episode)}">${esc(VB.line(j))}</p></div></div>` : '');
      const cur = VB.local.season?.[w.id] || (w.seasons?.length || 1);
      return {
        kind: 'push', title: w.title, back: '', light: w,
        right: `<button class="b-nb__btn" type="button" data-sheet="${w.id}" aria-label="Actions for ${esc(w.title)}">${icon('ellipsis', 22)}</button>`,
        html: `<div class="b-pw" style="${VB.tint(w)}">
          <div class="b-pw__stage"><img src="${ART.backdrop(w, { w: 786, h: 640 })}" alt=""><div class="b-pw__shade"></div></div>
          <div class="b-pw__id" data-collapse>${VB.logo(w, { maxW: 340, max: 58, min: 28 })}<p class="b-meta">${meta.filter(Boolean).map((m) => `<span>${esc(m)}</span>`).join('')}</p></div>
          <div class="b-pw__act"><button class="b-btn b-btn--primary b-btn--block" type="button" data-cmd="search" data-work="${w.id}">${icon('search', 18)}<span>${w.type === 'series' ? 'Search missing' : 'Search'}</span></button><button class="b-btn b-btn--block" type="button" data-sheet="${w.id}">${icon('ellipsis', 18)}More</button></div>
          ${stateLine}
          <p class="b-pw__ov">${esc(w.overview)}</p>
          ${w.type === 'series' ? `<section class="b-psec b-psec--flush"><div class="b-psec__h"><h2 class="b-ph2">Seasons</h2></div><div class="b-chips" role="tablist" aria-label="Season">${w.seasons.map((s) => `<button class="b-chipbtn" type="button" role="tab" aria-selected="${s.n === cur}" data-pseason="${s.n}">Season ${s.n}</button>`).join('')}</div><div data-eps>${VB.episodes(w, cur)}</div></section>` : ''}
          <section class="b-psec b-psec--flush"><div class="b-psec__h"><h2 class="b-ph2">Targets</h2><span class="b-dim">${esc(prof.name)} Profile</span></div>${VB.covMatrix(w)}</section>
          ${w.type === 'movie' ? `<section class="b-psec b-psec--flush"><h2 class="b-ph2">Files</h2>${VB.fileList(w)}</section>` : ''}
        </div>`,
      };
    }
    function activityPage() {
      const needs = D.jobs.filter((j) => (j.state === 'failed' || j.state === 'stalled') && !VB.local.resolved.has(j.id));
      const run = D.jobs.filter((j) => j.state === 'running');
      const q = D.jobs.filter((j) => j.state === 'queued');
      const g = (l, js) => (js.length ? `<section class="b-psec b-psec--flush"><h2 class="b-ph2">${l} <span class="b-dim">${js.length}</span></h2><div class="b-pjobs">${js.map(VB.jobRow).join('')}</div></section>` : '');
      return {
        kind: 'root', title: 'Activity',
        html: `<h1 class="b-lt" data-collapse>Activity</h1>
          <p class="b-psum"><span data-lv="active">${run.length}</span> running · <span data-lv="down">${F.rate(SS.sim.totals().down)}</span> down</p>
          ${g('Needs you', needs)}${g('Running', run)}${g('Queued', q)}
          <section class="b-psec b-psec--flush"><h2 class="b-ph2">Recently</h2><ol class="b-hist">${D.history.slice(0, 6).map((x) => `<li><span>${esc(D.W[x.work].title)}${x.episode ? ' ' + F.ep(x.episode) : ''}</span><span class="b-dim">${esc(x.text)} · ${esc(F.ago(x.at))}</span></li>`).join('')}</ol></section>`,
      };
    }
    const SRCH = { q: '' };
    function searchResults(q) {
      q = q.trim().toLowerCase();
      if (!q) return `<section class="b-psec b-psec--flush"><h2 class="b-ph2">Developing</h2><ul class="b-results" role="list">${VB.developing().slice(0, 4).map((x) => resultRow(x.w)).join('')}</ul></section>
        <section class="b-psec b-psec--flush"><h2 class="b-ph2">Missing</h2><ul class="b-results" role="list">${D.works.filter((w) => VB.status(w).s === 'missing' && w.type === 'movie').map(resultRow).join('')}</ul></section>`;
      const hits = D.works.filter((w) => w.title.toLowerCase().includes(q));
      return hits.length ? `<ul class="b-results" role="list">${hits.map(resultRow).join('')}</ul>` : `<p class="b-empty">No Work in the library matches “${esc(q)}”. Adding new Works from search is coming to the phone.</p>`;
    }
    const resultRow = (w) => `<li><button class="b-result" type="button" data-push="work/${w.id}" style="${VB.tint(w)}"><span class="b-result__art">${VB.art(w)}</span><span class="b-result__t">${esc(w.title)}<span class="b-dim">${w.year} · ${w.type === 'movie' ? 'Movie' : 'Series'}</span></span><span class="b-result__s">${VB.statusLabel[VB.status(w).s]}</span></button></li>`;
    function searchPage() {
      return {
        kind: 'root', title: 'Search',
        html: `<h1 class="b-lt" data-collapse>Search</h1>
          <label class="b-psearch">${icon('search', 18)}<span class="b-sr">Search the library</span><input type="search" enterkeyhint="search" placeholder="Movies and series" value="${esc(SRCH.q)}" data-psearch autocomplete="off"></label>
          <div data-results>${searchResults(SRCH.q)}</div>`,
      };
    }
    function morePage() {
      const pend = VB.reqs().filter((r) => VB.reqState(r) === 'pending').length;
      const missM = D.works.filter((w) => w.type === 'movie' && VB.status(w).s === 'missing').length;
      const missE = D.works.filter((w) => w.type === 'series').reduce((s, w) => s + w.counts.missing, 0);
      const row = (ic, l, v, go, tone) => `<li><button class="b-mrow" type="button" ${go ? `data-push="${go}"` : 'data-cmd="noop"'}>${ic}<span class="b-mrow__l">${l}</span><span class="b-mrow__v${tone ? ' b-' + tone : ''}">${v}</span>${icon('chevron-right', 18)}</button></li>`;
      return {
        kind: 'root', title: 'More',
        html: `<h1 class="b-lt" data-collapse>More</h1>
          <section class="b-mgroup"><h2 class="b-mgroup__h">Discover</h2><ul role="list">
            ${row(icon('ticket', 20), 'Requests', pend ? `${pend} waiting` : 'None waiting', 'requests', pend ? 'amber' : '')}
            ${row(icon('calendar-days', 20), 'Coming soon', `${D.calendar.length} this week`)}
            ${row(icon('circle-dashed', 20), 'Missing', `${missM} movies, ${missE} episodes`)}
          </ul></section>
          <section class="b-mgroup"><h2 class="b-mgroup__h">Settings</h2><ul role="list">
            ${row(icon('server', 20), 'Nodes', '1 offline', 'nodes')}
            ${row(icon('target', 20), 'Targets and Profiles', `${D.targets.length} Targets`)}
            ${row(icon('radar', 20), 'Indexers', 'Halide limited')}
            ${row(icon('download', 20), 'Download Clients', 'Built-in')}
            ${row(icon('monitor-play', 20), 'Media Servers', 'Plex, 2 streams')}
            ${row(icon('users', 20), 'Accounts', `${D.accounts.length} people`)}
          </ul></section>
          <section class="b-mgroup"><h2 class="b-mgroup__h">System</h2><ul role="list">
            ${row(icon('heart-pulse', 20), 'Health', '2 issues')}
            ${row(icon('list-checks', 20), 'Scheduled tasks', '2 running')}
            ${row(icon('scroll-text', 20), 'Logs', '')}
            ${row(icon('arrow-up', 20), 'Updates', '2.3.1 ready')}
          </ul></section>
          <p class="b-foot">SlipStream 2.3.0 · Controller atlas<br>Prototype with synthetic data</p>`,
      };
    }
    function requestsPage() {
      const list = VB.reqs().slice().sort((a, b) => (VB.reqState(a) === 'pending' ? -1 : 0) - (VB.reqState(b) === 'pending' ? -1 : 0) || b.at - a.at);
      return { kind: 'push', title: 'Requests', back: 'More', html: `<h1 class="b-lt" data-collapse>Requests</h1><div class="b-preqs">${list.map(VB.reqRow).join('')}</div>` };
    }
    function nodesPage() {
      return { kind: 'push', title: 'Nodes', back: 'More', html: `<h1 class="b-lt" data-collapse>Nodes</h1><div class="b-pnodelist">${D.nodes.map(VB.nodeRow).join('')}</div>` };
    }

    // ---------- Stack ----------
    function showTab(k, { reset = false } = {}) {
      if (k === tab && reset && stacks[k].length > 1) { while (stacks[k].length > 1) stacks[k].pop().el.remove(); placeTop(); syncView(); return; }
      tab = k;
      for (const [t] of TABS) box(t).hidden = t !== k;
      if (!stacks[k].length) stacks[k].push(mountPage(k, k));
      for (const b of root.querySelectorAll('[data-tab-go]')) b.toggleAttribute('aria-current', b.dataset.tabGo === k);
      placeTop();
      collapse(top());
      syncView();
    }
    function placeTop() {
      const s = stacks[tab];
      s.forEach((e, i) => {
        e.el.style.transform = '';
        e.el.hidden = i < s.length - 1;
        e.el.classList.toggle('is-under', false);
      });
    }
    function syncView() { const t = top(); ctx.setView(t.view === 'home' ? 'home' : t.view); }
    function push(view, kbd) {
      if (anim) return;
      const s = stacks[tab];
      const prev = s[s.length - 1];
      const entry = mountPage(tab, view, { push: true });
      entry.p.back = entry.p.back || prev.p.title;
      const bk = entry.el.querySelector('.b-nb__back span');
      if (bk) { bk.textContent = prev.p.title; entry.el.querySelector('.b-nb__back').setAttribute('aria-label', `Back to ${prev.p.title}`); }
      s.push(entry);
      syncView();
      if (kbd || VB.reduced()) { placeTop(); return; }
      prev.el.hidden = false;
      prev.el.classList.add('is-under');
      anim = Promise.all([
        VB.springTo(entry.el, 'translateX(100%)', 'translateX(0)', { response: 0.5 }),
        VB.springTo(prev.el, 'translateX(0)', 'translateX(-30%)', { response: 0.5 }),
      ]).then(() => { anim = null; placeTop(); });
    }
    function pop(kbd, v0 = 0, fromX = 0) {
      const s = stacks[tab];
      if (s.length < 2) return;
      const cur = s.pop();
      const prev = s[s.length - 1];
      syncView();
      if (kbd || VB.reduced()) { cur.el.remove(); placeTop(); collapse(prev); return; }
      prev.el.hidden = false;
      prev.el.classList.add('is-under');
      collapse(prev);
      const W = root.offsetWidth;
      const rel = W - fromX ? v0 / (W - fromX) : 0;
      anim = Promise.all([
        VB.springTo(cur.el, `translateX(${fromX}px)`, `translateX(${W}px)`, { response: 0.42, v0: rel }),
        VB.springTo(prev.el, `translateX(${-0.3 * W + 0.3 * fromX}px)`, 'translateX(0)', { response: 0.42, v0: rel }),
      ]).then(() => { anim = null; cur.el.remove(); placeTop(); });
    }

    // Edge swipe: the page follows the finger 1:1, then a spring takes the release velocity home.
    let drag = null;
    ctx.listen(root, 'pointerdown', (e) => {
      if (anim || stacks[tab].length < 2 || sheet) return;
      const r = VB.rel(root, root);
      const x = (e.clientX - root.getBoundingClientRect().left) / scale();
      if (x > 22 || e.button !== 0) return;
      const cur = top(), prev = stacks[tab][stacks[tab].length - 2];
      drag = { id: e.pointerId, x0: e.clientX, cur, prev, W: r.W, hist: [], on: false };
    });
    ctx.listen(root, 'pointermove', (e) => {
      if (!drag || e.pointerId !== drag.id) return;
      const dx = Math.max(0, (e.clientX - drag.x0) / scale());
      if (!drag.on) {
        if (dx < 8) return;
        drag.on = true;
        root.setPointerCapture?.(e.pointerId);
        drag.prev.el.hidden = false;
        drag.prev.el.classList.add('is-under');
        collapse(drag.prev);
      }
      drag.dx = dx;
      drag.hist.push([e.timeStamp, dx]);
      if (drag.hist.length > 6) drag.hist.shift();
      drag.cur.el.style.transform = `translateX(${dx}px)`;
      drag.prev.el.style.transform = `translateX(${-0.3 * drag.W + 0.3 * dx}px)`;
    });
    const endDrag = (e) => {
      if (!drag || e.pointerId !== drag.id) return;
      const d = drag;
      drag = null;
      if (!d.on) return;
      const h = d.hist, a = h[0], b = h[h.length - 1];
      const v = b && a && b[0] > a[0] ? ((b[1] - a[1]) / (b[0] - a[0])) * 1000 : 0;
      const land = d.dx + project(v);
      if (land > d.W / 2) {
        const s = stacks[tab];
        s.pop();
        syncView();
        const rel = v / Math.max(1, d.W - d.dx);
        anim = Promise.all([
          VB.springTo(d.cur.el, `translateX(${d.dx}px)`, `translateX(${d.W}px)`, { response: 0.4, v0: rel }),
          VB.springTo(d.prev.el, `translateX(${-0.3 * d.W + 0.3 * d.dx}px)`, 'translateX(0)', { response: 0.4, v0: rel }),
        ]).then(() => { anim = null; d.cur.el.remove(); placeTop(); });
      } else {
        const rel = d.dx ? -v / d.dx : 0;
        anim = Promise.all([
          VB.springTo(d.cur.el, `translateX(${d.dx}px)`, 'translateX(0)', { response: 0.4, v0: rel }),
          VB.springTo(d.prev.el, `translateX(${-0.3 * d.W + 0.3 * d.dx}px)`, 'translateX(-30%)', { response: 0.4, v0: rel }),
        ]).then(() => { anim = null; placeTop(); });
      }
    };
    ctx.listen(root, 'pointerup', endDrag);
    ctx.listen(root, 'pointercancel', endDrag);

    // ---------- Action sheet: drags to dismiss and keeps the finger's velocity ----------
    let sheet = null;
    function openSheet(id, kbd) {
      if (sheet) return;
      const w = D.W[id];
      const mon = VB.local.monitored[w.id] ?? true;
      const layer = root.querySelector('.b-layer');
      layer.insertAdjacentHTML('beforeend', `<div class="b-sheetwrap">
        <div class="b-scrim" data-sheetclose></div>
        <div class="b-sheet" role="dialog" aria-modal="true" aria-label="Actions for ${esc(w.title)}" style="${VB.tint(w)}">
          <div class="b-sheet__grab" aria-hidden="true"></div>
          <div class="b-sheet__head"><span class="b-sheet__art">${VB.art(w)}</span><div><p class="b-sheet__t">${esc(w.title)}</p><p class="b-dim">${w.year} · ${VB.statusLabel[VB.status(w).s]}</p></div></div>
          <div class="b-sheet__g">
            <button class="b-srow" type="button" data-scmd="search">${icon('search', 20)}${w.type === 'series' ? 'Search for missing episodes' : 'Search for a better release'}</button>
            <button class="b-srow" type="button" data-scmd="manual">${icon('list', 20)}Manual search</button>
            <button class="b-srow" type="button" role="switch" aria-checked="${mon}" data-scmd="monitor">${icon('eye', 20)}Monitored<span class="b-switch__t" aria-hidden="true"></span></button>
            <button class="b-srow" type="button" data-scmd="refresh">${icon('refresh-cw', 20)}Refresh metadata</button>
            <button class="b-srow" type="button" data-scmd="profile">${icon('target', 20)}Change Profile</button>
          </div>
          <div class="b-sheet__g"><button class="b-srow" type="button" data-scmd="delete">${icon('trash-2', 20)}Delete ${esc(w.title)}</button></div>
          <button class="b-btn b-btn--block b-sheet__cancel" type="button" data-sheetclose>Cancel</button>
        </div></div>`);
      const wrap = layer.lastElementChild;
      const el = wrap.querySelector('.b-sheet');
      const scrim = wrap.querySelector('.b-scrim');
      sheet = { wrap, el, scrim, id };
      ctx.setView(`work/${id}/actions`);
      if (kbd || VB.reduced()) { scrim.style.opacity = '1'; return; }
      scrim.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 300, easing: 'ease-out' });
      VB.springTo(el, 'translateY(100%)', 'translateY(0)', { response: 0.45 });
      el.querySelector('button').focus({ preventScroll: true });
    }
    function closeSheet(v = 0, from = 0) {
      if (!sheet) return;
      const s = sheet;
      sheet = null;
      ctx.setView(`work/${s.id}`);
      if (VB.reduced()) { s.wrap.remove(); return; }
      const H = s.el.offsetHeight;
      s.scrim.animate([{ opacity: getComputedStyle(s.scrim).opacity }, { opacity: 0 }], { duration: 260, easing: 'ease-out', fill: 'forwards' });
      VB.springTo(s.el, `translateY(${from}px)`, `translateY(${H + 20}px)`, { response: 0.38, v0: v / Math.max(1, H - from) }).then(() => s.wrap.remove());
    }
    let sdrag = null;
    ctx.listen(root, 'pointerdown', (e) => {
      if (!sheet || !sheet.el.contains(e.target)) return;
      sdrag = { id: e.pointerId, y0: e.clientY, base: VB.ty(sheet.el), hist: [], on: false, dy: 0 };
    });
    ctx.listen(root, 'pointermove', (e) => {
      if (!sdrag || e.pointerId !== sdrag.id || !sheet) return;
      let dy = (e.clientY - sdrag.y0) / scale() + sdrag.base;
      if (!sdrag.on) {
        if (Math.abs(dy - sdrag.base) < 8) return;
        sdrag.on = true;
        sheet.el.getAnimations().forEach((a) => a.cancel());
        sheet.el.setPointerCapture?.(e.pointerId);
      }
      if (dy < 0) dy = -((-dy * 0.55 * 300) / (300 + 0.55 * -dy));
      sdrag.dy = dy;
      sdrag.hist.push([e.timeStamp, dy]);
      if (sdrag.hist.length > 6) sdrag.hist.shift();
      sheet.el.style.transform = `translateY(${dy}px)`;
      sheet.scrim.style.opacity = String(Math.max(0, 1 - dy / sheet.el.offsetHeight));
    });
    const endSheet = (e) => {
      if (!sdrag || e.pointerId !== sdrag.id) return;
      const d = sdrag;
      sdrag = null;
      if (!d.on || !sheet) return;
      const h = d.hist, a = h[0], b = h[h.length - 1];
      const v = b && a && b[0] > a[0] ? ((b[1] - a[1]) / (b[0] - a[0])) * 1000 : 0;
      const H = sheet.el.offsetHeight;
      if (d.dy + project(v, 0.99) > H * 0.4 || v > 900) closeSheet(v, d.dy);
      else {
        sheet.scrim.animate([{ opacity: sheet.scrim.style.opacity }, { opacity: 1 }], { duration: 300, easing: 'ease-out' });
        sheet.scrim.style.opacity = '1';
        VB.springTo(sheet.el, `translateY(${d.dy}px)`, 'translateY(0)', { response: 0.36, damping: Math.abs(v) > 300 ? 0.8 : 1, v0: d.dy ? -v / d.dy : 0 });
      }
    };
    ctx.listen(root, 'pointerup', endSheet);
    ctx.listen(root, 'pointercancel', endSheet);

    // ---------- Events ----------
    ctx.listen(root, 'click', (e) => {
      const kbd = e.detail === 0;
      const t = e.target.closest('[data-tab-go],[data-push],[data-open],[data-go],[data-back],[data-sheet],[data-sheetclose],[data-scmd],[data-cmd],[data-libtype],[data-libfilter],[data-pseason],[data-aisle-act],[data-req-approve],[data-req-deny],[data-req-cancel],[data-req-confirm]');
      if (!t) return;
      if (t.closest('a[href]')) e.preventDefault();
      const d = t.dataset;
      if (d.tabGo) return showTab(d.tabGo, { reset: true });
      if (d.push) return push(d.push, kbd);
      if (d.open) return push(`work/${d.open}`, kbd);
      if (d.go) return push(d.go, kbd);
      if ('back' in d) return pop(kbd);
      if (d.sheet) return openSheet(d.sheet, kbd);
      if ('sheetclose' in d) return closeSheet();
      if (d.scmd) {
        const w = D.W[sheet?.id];
        if (d.scmd === 'monitor') { const on = t.getAttribute('aria-checked') !== 'true'; t.setAttribute('aria-checked', String(on)); VB.local.monitored[w.id] = on; return; }
        closeSheet();
        VB.toast(root, { search: `Searching 4 indexers for ${w.title}`, manual: 'Manual search opens on the desk in this prototype', refresh: `Refreshing metadata for ${w.title}`, profile: 'Profiles: Cinema, Standard, Family', delete: 'Delete is disabled in the prototype' }[d.scmd]);
        return;
      }
      if (d.cmd) {
        const w = d.work ? D.W[d.work] : null;
        if (d.cmd === 'play') return VB.toast(root, `Opening ${w.title} on Plex`);
        if (d.cmd === 'search') {
          if (t.disabled) return;
          t.disabled = true; t.setAttribute('aria-busy', 'true');
          setTimeout(() => { t.disabled = false; t.removeAttribute('aria-busy'); VB.toast(root, `Searched 4 indexers for ${w.title}`); }, 1400);
          return;
        }
        if (d.cmd === 'blocklist') { VB.local.resolved.add(d.job); VB.toast(root, 'Blocklisted. Searching again'); return rebuild(top()); }
        if (d.cmd === 'another') return push(`work/${VB.job(d.job).work}`, kbd);
        return VB.toast(root, 'Opens in Settings on the desk in this prototype');
      }
      if (d.libtype) { LIBP.type = d.libtype; LIBP.filter = 'all'; return rebuild(top()); }
      if (d.libfilter) { LIBP.filter = d.libfilter; return rebuild(top()); }
      if (d.pseason) {
        const w = D.W[top().view.split('/')[1]];
        (VB.local.season ||= {})[w.id] = +d.pseason;
        for (const b of top().el.querySelectorAll('[data-pseason]')) b.setAttribute('aria-selected', String(b === t));
        const bx = top().el.querySelector('[data-eps]');
        bx.innerHTML = VB.episodes(w, +d.pseason);
        live.scan(bx); VB.fillStills(bx);
        return;
      }
      if (d.aisleAct) {
        const item = VB.aisle().find((x) => x.id === d.aisleAct);
        if (!item) return;
        if (item.cmd === 'blocklist') { VB.local.resolved.add(item.job); VB.toast(root, 'Blocklisted the sample. Searching again'); return rebuild(top()); }
        if (item.go === 'requests' || item.go === 'nodes') { showTab('more'); return push(item.go, kbd); }
        return push(item.job ? `work/${VB.job(item.job).work}` : item.go, kbd);
      }
      const row = t.closest('.b-req');
      if (d.reqApprove) { VB.local.req[d.reqApprove] = 'approved'; VB.toast(root, 'Approved. Searching now'); return rebuild(top()); }
      if (d.reqDeny) { row.querySelector('.b-req__deny').hidden = false; row.querySelector('.b-req__act').hidden = true; return; }
      if (d.reqCancel) { row.querySelector('.b-req__deny').hidden = true; row.querySelector('.b-req__act').hidden = false; return; }
      if (d.reqConfirm) { VB.local.req[d.reqConfirm] = 'denied'; VB.toast(root, 'Denied'); return rebuild(top()); }
    });
    ctx.listen(root, 'input', (e) => {
      if (!e.target.matches('[data-psearch]')) return;
      SRCH.q = e.target.value;
      const r = top().el.querySelector('[data-results]');
      r.innerHTML = searchResults(SRCH.q);
      live.scan(r);
    });
    ctx.listen(window, 'keydown', (e) => {
      if (e.key !== 'Escape' || e.defaultPrevented) return;
      if (sheet) return closeSheet();
      if (!e.target.closest('input') && stacks[tab].length > 1) pop(true);
    });
    const moreDot = () => { root.querySelector('[data-moredot]').hidden = !VB.reqs().some((r) => VB.reqState(r) === 'pending'); };
    ctx.on('second', () => { root.querySelector('.b-tab__badge').textContent = VB.LV.active(); });
    ctx.on('job:done', (j) => {
      if (j.kind !== 'import') return;
      setTimeout(() => { const t = top(); if (t && (t.view === 'home' || t.view === 'activity')) rebuild(t); }, 4200);
    });
    ctx.on('job:new', () => { const t = top(); if (t && t.view === 'activity') rebuild(t); });

    // ---------- Deep link ----------
    const v = ctx.view || 'home';
    const [a, b, c] = v.split('/');
    if (a === 'work' && D.W[b]) {
      showTab('library');
      stacks.library.push(mountPage('library', `work/${b}`, { push: true }));
      const e = stacks.library[1];
      e.el.querySelector('.b-nb__back span').textContent = 'Library';
      placeTop();
      syncView();
      if (c === 'actions') openSheet(b, true);
    } else if (a === 'requests' || a === 'nodes') {
      showTab('more');
      stacks.more.push(mountPage('more', a, { push: true }));
      placeTop();
      syncView();
    } else showTab(TABS.some((t) => t[0] === a) ? a : 'home');
    moreDot();
    ctx.on('request:update', moreDot);
    return { unmount() {} };
  };
})();
