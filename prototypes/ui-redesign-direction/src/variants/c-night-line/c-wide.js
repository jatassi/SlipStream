(function () {
  'use strict';
  const SS = window.SS;
  const d = SS.data;
  const F = SS.fmt;
  const esc = F.esc;
  const VC = window.VC;
  const I = (name, size = 16) => SS.icon(name, { size, stroke: 1.75 });
  const $ = (el, s) => el.querySelector(s);
  const $$ = (el, s) => [...el.querySelectorAll(s)];

  const store = {
    get(k, def) { try { return JSON.parse(localStorage.getItem('c-' + k)) ?? def; } catch { return def; } },
    set(k, v) { try { localStorage.setItem('c-' + k, JSON.stringify(v)); } catch { /* storage blocked */ } },
  };

  const toneOf = (j) => ({ failed: 'failed', stalled: 'delays', paused: 'paused', queued: 'muted', done: 'arrived' })[j.state] || '';
  const routeText = (j) => VC.jobPath(j).map((n) => `<span class="c-stn">${esc(n)}</span>`).join(`<span class="c-arrow">${I('arrow-right', 12)}</span>`);
  const stateNote = (j) => ({ stalled: 'Stalled', paused: 'Paused', queued: 'Waiting', done: 'Arrived' })[j.state] || '';
  const ORDER = { running: 0, paused: 1, stalled: 2, queued: 3, failed: 4, done: 5 };
  const sortJobs = (list) => list.sort((a, b) => VC.LINES.findIndex((l) => l.id === VC.jobLine(a)) - VC.LINES.findIndex((l) => l.id === VC.jobLine(b)) || ORDER[a.state] - ORDER[b.state]);

  function jobActions(j) {
    if (j.state === 'failed') return `<button class="c-btn c-btn--s" data-act="blocklist" data-work="${j.work}">Blocklist and search</button>`;
    if (j.state === 'stalled') return `<button class="c-btn c-btn--s" data-act="find-release" data-work="${j.work}">Find another release</button>`;
    if (j.state === 'paused') return `<button class="c-icon-btn" data-act="resume-node" data-node="${j.node}" aria-label="Resume ${esc(j.node)}">${I('play', 15)}</button>`;
    if (j.state === 'running' && j.node === 'ams-1') return `<button class="c-icon-btn" data-act="pause-node" data-node="${j.node}" aria-label="Pause ams-1">${I('pause', 15)}</button>`;
    return '';
  }

  function jobRow(j, { actions = false } = {}) {
    const w = d.W[j.work];
    const line = VC.jobLine(j);
    const lit = VC.lit.has(j.id);
    const progress = j.state === 'failed'
      ? `<span class="c-msg c-tone-failed" title="${esc(j.message)}">${esc(j.message)}</span>`
      : `${VC.bar(j.progress, line)}<span class="c-num c-pct" data-f="pct">${F.pct(j.progress)}</span>`;
    return `<tr class="c-row" data-job="${j.id}" data-state="${j.state}" data-go="work/${w.id}" tabindex="0"${lit ? ` data-lit="${j.id}"` : ''}>
      <td class="c-td-b">${VC.bullet(line)}</td>
      <td class="c-td-w"><span class="c-trunc"><b>${esc(w.title)}</b>${j.episode ? ` <span class="c-dim">${F.ep(j.episode)}</span>` : ''}</span></td>
      <td class="c-td-s"><span class="c-stage c-tone-${toneOf(j)}" data-f="stage">${esc(VC.jobStage(j))}</span></td>
      <td class="c-td-route">${routeText(j)}</td>
      <td class="c-td-p">${progress}</td>
      <td class="c-num c-td-r" data-f="rate">${VC.jobRate(j)}</td>
      <td class="c-num c-td-r" data-f="eta">${VC.jobEta(j) || stateNote(j)}</td>
      <td class="c-td-t">${esc(VC.target(j.target)?.short || '')}</td>
      ${actions ? `<td class="c-td-a">${jobActions(j)}</td>` : ''}
    </tr>`;
  }
  const JOB_HEAD = (actions) => `<thead><tr><th class="c-td-b"><span class="c-vh">Line</span></th><th>Work</th><th>Stage</th><th class="c-td-route">Route</th><th class="c-td-p">Progress</th><th class="c-td-r">Rate</th><th class="c-td-r">Arrives in</th><th>Target</th>${actions ? '<th class="c-td-a"><span class="c-vh">Actions</span></th>' : ''}</tr></thead>`;

  function liveRows(container) {
    const rows = new Map();
    const scan = () => {
      rows.clear();
      for (const tr of $$(container, 'tr[data-job]')) {
        const j = d.jobs.find((x) => x.id === tr.dataset.job);
        if (!j) continue;
        rows.set(j.id, { tr, j, fill: $(tr, '.c-bar__fill'), pct: $(tr, '[data-f="pct"]'), rate: $(tr, '[data-f="rate"]'), eta: $(tr, '[data-f="eta"]'), stage: $(tr, '[data-f="stage"]') });
      }
    };
    scan();
    return {
      scan,
      tick() {
        for (const r of rows.values()) if (r.fill && r.j.state === 'running') r.fill.style.transform = `scaleX(${r.j.progress.toFixed(4)})`;
      },
      second() {
        for (const r of rows.values()) {
          VC.set(r.pct, F.pct(r.j.progress));
          VC.set(r.rate, VC.jobRate(r.j));
          VC.set(r.eta, VC.jobEta(r.j) || stateNote(r.j));
          VC.flap(r.stage, VC.jobStage(r.j));
        }
      },
    };
  }

  function coverageCell(c) {
    const s = c.state;
    if (!c.inProfile) return `<td class="c-cov__c" data-s="unwanted"><span class="c-dim">Not in Profile</span></td>`;
    const body = s === 'moving'
      ? `${VC.bar(c.p, VC.jobLine(c.job))}<span class="c-num" data-f="cov-p">${VC.jobVerb(c.job)} ${F.pct(c.job.progress)}</span>`
      : {
        met: `<span class="c-cov__mark c-cov__mark--met">${I('check', 13)}</span>Met`,
        queued: '<span class="c-cov__mark c-cov__mark--wait"></span>Waiting',
        waiting: '<span class="c-cov__mark c-cov__mark--derive"></span>Derives after download',
        upgradable: `<span class="c-cov__mark c-cov__mark--up">${I('arrow-up', 12)}</span>Upgradable`,
        missing: '<span class="c-cov__mark c-cov__mark--miss"></span>Missing',
        unreleased: '<span class="c-cov__mark c-cov__mark--wait"></span>Unreleased',
      }[s] || esc(s);
    return `<td class="c-cov__c" data-s="${s}">${body}</td>`;
  }

  function routeHTML(w) {
    const r = VC.route(w);
    const head = `<header class="c-panel__h"><h2>Route</h2>${r.context ? `<span class="c-panel__meta">${esc(r.context)}</span>` : ''}</header>`;
    return `${head}<ol class="c-route">${r.stops.map((s, i) => `
      <li class="c-stop" data-state="${s.state}" style="--ink:${VC.L[s.line].ink}" data-stop="${s.id}">
        ${i < r.stops.length - 1 ? `<span class="c-stop__rail" data-seg="${r.stops[i + 1].state}" style="--ink:${VC.L[r.stops[i + 1].line].ink}"></span>` : ''}
        <span class="c-stop__dot">${s.state === 'current' || s.state === 'delayed' || s.state === 'paused' ? '<span class="c-stop__train"></span>' : ''}</span>
        <span class="c-stop__b"><span class="c-stop__l">${esc(s.label)}</span><span class="c-stop__d" data-f="detail">${esc(s.detail || '')}</span></span>
        <span class="c-stop__t c-num" data-f="time">${s.at ? `${s.est ? '<span class="c-approx">about</span> ' : ''}${VC.timeOrDay(s.at)}` : ''}</span>
      </li>`).join('')}</ol>`;
  }

  function releasesFor(w) {
    const base = w.title.replace(/[^A-Za-z0-9]+/g, '.');
    const groups = ['FRAMESTOR', 'HONE', 'BYNDR', 'LUMiNA', 'SPiRiT', 'HDO'];
    const idx = ['Halide', 'Kestrel', 'Aperture', 'Lumen Torznab'];
    const rows = [
      ['2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC.TrueHD.Atmos.7.1', 'uhd', 61.2e9, 212, true, ''],
      ['2160p.WEB-DL.DV.HDR.DDP5.1.Atmos.H.265', 'uhd', 21.4e9, 640, true, ''],
      ['1080p.BluRay.x264.DTS-HD.MA.5.1', 'hd', 14.8e9, 388, true, ''],
      ['1080p.WEB-DL.DDP5.1.H.264', 'hd', 7.1e9, 1204, true, ''],
      ['720p.WEBRip.x264.AAC', 'mobile', 1.4e9, 96, false, 'Below the 1080p cutoff'],
      ['1080p.HDTV.x264', 'hd', 2.2e9, 3, false, 'On the Blocklist'],
    ];
    return rows.map(([q, t, size, seeds, ok, why], i) => ({ name: `${base}.${w.year}.${q}-${groups[i]}`, target: t, size, seeds, ok, why, indexer: idx[i % idx.length], age: `${2 + i * 3} days` }));
  }

  VC.routeHTML = routeHTML;
  VC.releasesFor = releasesFor;

  VC.wide = (ctx) => {
    const root = ctx.root;
    root.classList.add('c-app', 'c-app--wide');
    let view = null;
    let route = '';
    let pop = null;
    let palette = null;

    root.innerHTML = `
      <aside class="c-side" aria-label="Sidebar">
        <div class="c-side__brand">${VC.logo(22)}<span>SlipStream</span></div>
        <nav class="c-side__nav" aria-label="Main">
          ${[['home', 'route', 'Network', 'net'], ['library', 'film', 'Movies', 'movies'], ['series', 'tv', 'Series', 'series'], ['activity', 'activity', 'Activity', 'activity'], ['requests', 'inbox', 'Requests', 'requests'], ['nodes', 'server', 'Nodes', 'nodes']]
            .map(([go, icon, label, key]) => `<a class="c-nav" href="#" data-go="${go}" data-nav="${go}">${SS.icon(icon, { size: 17, stroke: 1.75 })}<span>${label}</span><span class="c-nav__n c-num" data-count="${key}"></span></a>`).join('')}
        </nav>
        <div class="c-side__lines">
          <h2 class="c-side__h">Lines</h2>
          <ul>${VC.LINES.map((l) => `<li><button class="c-lineitem" data-line-btn="${l.id}">${VC.bullet(l.id)}<span class="c-lineitem__n">${l.name}</span><span class="c-lineitem__s" data-line-state="${l.id}"></span></button></li>`).join('')}</ul>
        </div>
        <div class="c-side__me">${VC.avatar('alex')}<span><b>Alex Rowan</b><span>Admin · atlas 2.3.0</span></span></div>
      </aside>
      <header class="c-head">
        <div class="c-head__title" data-title></div>
        <button class="c-search" data-act="palette" aria-label="Search Works, Nodes and commands">${I('search', 15)}<span>Search Works, Nodes and commands</span><kbd>⌘K</kbd></button>
        <div class="c-head__right">
          <button class="c-headbtn" data-pop="running" aria-label="Running work">${SS.icon('train-front', { size: 16, stroke: 1.75 })}<span class="c-num" data-h="running"></span><span class="c-head__rate c-num" data-h="rate"></span></button>
          <button class="c-pill" data-pop="disruptions" data-h="pill"></button>
          <button class="c-icon-btn c-bell" data-pop="bell" aria-label="Notifications">${I('bell', 17)}<span class="c-bell__dot" data-h="bell"></span></button>
        </div>
      </header>
      <main class="c-main" tabindex="-1"></main>`;
    const main = $(root, '.c-main');

    const VIEWS = { home: homeView, library: (el) => libraryView(el, 'movie'), series: (el) => libraryView(el, 'series'), work: workView, activity: activityView, nodes: nodesView, requests: requestsView };
    const TITLES = { home: 'Network', library: 'Movies', series: 'Series', activity: 'Activity', nodes: 'Nodes', requests: 'Requests' };

    function go(to, { transition = null } = {}) {
      closePop();
      const [name, ...rest] = (to || 'home').split('/');
      const key = VIEWS[name] ? name : 'home';
      const arg = rest.join('/');
      if (key === 'work' && !d.W[arg]) return go('library');
      const render = () => {
        view?.destroy?.();
        main.innerHTML = '';
        route = key === 'home' ? 'home' : [key, arg].filter(Boolean).join('/');
        view = VIEWS[key](main, arg) || {};
        ctx.setView(route);
        const w = key === 'work' ? d.W[arg] : null;
        $(root, '[data-title]').textContent = w ? w.title : TITLES[key];
        $$(root, '[data-nav]').forEach((a) => a.toggleAttribute('aria-current', a.dataset.nav === (w ? (w.type === 'movie' ? 'library' : 'series') : key)));
        main.scrollTop = 0;
      };
      if (transition && document.startViewTransition && !VC.reduced()) {
        transition.before?.();
        const vt = document.startViewTransition(() => { render(); transition.after?.(); });
        vt.finished.finally(() => transition.done?.());
      } else render();
    }
    VC.go = go;

    function shellSecond() {
      const net = VC.net();
      const counts = {
        net: '',
        movies: d.works.filter((w) => w.type === 'movie').length,
        series: d.works.filter((w) => w.type === 'series').length,
        activity: d.jobs.filter((j) => j.state === 'running').length,
        requests: d.requests.filter((r) => r.state === 'pending').length,
        nodes: d.nodes.length,
      };
      for (const [k, v] of Object.entries(counts)) VC.set($(root, `[data-count="${k}"]`), String(v));
      for (const l of VC.LINES) {
        const el = $(root, `[data-line-state="${l.id}"]`);
        VC.flap(el, net.lines[l.id].short);
        el.title = net.lines[l.id].label;
        el.dataset.state = net.lines[l.id].state;
      }
      const t = SS.sim.totals();
      VC.set($(root, '[data-h="running"]'), `${t.active} running`);
      VC.set($(root, '[data-h="rate"]'), F.rate(t.down));
      const dis = VC.disruptions();
      const pill = $(root, '[data-h="pill"]');
      const worst = dis.some((x) => x.kind === 'failed') ? 'failed' : dis.length ? 'delays' : 'good';
      pill.dataset.tone = worst;
      const label = dis.length ? F.plural(dis.length, 'disruption') : 'Good service';
      if (pill.dataset.v !== label) { pill.dataset.v = label; pill.innerHTML = `<span class="c-pill__dot"></span>${esc(label)}`; }
      $(root, '[data-h="bell"]').hidden = !d.alerts.length;
    }

    function openPop(kind, anchor) {
      if (pop?.kind === kind) return closePop();
      closePop();
      const el = document.createElement('div');
      el.className = 'c-pop';
      el.dataset.kind = kind;
      el.setAttribute('role', 'dialog');
      root.appendChild(el);
      const r = anchor.getBoundingClientRect(), rr = root.getBoundingClientRect();
      el.style.top = `${r.bottom - rr.top + 8}px`;
      el.style.right = `${rr.right - r.right}px`;
      pop = { kind, el, anchor };
      anchor.setAttribute('aria-expanded', 'true');
      renderPop();
    }
    function renderPop() {
      if (!pop) return;
      const { kind, el } = pop;
      if (kind === 'running') {
        const run = sortJobs(d.jobs.filter((j) => j.state === 'running' || j.state === 'paused' || j.state === 'stalled'));
        el.innerHTML = `<header class="c-pop__h"><b>Running</b><a href="#" data-go="activity">Open Activity</a></header>
          <ul class="c-pop__list">${run.map((j) => `<li data-go="work/${j.work}" data-job="${j.id}">${VC.bullet(VC.jobLine(j))}<span class="c-pop__t"><b>${esc(VC.jobTitle(j))}</b><span>${esc(VC.jobStage(j))} · ${VC.jobPath(j).join(' to ')}</span></span><span class="c-pop__v c-num" data-f="pct">${F.pct(j.progress)}</span>${VC.bar(j.progress, VC.jobLine(j), 'c-bar--thin')}</li>`).join('')}</ul>`;
      } else if (kind === 'disruptions') {
        el.innerHTML = `<header class="c-pop__h"><b>Disruptions</b><a href="#" data-go="home">Open the map</a></header>${disruptionList()}`;
      } else if (kind === 'menu') {
        el.classList.add('c-pop--menu');
        el.innerHTML = `<div role="menu">${['Change Profile', 'Edit Editions', 'Open in Plex', 'Blocklist current release'].map((x) => `<button role="menuitem" class="c-menu__i" data-act="menu" data-v="${x}">${esc(x)}</button>`).join('')}<hr><button role="menuitem" class="c-menu__i c-menu__i--danger" data-act="menu" data-v="Delete files">Delete files</button></div>`;
      } else if (kind === 'bell') {
        const items = [...d.alerts.map((a) => ({ at: a.at, t: a.title, s: a.detail, tone: a.level === 'error' ? 'failed' : a.level === 'warning' ? 'delays' : '' })), ...d.history.slice(0, 5).map((h) => ({ at: h.at, t: `${d.W[h.work].title}${h.episode ? ' ' + F.ep(h.episode) : ''}`, s: h.text, tone: h.kind === 'failed' ? 'failed' : 'arrived' }))].sort((a, b) => b.at - a.at);
        el.innerHTML = `<header class="c-pop__h"><b>Notifications</b><span class="c-dim">${F.plural(items.length, 'item')}</span></header>
          <ul class="c-pop__list c-pop__list--notes">${items.map((x) => `<li><span class="c-note-mark c-tone-${x.tone}"></span><span class="c-pop__t"><b>${esc(x.t)}</b><span>${esc(x.s)}</span></span><time class="c-dim c-num">${F.ago(x.at)}</time></li>`).join('')}</ul>`;
      }
    }
    function closePop() {
      if (!pop) return;
      pop.anchor.removeAttribute('aria-expanded');
      const el = pop.el;
      el.classList.add('is-out');
      setTimeout(() => el.remove(), 140);
      pop = null;
    }

    function disruptionList() {
      const dis = VC.disruptions();
      if (!dis.length) return `<p class="c-empty">Every line has good service. New failures and delays land here first.</p>`;
      return `<ul class="c-disr">${dis.map((x) => {
        const act = x.kind === 'suspended' || (x.nodeRef && !x.job) ? `<button class="c-btn c-btn--s" data-go="nodes/${x.node}">${esc(x.action)}</button>`
          : x.kind === 'failed' ? `<button class="c-btn c-btn--s" data-act="blocklist" data-work="${x.work}">Blocklist and search</button>`
          : `<button class="c-btn c-btn--s" data-act="find-release" data-work="${x.work}">Find another release</button>`;
        return `<li class="c-disr__i" data-kind="${x.kind}" data-id="${x.id}">
          <span class="c-disr__mark"></span>${VC.bullet(x.line)}
          <span class="c-disr__b"><b>${esc(x.title)}</b><span><time class="c-num">${F.ago(x.at)}</time> · ${esc(x.detail)}</span></span>
          ${act}
        </li>`;
      }).join('')}</ul>`;
    }

    function openPalette() {
      if (palette) return closePalette();
      closePop();
      const el = document.createElement('div');
      el.className = 'c-palette';
      el.innerHTML = `<div class="c-palette__scrim" data-close></div><div class="c-palette__box" role="dialog" aria-label="Command palette"><div class="c-palette__in">${I('search', 17)}<input type="text" placeholder="Search Works, Nodes and commands" autocomplete="off" spellcheck="false" aria-label="Search"><kbd>esc</kbd></div><div class="c-palette__list" role="listbox"></div></div>`;
      root.appendChild(el);
      const input = $(el, 'input');
      const list = $(el, '.c-palette__list');
      let items = [], sel = 0;
      const commands = () => {
        const ams = d.N['ams-1'];
        const missingEp = VC.act.missingEpisodes().length;
        return [
          { g: 'Commands', label: `Search for missing episodes`, hint: F.plural(missingEp, 'episode'), icon: 'search', run: () => { VC.act.missingEpisodes().forEach((x) => VC.act.grab(x.w.id, [x.s, x.e.n])); toast(`Searching for ${F.plural(missingEp, 'missing episode')}`, 'Grabs ride the Acquire line from atlas'); } },
          ams.paused ? { g: 'Commands', label: 'Resume ams-1', icon: 'play', run: () => { VC.act.resumeNode('ams-1'); toast('ams-1 resumed'); } } : { g: 'Commands', label: 'Pause ams-1', hint: 'Holds every download', icon: 'pause', run: () => { VC.act.pauseNode('ams-1'); toast('ams-1 paused', 'Downloads hold their place'); } },
          d.jobs.some((j) => j.state === 'failed') && { g: 'Commands', label: 'Blocklist Coldwater Static and search again', icon: 'ban', run: () => act('blocklist', { work: 'coldwater-static' }) },
          d.jobs.some((j) => j.state === 'stalled') && { g: 'Commands', label: 'Find another release for The Velvet Hour', icon: 'refresh-cw', run: () => act('find-release', { work: 'velvet-hour' }) },
          { g: 'Commands', label: 'Approve every pending request', hint: F.plural(d.requests.filter((r) => r.state === 'pending').length, 'request'), icon: 'check', run: () => { d.requests.filter((r) => r.state === 'pending').forEach(VC.act.approve); toast('Approved every pending request'); } },
        ].filter(Boolean);
      };
      const views = [['home', 'Network'], ['library', 'Movies'], ['series', 'Series'], ['activity', 'Activity'], ['requests', 'Requests'], ['nodes', 'Nodes']];
      function build(q) {
        const ql = q.trim().toLowerCase();
        const match = (s) => !ql || s.toLowerCase().includes(ql);
        const works = ql ? VC.searchWorks(q).slice(0, 6) : d.works.filter((w) => VC.workJob(w)).slice(0, 4);
        items = [
          ...works.map((w) => ({ g: ql ? 'Works' : 'In flight', label: w.title, hint: `${w.type === 'movie' ? 'Movie' : 'Series'} · ${w.year}`, poster: w, run: () => go('work/' + w.id) })),
          ...views.filter(([, l]) => match(l)).map(([v, l]) => ({ g: 'Go to', label: l, icon: 'arrow-right', run: () => go(v) })),
          ...d.nodes.filter((n) => ql && match(n.name)).map((n) => ({ g: 'Nodes', label: n.name, hint: n.hardware, bullets: VC.nodeLines(n), run: () => go('nodes/' + n.id) })),
          ...commands().filter((c) => match(c.label)),
        ];
        sel = 0;
        let g = '';
        list.innerHTML = items.length ? items.map((it, i) => `${it.g !== g ? `<div class="c-palette__g">${esc((g = it.g))}</div>` : ''}<div class="c-palette__i" role="option" data-i="${i}"${i === sel ? ' aria-selected="true"' : ''}>${it.poster ? VC.poster(it.poster, 'c-palette__poster') : it.bullets ? VC.bullets(it.bullets) : `<span class="c-palette__ic">${SS.icon(it.icon || 'arrow-right', { size: 15, stroke: 1.75 })}</span>`}<span class="c-palette__l">${esc(it.label)}</span>${it.hint ? `<span class="c-palette__h">${esc(it.hint)}</span>` : ''}</div>`).join('')
          : `<p class="c-empty">Nothing matches “${esc(q)}”. Try a Work, a Node such as forge, or a command such as pause.</p>`;
      }
      function move(k) {
        if (!items.length) return;
        sel = (sel + k + items.length) % items.length;
        $$(list, '[data-i]').forEach((x) => x.toggleAttribute('aria-selected', +x.dataset.i === sel));
        $(list, `[data-i="${sel}"]`)?.scrollIntoView({ block: 'nearest' });
      }
      function run(i) { const it = items[i]; if (!it) return; closePalette(); it.run(); }
      input.addEventListener('input', () => build(input.value));
      input.addEventListener('keydown', (e) => {
        if (e.key === 'ArrowDown') { e.preventDefault(); move(1); }
        else if (e.key === 'ArrowUp') { e.preventDefault(); move(-1); }
        else if (e.key === 'Enter') { e.preventDefault(); run(sel); }
        else if (e.key === 'Escape') { e.preventDefault(); closePalette(); }
        else if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') e.stopPropagation();
      });
      list.addEventListener('pointermove', (e) => { const it = e.target.closest('[data-i]'); if (it && +it.dataset.i !== sel) { sel = +it.dataset.i; $$(list, '[data-i]').forEach((x) => x.toggleAttribute('aria-selected', +x.dataset.i === sel)); } });
      list.addEventListener('click', (e) => { const it = e.target.closest('[data-i]'); if (it) run(+it.dataset.i); });
      $(el, '[data-close]').addEventListener('click', closePalette);
      build('');
      palette = { el, prev: document.activeElement };
      input.focus();
    }
    function closePalette() {
      if (!palette) return;
      palette.el.remove();
      palette.prev?.focus?.({ preventScroll: true });
      palette = null;
    }

    const toast = (t, s, tone) => VC.toast(root, t, { sub: s, tone });

    function act(name, ds, btn) {
      if (name === 'palette') return openPalette();
      if (name === 'blocklist') {
        const w = d.W[ds.work];
        VC.act.blocklistAndSearch(w.id);
        toast(`${w.title} is on the Blocklist`, 'A new release is downloading on ams-1');
      } else if (name === 'find-release') {
        const w = d.W[ds.work];
        VC.act.findRelease(w.id);
        toast(`New release for ${w.title}`, 'Kestrel · 64 seeders');
      } else if (name === 'pause-node') { VC.act.pauseNode(ds.node); toast(`${ds.node} paused`, 'Downloads hold their place'); }
      else if (name === 'resume-node') { VC.act.resumeNode(ds.node); toast(`${ds.node} resumed`); }
      else return view?.act?.(name, ds, btn);
      refreshAll();
    }

    function refreshAll() {
      shellSecond();
      view?.refresh?.();
      renderPop();
    }

    ctx.listen(root, 'click', (e) => {
      const t = e.target;
      if (pop && !t.closest('.c-pop') && !t.closest('[data-pop]')) closePop();
      const popBtn = t.closest('[data-pop]');
      if (popBtn) { e.preventDefault(); return openPop(popBtn.dataset.pop, popBtn); }
      const a = t.closest('[data-act]');
      if (a && !a.disabled) { e.preventDefault(); e.stopPropagation(); return act(a.dataset.act, a.dataset, a); }
      const lb = t.closest('[data-line-btn]');
      if (lb) {
        if (route !== 'home') go('home');
        const on = view?.focusLine?.(lb.dataset.lineBtn);
        $$(root, '[data-line-btn]').forEach((b) => b.setAttribute('aria-pressed', b.dataset.lineBtn === on ? 'true' : 'false'));
        return;
      }
      const g = t.closest('[data-go]');
      if (g && !t.closest('button:not([data-go]), input, select, label')) {
        e.preventDefault();
        const to = g.dataset.go;
        const img = g.querySelector('.c-tile__art img');
        if (to.startsWith('work/') && img) {
          go(to, { transition: { before: () => (img.style.viewTransitionName = 'c-poster'), done: () => { const p = $(main, '.c-work__poster'); if (p) p.style.viewTransitionName = ''; } } });
        } else go(to);
      }
    });
    ctx.listen(root, 'keydown', (e) => {
      if (e.key !== 'Enter') return;
      if (e.target.matches('tr[data-act]')) act(e.target.dataset.act, e.target.dataset, e.target);
      else if (e.target.matches('tr[data-go]')) go(e.target.dataset.go);
    });
    ctx.listen(root, 'pointerover', (e) => {
      const lit = e.target.closest('[data-lit]');
      if (lit) VC.notice(lit);
      const ln = e.target.closest('[data-line], [data-line-btn]');
      if (ln && view?.hoverLine) view.hoverLine(ln.dataset.line || ln.dataset.lineBtn);
    });
    ctx.listen(root, 'pointerout', (e) => {
      const ln = e.target.closest('[data-line], [data-line-btn]');
      if (ln && view?.hoverLine && !e.relatedTarget?.closest?.('[data-line], [data-line-btn]')) view.hoverLine(null);
    });
    ctx.listen(window, 'keydown', (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); openPalette(); }
      else if (e.key === 'Escape') { if (palette) closePalette(); else closePop(); }
    });

    ctx.on('tick', () => view?.tick?.());
    ctx.on('second', () => { shellSecond(); view?.second?.(); if (pop?.kind === 'running') { const run = d.jobs; $$(pop.el, '[data-job]').forEach((li) => { const j = run.find((x) => x.id === li.dataset.job); if (j) { VC.set($(li, '[data-f="pct"]'), F.pct(j.progress)); $(li, '.c-bar__fill').style.transform = `scaleX(${j.progress})`; } }); } });
    ctx.on('job:new', (j) => { VC.light(j.id); view?.jobs?.(j); shellSecond(); });
    ctx.on('job:done', (j) => { VC.light(j.id); view?.jobs?.(j); shellSecond(); });
    ctx.on('job:update', (j) => { view?.jobs?.(j); shellSecond(); });
    ctx.on('work:update', (w) => view?.work?.(w));
    ctx.on('request:update', (r) => { VC.light(r.id); view?.requests?.(r); shellSecond(); });
    ctx.on('alerts:update', refreshAll);

    function homeView(el) {
      el.innerHTML = `<div class="c-home">
        <section class="c-status" aria-live="polite">
          <div class="c-status__lines">${VC.LINES.map((l) => `<span class="c-status__l" data-ls="${l.id}">${VC.bullet(l.id)}</span>`).join('')}</div>
          <p class="c-status__text" data-status></p>
          <span class="c-status__clock c-num" data-clock></span>
        </section>
        <div class="c-home__grid">
          <section class="c-panel c-panel--map" aria-label="Network map">
            <div class="c-mapwrap"></div>
            <footer class="c-legend">${VC.LINES.map((l) => `<button class="c-legend__i" data-line-btn="${l.id}">${VC.bullet(l.id)}<span>${l.name}</span><span class="c-num c-dim" data-trains="${l.id}"></span></button>`).join('')}<span class="c-legend__key"><span class="c-key c-key--train"></span>Train<span class="c-key c-key--delay"></span>Delayed<span class="c-key c-key--fail"></span>Failed<span class="c-key c-key--susp"></span>Suspended</span></footer>
          </section>
          <div class="c-home__side">
            <section class="c-panel c-panel--disr">
              <header class="c-panel__h"><h2>Disruptions</h2><span class="c-count c-num" data-dcount></span></header>
              <div data-disr></div>
            </section>
            <section class="c-panel c-panel--dep">
              <header class="c-panel__h"><h2>Departures</h2><span class="c-panel__meta">Next to leave</span></header>
              <table class="c-board"><tbody data-dep></tbody></table>
            </section>
          </div>
        </div>
        <section class="c-panel c-panel--acts">
          <header class="c-panel__h"><h2>Activity</h2><span class="c-count c-num" data-acount></span><a class="c-link" href="#" data-go="activity">All activity ${I('arrow-right', 13)}</a></header>
          <table class="c-table c-table--jobs">${JOB_HEAD(false)}<tbody data-jobs></tbody></table>
        </section>
      </div>`;
      const map = VC.netMap($(el, '.c-mapwrap'), ctx, { onStation: (id) => go('nodes/' + id) });
      let rows = null;
      const status = () => {
        const net = VC.net();
        const parts = net.sentence.map((p) => (typeof p === 'string' ? `<span>${esc(p)}</span>` : `<span class="c-tone-${p.tone}">${esc(p.text)}</span>`));
        const html = parts.join('<span class="c-status__sep" aria-hidden="true">·</span>');
        const t = $(el, '[data-status]');
        if (t.innerHTML !== html) t.innerHTML = html;
        for (const l of VC.LINES) $(el, `[data-ls="${l.id}"]`).dataset.state = net.lines[l.id].state;
        for (const l of VC.LINES) VC.set($(el, `[data-trains="${l.id}"]`), net.lines[l.id].running ? String(net.lines[l.id].running) : '');
      };
      const clock = () => VC.set($(el, '[data-clock]'), VC.clock().toLocaleTimeString('en-GB'));
      const disr = () => {
        const html = disruptionList();
        const host = $(el, '[data-disr]');
        if (host.dataset.v !== html) { host.dataset.v = html; host.innerHTML = html; }
        VC.set($(el, '[data-dcount]'), String(VC.disruptions().length));
      };
      const dep = () => {
        const body = $(el, '[data-dep]');
        const list = VC.departures(6);
        const ids = list.map((x) => x.id).join();
        if (body.dataset.ids !== ids) {
          body.dataset.ids = ids;
          body.innerHTML = list.map((x) => `<tr data-go="work/${x.work}" data-dep="${x.id}">
            <td class="c-board__t c-num" data-f="t"></td><td class="c-td-b">${VC.bullet(x.line)}</td>
            <td class="c-board__d"><span class="c-trunc"><b>${esc(x.title)}</b> <span class="c-dim">${esc(x.sub)}</span></span></td>
            <td class="c-board__p">${esc(x.platform)}</td><td class="c-board__s" data-f="s"></td></tr>`).join('');
        }
        for (const x of list) {
          const tr = $(body, `[data-dep="${x.id}"]`);
          VC.flap($(tr, '[data-f="t"]'), VC.boardTime(x.at));
          const s = $(tr, '[data-f="s"]');
          VC.flap(s, x.status);
          s.dataset.tone = x.tone;
        }
      };
      const jobs = () => {
        const list = sortJobs(d.jobs.filter((j) => j.state !== 'queued' && (j.state !== 'done' || VC.lit.has(j.id))));
        $(el, '[data-jobs]').innerHTML = list.length ? list.map((j) => jobRow(j)).join('') : `<tr><td colspan="8" class="c-empty">Nothing is moving. Approve a request or search for a missing Work and its train appears here.</td></tr>`;
        VC.set($(el, '[data-acount]'), String(d.jobs.filter((j) => j.state === 'running').length));
        rows = liveRows($(el, '[data-jobs]'));
        map.sync();
      };
      status(); clock(); disr(); dep(); jobs();
      return {
        tick() { map.tick(); rows.tick(); },
        second() { clock(); status(); dep(); rows.second(); map.second(); },
        jobs() { jobs(); dep(); status(); },
        refresh() { disr(); dep(); jobs(); status(); map.relayout(); },
        focusLine: (id) => map.focus(id, true),
        hoverLine: (id) => map.focus(id),
        destroy: () => map.destroy(),
      };
    }

    function libraryView(el, type) {
      const key = type === 'movie' ? 'lib' : 'ser';
      const prefs = store.get(key, { filter: 'all', sort: 'added', size: 'm', mode: 'grid' });
      const all = () => d.works.filter((w) => w.type === type);
      const GROUP = { moving: 'moving', stalled: 'moving', paused: 'moving', failed: 'missing', missing: 'missing', available: 'available', upgradable: 'upgradable', unreleased: 'unreleased', downloading: 'moving' };
      const FILTERS = [['all', 'All'], ['available', 'Available'], ['moving', 'In progress'], ['missing', 'Missing'], ['upgradable', 'Upgradable'], ['unreleased', 'Unreleased']];
      const SORTS = [['added', 'Date added'], ['title', 'Title'], ['year', 'Year'], ['rating', 'Rating']];
      const groupOf = (w) => GROUP[VC.statusOf(w).key] || 'available';
      const tileLabel = (w, s) => (s.key === 'available' ? (w.type === 'series' ? F.plural(w.seasons.length, 'season') : F.runtime(w.runtime)) : s.label);
      function list() {
        let ws = all();
        if (prefs.filter !== 'all') ws = ws.filter((w) => groupOf(w) === prefs.filter);
        const by = { added: (a, b) => b.added - a.added, title: (a, b) => a.title.replace(/^The /, '').localeCompare(b.title.replace(/^The /, '')), year: (a, b) => b.year - a.year, rating: (a, b) => (b.rating || 0) - (a.rating || 0) }[prefs.sort];
        return ws.sort(by);
      }
      function tile(w) {
        const s = VC.statusOf(w);
        return `<a class="c-tile" href="#" data-go="work/${w.id}" data-work="${w.id}">
          <span class="c-tile__art">${VC.poster(w)}</span>
          <span class="c-tile__t">${esc(w.title)}</span>
          <span class="c-tile__m"><span class="c-num">${w.year}</span><span class="c-tile__s c-st-${s.key}" data-f="s">${esc(tileLabel(w, s))}</span></span>
          ${s.key === 'moving' ? `<span class="c-tile__track" style="--ink:${VC.L[s.line].ink}"><span class="c-tile__fill" style="transform:scaleX(${s.p.toFixed(4)})"></span><span class="c-tile__train" style="left:${(s.p * 100).toFixed(2)}%"></span></span>` : ''}
        </a>`;
      }
      function row(w) {
        const s = VC.statusOf(w);
        const cov = VC.coverage(w).filter((c) => c.inProfile);
        return `<tr class="c-row" data-go="work/${w.id}" data-work="${w.id}" tabindex="0">
          <td class="c-td-thumb">${VC.poster(w, 'c-thumb')}</td>
          <td class="c-td-w"><span class="c-trunc"><b>${esc(w.title)}</b></span></td>
          <td class="c-num">${w.year}</td>
          <td><span class="c-tile__s c-st-${s.key}" data-f="s">${esc(s.label)}</span></td>
          <td><span class="c-covmini">${cov.map((c) => `<span class="c-covmini__c" data-s="${c.state}" title="${esc(c.target.name)} · ${esc(F.coverageLabel[c.raw] || c.raw)}">${esc(c.target.short)}</span>`).join('')}</span></td>
          <td class="c-num c-td-r">${w.type === 'movie' ? (w.files.length ? F.bytes(w.files.reduce((a, f) => a + f.size, 0)) : '') : F.bytes(w.size)}</td>
          <td>${esc(VC.profile(w).name)}</td>
          <td class="c-num c-td-r">${esc(F.ago(w.added))}</td>
        </tr>`;
      }
      function render() {
        const ws = list();
        const counts = Object.fromEntries(FILTERS.map(([k]) => [k, k === 'all' ? all().length : all().filter((w) => groupOf(w) === k).length]));
        el.innerHTML = `<div class="c-page c-lib">
          <header class="c-pagehead">
            <h1>${type === 'movie' ? 'Movies' : 'Series'}<span class="c-count c-num">${all().length}</span></h1>
            <div class="c-toolbar">
              <div class="c-seg" role="tablist" aria-label="Status">${FILTERS.filter(([k]) => counts[k] || k === 'all').map(([k, l]) => `<button role="tab" aria-selected="${prefs.filter === k}" data-act="filter" data-v="${k}">${l}<span class="c-num">${counts[k]}</span></button>`).join('')}</div>
              <label class="c-select"><span>Sort</span><select data-sort>${SORTS.map(([k, l]) => `<option value="${k}"${prefs.sort === k ? ' selected' : ''}>${l}</option>`).join('')}</select>${I('chevron-down', 14)}</label>
              ${prefs.mode === 'grid' ? `<div class="c-seg c-seg--compact" aria-label="Poster size">${[['s', 'Small'], ['m', 'Medium'], ['l', 'Large']].map(([k, l]) => `<button aria-pressed="${prefs.size === k}" data-act="size" data-v="${k}" aria-label="${l} posters">${k.toUpperCase()}</button>`).join('')}</div>` : ''}
              <div class="c-seg c-seg--compact" aria-label="View">${[['grid', 'layout-grid', 'Grid'], ['table', 'list', 'Table']].map(([k, ic, l]) => `<button aria-pressed="${prefs.mode === k}" data-act="mode" data-v="${k}" aria-label="${l}">${k === 'grid' ? I('layout-grid', 15) : I('list', 15)}</button>`).join('')}</div>
            </div>
          </header>
          ${!ws.length ? `<p class="c-empty c-empty--page">No ${type === 'movie' ? 'movies' : 'series'} are ${esc(FILTERS.find(([k]) => k === prefs.filter)[1].toLowerCase())} right now. <button class="c-link" data-act="filter" data-v="all">Show all ${all().length}</button></p>`
            : prefs.mode === 'grid' ? `<div class="c-grid" data-size="${prefs.size}">${ws.map(tile).join('')}</div>`
            : `<table class="c-table c-table--lib"><thead><tr><th class="c-td-thumb"><span class="c-vh">Poster</span></th><th>Title</th><th>Year</th><th>Status</th><th>Targets</th><th class="c-td-r">Size</th><th>Profile</th><th class="c-td-r">Added</th></tr></thead><tbody>${ws.map(row).join('')}</tbody></table>`}
        </div>`;
        $(el, '[data-sort]').addEventListener('change', (e) => { prefs.sort = e.target.value; store.set(key, prefs); render(); });
      }
      render();
      const live = () => {
        for (const t of $$(el, '[data-work]')) {
          const w = d.W[t.dataset.work];
          const s = VC.statusOf(w);
          VC.set($(t, '[data-f="s"]'), t.classList.contains('c-tile') ? tileLabel(w, s) : s.label);
          const fill = $(t, '.c-tile__fill');
          if (fill && s.p != null) { fill.style.transform = `scaleX(${s.p.toFixed(4)})`; $(t, '.c-tile__train').style.left = (s.p * 100).toFixed(2) + '%'; }
          if (!fill && s.key === 'moving' && t.classList.contains('c-tile')) t.outerHTML = tile(w);
        }
      };
      return {
        act(name, ds) {
          if (name === 'filter') prefs.filter = ds.v;
          if (name === 'size') prefs.size = ds.v;
          if (name === 'mode') prefs.mode = ds.v;
          store.set(key, prefs);
          render();
        },
        second: live,
        work: live,
        jobs: live,
      };
    }

    function workView(el, id) {
      const w = d.W[id];
      let season = w.type === 'series' ? (VC.workJob(w)?.episode?.[0] || w.seasons.length) : 0;
      let manual = false;
      let monitored = true;
      function render() {
        const s = VC.statusOf(w);
        const cov = VC.coverage(w);
        const prof = VC.profile(w);
        const by = w.addedBy ? d.A[w.addedBy] : null;
        el.innerHTML = `<article class="c-work">
          <div class="c-work__hero"><img class="c-work__backdrop" src="${SS.art.backdrop(w, { w: 1600, h: 640 })}" alt=""><div class="c-work__scrim"></div></div>
          <div class="c-work__head">
            <img class="c-work__poster" src="${SS.art.poster(w)}" alt="${esc(w.title)} poster" style="view-transition-name:c-poster">
            <div class="c-work__id">
              <nav class="c-crumbs" aria-label="Breadcrumb"><a href="#" data-go="${w.type === 'movie' ? 'library' : 'series'}">${w.type === 'movie' ? 'Movies' : 'Series'}</a>${I('chevron-right', 13)}<span>${esc(w.title)}</span></nav>
              <h1 class="c-work__title">${esc(w.title)}</h1>
              <p class="c-work__meta">${esc(VC.workMeta(w))}<span>${esc(w.genres.join(', '))}</span>${w.rating ? `<span>Rated <b class="c-num">${w.rating.toFixed(1)}</b></span>` : ''}</p>
              <p class="c-work__now" data-now>${nowLine(s)}</p>
            </div>
            <div class="c-work__actions">
              <button class="c-switch" role="switch" aria-checked="${monitored}" data-act="monitor"><span class="c-switch__k"></span>Monitored</button>
              <button class="c-btn" data-act="manual" aria-expanded="${manual}">${I('list-filter', 15)}Manual search</button>
              <button class="c-btn c-btn--primary" data-act="search">${I('search', 15)}Search</button>
              <button class="c-icon-btn c-icon-btn--b" data-pop="menu" aria-label="More actions">${I('ellipsis', 17)}</button>
            </div>
          </div>
          <div class="c-work__body">
            <div class="c-work__main">
              <p class="c-work__overview">${esc(w.overview)}</p>
              ${manual ? manualHTML() : ''}
              ${w.type === 'series' ? seasonsHTML() : ''}
              <section class="c-sec">
                <header class="c-sec__h"><h2>Targets and Editions</h2><span class="c-dim">Profile ${esc(prof.name)} · ${F.plural(w.editions.length, 'Edition')}</span></header>
                <table class="c-cov"><thead><tr><th>Edition</th>${cov.map((c) => `<th><b>${esc(c.target.name)}</b><span>${c.target.kind === 'derived' ? `Derived from ${esc(VC.targetName(c.target.from))}` : 'Downloaded'}</span></th>`).join('')}</tr></thead>
                <tbody>${w.editions.map((ed) => `<tr><th>${esc(ed)}</th>${cov.map(coverageCell).join('')}</tr>`).join('')}</tbody></table>
              </section>
              ${w.type === 'movie' ? `<section class="c-sec">
                <header class="c-sec__h"><h2>Files</h2><span class="c-dim">${w.files.length ? `${F.plural(w.files.length, 'file')} on vault · ${F.bytes(w.files.reduce((a, f) => a + f.size, 0))}` : 'None yet'}</span></header>
                ${w.files.length ? `<table class="c-table c-table--files"><thead><tr><th>Target</th><th>File</th><th class="c-td-r">Size</th><th>Video</th><th>Audio</th><th>Resolution</th></tr></thead><tbody>${w.files.map((f) => `<tr><td>${esc(VC.targetName(f.target))}${f.derived ? ' <span class="c-dim">derived</span>' : ''}</td><td class="c-td-file"><span class="c-trunc" title="${esc(f.name)}">${esc(f.name)}</span></td><td class="c-num c-td-r">${F.bytes(f.size)}</td><td>${esc(f.video)}</td><td>${esc(f.audio)}</td><td class="c-num">${esc(f.resolution)}</td></tr>`).join('')}</tbody></table>`
                  : `<p class="c-empty">Files appear here once a release is imported. The Route shows where this Work is on its way.</p>`}
              </section>` : ''}
            </div>
            <aside class="c-work__side">
              <section class="c-panel c-panel--route" data-route>${routeHTML(w)}</section>
              <dl class="c-facts">
                <dt>Profile</dt><dd>${esc(prof.name)} · ${prof.targets.map((t) => esc(VC.target(t).short)).join(', ')}</dd>
                ${w.type === 'movie' ? `<dt>Director</dt><dd>${esc(w.director)}</dd><dt>Studio</dt><dd>${esc(w.studio)}</dd>` : `<dt>Creator</dt><dd>${esc(w.creator)}</dd><dt>Network</dt><dd>${esc(w.network)}</dd>`}
                <dt>Cast</dt><dd>${esc(w.cast.slice(0, 3).join(', '))}</dd>
                <dt>Added</dt><dd>${esc(F.ago(w.added))}${by ? ` · requested by ${esc(by.name)}` : ''}</dd>
              </dl>
            </aside>
          </div>
        </article>`;
      }
      function nowLine(s) {
        const j = VC.workJob(w);
        if (j) return `${VC.bullet(VC.jobLine(j))}<span class="c-tone-${toneOf(j)}">${esc(VC.jobStage(j))}${j.episode ? ' ' + F.ep(j.episode) : ''}</span><span class="c-dim">${j.state === 'failed' ? esc(j.message) : `${F.pct(j.progress)} at ${esc(VC.jobPath(j).join(' to '))}${VC.jobEta(j) ? ' · ' + VC.jobEta(j) + ' left' : ''}`}</span>`;
        const cls = { available: 'arrived', upgradable: 'delays', missing: 'muted', unreleased: 'muted' }[s.key] || '';
        return `<span class="c-tone-${cls}">${esc(s.label)}</span>${w.type === 'series' && w.nextAir ? `<span class="c-dim">Next episode ${esc(F.until(w.nextAir))}</span>` : ''}`;
      }
      function seasonsHTML() {
        const S = w.seasons[season - 1];
        return `<section class="c-sec">
          <header class="c-sec__h"><h2>Seasons</h2>
            <div class="c-seg" role="tablist">${w.seasons.map((x) => `<button role="tab" aria-selected="${x.n === season}" data-act="season" data-v="${x.n}">Season ${x.n}<span class="c-num">${x.episodes.filter((e) => e.status === 'available').length}/${x.episodes.length}</span></button>`).join('')}</div>
          </header>
          <table class="c-table c-table--eps"><thead><tr><th class="c-td-n">#</th><th>Episode</th><th>Aired</th><th>Status</th><th class="c-td-a"><span class="c-vh">Actions</span></th></tr></thead><tbody>
          ${S.episodes.map((e) => {
            const j = d.jobs.find((x) => x.work === w.id && x.episode && x.episode[0] === S.n && x.episode[1] === e.n && x.state !== 'done');
            const st = j ? `${VC.bar(j.progress, VC.jobLine(j))}<span class="c-num" data-ep-p="${j.id}">${esc(VC.jobVerb(j))} ${F.pct(j.progress)}</span>`
              : { available: '<span class="c-tone-arrived">Available</span>', missing: '<span class="c-cov__mark c-cov__mark--miss"></span>Missing', failed: '<span class="c-tone-failed">Failed</span>', unreleased: `<span class="c-dim">Airs ${esc(F.until(e.air))}</span>`, downloading: 'Waiting' }[e.status];
            return `<tr class="c-row"><td class="c-num c-td-n">${e.n}</td><td class="c-td-w"><span class="c-trunc">${esc(e.title)}</span></td><td class="c-num c-dim">${e.air.toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })}</td><td class="c-td-st">${st}</td>
              <td class="c-td-a">${e.status === 'missing' || e.status === 'failed' ? `<button class="c-btn c-btn--s" data-act="grab-ep" data-s="${S.n}" data-e="${e.n}">${I('search', 13)}Search</button>` : ''}</td></tr>`;
          }).join('')}</tbody></table></section>`;
      }
      function manualHTML() {
        return `<section class="c-sec c-manual">
          <header class="c-sec__h"><h2>Manual search</h2><span class="c-dim">4 indexers · ${releasesFor(w).length} releases</span><button class="c-icon-btn" data-act="manual" aria-label="Close manual search">${I('x', 15)}</button></header>
          <table class="c-table c-table--rel"><thead><tr><th>Release</th><th>Target</th><th class="c-td-r">Size</th><th class="c-td-r">Seeders</th><th>Indexer</th><th>Verdict</th><th class="c-td-a"><span class="c-vh">Grab</span></th></tr></thead><tbody>
          ${releasesFor(w).map((r, i) => `<tr class="c-row"><td class="c-td-file"><span class="c-trunc" title="${esc(r.name)}">${esc(r.name)}</span></td><td>${esc(VC.target(r.target).short)}</td><td class="c-num c-td-r">${F.bytes(r.size)}</td><td class="c-num c-td-r">${r.seeds}</td><td>${esc(r.indexer)}</td><td>${r.ok ? '<span class="c-tone-arrived">Passes Profile</span>' : `<span class="c-tone-muted">${esc(r.why)}</span>`}</td><td class="c-td-a"><button class="c-btn c-btn--s" data-act="grab" data-i="${i}"${r.ok ? '' : ' disabled'}>${I('download', 13)}Grab</button></td></tr>`).join('')}
          </tbody></table></section>`;
      }
      render();
      const live = () => {
        const now = $(el, '[data-now]');
        const html = nowLine(VC.statusOf(w));
        if (now.innerHTML !== html) now.innerHTML = html;
        const r = VC.route(w);
        const routeEl = $(el, '[data-route]');
        const stops = $$(routeEl, '.c-stop');
        if (stops.map((s) => s.dataset.state).join() !== r.stops.map((s) => s.state).join() || stops.length !== r.stops.length) routeEl.innerHTML = routeHTML(w);
        else r.stops.forEach((s, i) => { VC.set($(stops[i], '[data-f="detail"]'), s.detail || ''); const t = $(stops[i], '[data-f="time"]'); const html2 = s.at ? `${s.est ? '<span class="c-approx">about</span> ' : ''}${VC.timeOrDay(s.at)}` : ''; if (t.innerHTML !== html2) t.innerHTML = html2; });
        const cov = VC.coverage(w);
        $$(el, '.c-cov tbody tr').forEach((tr) => $$(tr, 'td').forEach((td, i) => {
          if (td.dataset.s !== cov[i].state) { td.outerHTML = coverageCell(cov[i]); return; }
          if (cov[i].state === 'moving') { $(td, '.c-bar__fill').style.transform = `scaleX(${cov[i].p.toFixed(4)})`; VC.set($(td, '[data-f="cov-p"]'), `${VC.jobVerb(cov[i].job)} ${F.pct(cov[i].job.progress)}`); }
        }));
        $$(el, '[data-ep-p]').forEach((n) => { const j = d.jobs.find((x) => x.id === n.dataset.epP); if (j) { VC.set(n, `${VC.jobVerb(j)} ${F.pct(j.progress)}`); n.previousElementSibling.firstElementChild.style.transform = `scaleX(${j.progress.toFixed(4)})`; } });
      };
      return {
        second: live,
        jobs(j) { if (j && j.work !== w.id) return; const y = el.scrollTop; render(); el.scrollTop = y; },
        work: (x) => { if (x === w) { render(); } },
        act(name, ds, btn) {
          if (name === 'monitor') { monitored = !monitored; btn.setAttribute('aria-checked', monitored); toast(monitored ? `Monitoring ${w.title}` : `${w.title} is no longer monitored`, monitored ? 'New releases and upgrades are grabbed' : 'Files stay; nothing new is grabbed'); return; }
          if (name === 'search') {
            btn.disabled = true;
            btn.classList.add('is-loading');
            setTimeout(() => {
              btn.disabled = false;
              btn.classList.remove('is-loading');
              const job = VC.workJob(w);
              if (job) toast(`${w.title} is already on its way`, `${VC.jobStage(job)} at ${VC.jobPath(job).join(' to ')}`);
              else if (w.status === 'unreleased') toast('Nothing to grab yet', `${w.title} is not released`);
              else if (w.type === 'series' && w.counts.missing) { const m = VC.act.missingEpisodes().filter((x) => x.w === w); m.forEach((x) => VC.act.grab(w.id, [x.s, x.e.n])); toast(`Grabbed ${F.plural(m.length, 'episode')}`, 'Downloading on ams-1'); }
              else if (w.status === 'missing') { VC.act.grab(w.id); toast(`Grabbed ${w.title}`, 'Downloading on ams-1'); }
              else toast(`${w.title} meets its Profile`, 'Searched 4 indexers; no upgrade passes the cutoff');
            }, 900);
            return;
          }
          if (name === 'manual') { manual = !manual; render(); if (manual) $(el, '.c-manual')?.scrollIntoView({ block: 'nearest', behavior: VC.reduced() ? 'auto' : 'smooth' }); return; }
          if (name === 'grab') { const r = releasesFor(w)[+ds.i]; VC.act.grab(w.id); manual = false; render(); toast(`Grabbed ${r.name.split('.').slice(-1)[0].split('-').pop()}`, `${F.bytes(r.size)} from ${r.indexer}`); return; }
          if (name === 'grab-ep') { VC.act.grab(w.id, [+ds.s, +ds.e]); render(); toast(`Grabbed ${w.title} ${F.ep([+ds.s, +ds.e])}`, 'Downloading on ams-1'); return; }
          if (name === 'season') { season = +ds.v; render(); return; }
          if (name === 'menu') { toast(ds.v, 'Prototype: this action is not wired'); closePop(); }
        },
        destroy() {},
      };
    }

    function activityView(el) {
      let tab = 'all';
      let only = null;
      let rows = null;
      const TABS = [['all', 'All'], ['running', 'Running'], ['queued', 'Queued'], ['stalled', 'Stalled'], ['failed', 'Failed']];
      const match = (j) => (tab === 'all' ? j.state !== 'done' || VC.lit.has(j.id) : tab === 'stalled' ? j.state === 'stalled' || j.state === 'paused' : j.state === tab) && (!only || VC.jobLine(j) === only);
      function render() {
        const list = sortJobs(d.jobs.filter(match));
        const count = (k) => d.jobs.filter((j) => (k === 'all' ? j.state !== 'done' : k === 'stalled' ? j.state === 'stalled' || j.state === 'paused' : j.state === k)).length;
        let lastLine = '';
        el.innerHTML = `<div class="c-page">
          <header class="c-pagehead"><h1>Activity<span class="c-count c-num">${count('all')}</span></h1>
            <div class="c-toolbar">
              <div class="c-seg" role="tablist">${TABS.map(([k, l]) => `<button role="tab" aria-selected="${tab === k}" data-act="tab" data-v="${k}">${l}<span class="c-num">${count(k)}</span></button>`).join('')}</div>
              <div class="c-linefilter" aria-label="Filter by line">${VC.LINES.filter((l) => l.id !== 'notify' && l.id !== 'deliver').map((l) => `<button aria-pressed="${only === l.id}" data-act="only" data-v="${l.id}">${VC.bullet(l.id)}<span>${l.name}</span></button>`).join('')}</div>
            </div>
          </header>
          <table class="c-table c-table--jobs c-table--grouped">${JOB_HEAD(true)}<tbody data-jobs>${list.map((j) => {
            const l = VC.jobLine(j);
            const head = l !== lastLine ? `<tr class="c-grouprow"><th colspan="9"><span class="c-grouprow__in">${VC.bullet(l)}<b>${VC.L[l].name}</b><span class="c-dim">${{ acquire: 'atlas to ams-1 to vault', library: 'vault to atlas', process: 'vault to forge, den suspended' }[l] || ''}</span></span></th></tr>` : '';
            lastLine = l;
            return head + jobRow(j, { actions: true });
          }).join('') || `<tr><td colspan="9" class="c-empty">No ${tab === 'all' ? '' : esc(TABS.find(([k]) => k === tab)[1].toLowerCase()) + ' '}work${only ? ' on the ' + VC.L[only].name + ' line' : ''}. Trains appear here the moment a job starts.</td></tr>`}</tbody></table>
        </div>`;
        rows = liveRows($(el, '[data-jobs]'));
      }
      render();
      return {
        tick: () => rows.tick(),
        second: () => rows.second(),
        jobs: render,
        refresh: render,
        act(name, ds) {
          if (name === 'tab') tab = ds.v;
          if (name === 'only') only = only === ds.v ? null : ds.v;
          render();
        },
      };
    }

    function nodesView(el, id) {
      let sel = d.N[id] ? id : 'atlas';
      const stateLabel = { good: 'Online', delays: 'Delays', suspended: 'Suspended', paused: 'Paused' };
      function render() {
        el.innerHTML = `<div class="c-page c-nodes">
          <header class="c-pagehead"><h1>Nodes<span class="c-count c-num">${d.nodes.length}</span></h1><p class="c-pagehead__sub">${esc(VC.net().sentence.map((p) => (typeof p === 'string' ? p : p.text)).join(' · '))}</p></header>
          <div class="c-nodes__grid">
            <table class="c-table c-table--nodes"><thead><tr><th>Node</th><th>Lines</th><th>State</th><th class="c-td-r">CPU</th><th class="c-td-r">GPU</th><th class="c-td-r">Memory</th><th class="c-td-r">In</th><th class="c-td-r">Out</th><th>Storage</th></tr></thead>
              <tbody>${d.nodes.map((n) => {
                const s = VC.nodeState(n);
                const disk = n.disks[0];
                return `<tr class="c-row" data-node="${n.id}" data-act="select" data-v="${n.id}" tabindex="0" aria-selected="${n.id === sel}">
                  <td class="c-td-w"><span class="c-trunc"><b>${esc(n.name)}</b> <span class="c-dim">${esc(n.place)}</span></span></td>
                  <td>${VC.bullets(VC.nodeLines(n))}</td>
                  <td><span class="c-nstate" data-s="${s}">${stateLabel[s]}</span></td>
                  <td class="c-num c-td-r" data-f="cpu">${s === 'suspended' ? '' : F.pct(n.cpu)}</td>
                  <td class="c-num c-td-r" data-f="gpu">${n.gpu != null ? F.pct(n.gpu) : ''}</td>
                  <td class="c-num c-td-r">${s === 'suspended' ? '' : F.pct(n.mem)}</td>
                  <td class="c-num c-td-r" data-f="down">${n.down ? F.rate(n.down) : ''}</td>
                  <td class="c-num c-td-r" data-f="up">${n.up && s !== 'suspended' ? F.rate(n.up) : ''}</td>
                  <td class="c-td-disk">${VC.bar(disk.used / disk.total, 'library', 'c-bar--thin')}<span class="c-num c-dim">${F.pct(disk.used / disk.total)}</span></td>
                </tr>`;
              }).join('')}</tbody></table>
            <aside class="c-panel c-nodecard" data-detail>${detail()}</aside>
          </div>
        </div>`;
      }
      function detail() {
        const n = d.N[sel];
        const s = VC.nodeState(n);
        const groups = VC.nodeLines(n);
        return `<header class="c-nodecard__h"><h2>${esc(n.name)}</h2><span class="c-nstate" data-s="${s}">${n.controller ? 'Controller · ' : ''}${stateLabel[s]}</span></header>
          <p class="c-nodecard__hw">${esc(n.hardware)}<br>${esc(n.os)} · ${esc(n.address)} · SlipStream ${esc(n.version)}</p>
          ${n.stateNote ? `<p class="c-nodecard__note c-tone-${s}">${esc(n.stateNote)}</p>` : ''}
          <h3>Roles</h3>
          <ul class="c-roles">${groups.map((g) => `<li>${VC.bullet(g)}<span><b>${VC.L[g].name}</b><span class="c-dim">${n.roles.filter((r) => VC.roleLine[r] === g).map(esc).join(', ')}</span></span></li>`).join('')}</ul>
          ${s === 'suspended' ? '' : `<h3>Load</h3><dl class="c-load">
            <dt>CPU</dt><dd>${VC.bar(n.cpu, 'library', 'c-bar--thin')}<span class="c-num" data-d="cpu">${F.pct(n.cpu)}</span></dd>
            ${n.gpu != null ? `<dt>GPU</dt><dd>${VC.bar(n.gpu, 'process', 'c-bar--thin')}<span class="c-num" data-d="gpu">${F.pct(n.gpu)}</span></dd>` : ''}
            <dt>Memory</dt><dd>${VC.bar(n.mem, 'library', 'c-bar--thin')}<span class="c-num">${F.pct(n.mem)}</span></dd>
            ${n.temp ? `<dt>Temperature</dt><dd><span class="c-num${n.temp > 80 ? ' c-tone-delays' : ''}">${n.temp} °C</span></dd>` : ''}
            <dt>Uptime</dt><dd><span class="c-num">${F.dur(n.uptime / 1000)}</span></dd>
          </dl>`}
          <h3>Storage</h3>
          <ul class="c-disks">${n.disks.map((k) => `<li><span>${esc(k.path)}</span><span class="c-num c-dim">${F.bytes(k.used)} of ${F.bytes(k.total)}</span>${VC.bar(k.used / k.total, 'library')}</li>`).join('')}</ul>
          <div class="c-nodecard__actions">
            ${s === 'suspended' ? `<button class="c-btn c-btn--primary" data-act="wake" data-v="${n.id}">Wake ${esc(n.name)}</button>` : n.paused ? `<button class="c-btn c-btn--primary" data-act="resume-node" data-node="${n.id}">${I('play', 14)}Resume</button>` : `<button class="c-btn" data-act="pause-node" data-node="${n.id}">${I('pause', 14)}Pause</button>`}
            <button class="c-btn" data-act="logs" data-v="${n.id}">Open logs</button>
          </div>`;
      }
      render();
      return {
        second() {
          for (const tr of $$(el, 'tr[data-node]')) {
            const n = d.N[tr.dataset.node];
            if (VC.nodeState(n) === 'suspended') continue;
            VC.set($(tr, '[data-f="cpu"]'), F.pct(n.cpu));
            if (n.gpu != null) VC.set($(tr, '[data-f="gpu"]'), F.pct(n.gpu));
            VC.set($(tr, '[data-f="down"]'), n.down ? F.rate(n.down) : '');
            VC.set($(tr, '[data-f="up"]'), n.up ? F.rate(n.up) : '');
          }
          const n = d.N[sel];
          VC.set($(el, '[data-d="cpu"]'), F.pct(n.cpu));
          if (n.gpu != null) VC.set($(el, '[data-d="gpu"]'), F.pct(n.gpu));
        },
        refresh: render,
        act(name, ds) {
          if (name === 'select') { sel = ds.v; ctx.setView('nodes/' + sel); $$(el, 'tr[data-node]').forEach((tr) => tr.setAttribute('aria-selected', tr.dataset.node === sel)); $(el, '[data-detail]').innerHTML = detail(); }
          if (name === 'wake') toast('den did not answer', `Last seen at ${F.time(d.N.den.lastSeen)}. forge covers its Transcoding Role.`, 'muted');
          if (name === 'logs') toast(`Logs for ${ds.v}`, 'Prototype: logs are not part of this direction');
        },
      };
    }

    function requestsView(el) {
      let tab = 'pending';
      let denying = null;
      const TABS = [['pending', 'Pending'], ['approved', 'Approved'], ['downloading', 'On its way'], ['available', 'Available'], ['denied', 'Denied']];
      const statusText = { pending: 'Pending', approved: 'Approved · searching', downloading: 'On its way', available: 'Available', denied: 'Denied' };
      function render() {
        const list = d.requests.filter((r) => r.state === tab || (VC.lit.has(r.id) && tab === 'pending' && r.decidedAt));
        el.innerHTML = `<div class="c-page">
          <header class="c-pagehead"><h1>Requests<span class="c-count c-num">${d.requests.length}</span></h1>
            <div class="c-toolbar"><div class="c-seg" role="tablist">${TABS.map(([k, l]) => `<button role="tab" aria-selected="${tab === k}" data-act="tab" data-v="${k}">${l}<span class="c-num">${d.requests.filter((r) => r.state === k).length}</span></button>`).join('')}</div></div>
          </header>
          <table class="c-table c-table--req"><thead><tr><th class="c-td-thumb"><span class="c-vh">Poster</span></th><th>Work</th><th>Requested by</th><th>When</th><th>Note</th><th>Status</th><th class="c-td-a"><span class="c-vh">Actions</span></th></tr></thead><tbody>
          ${list.map((r) => {
            const w = d.W[r.work];
            const a = d.A[r.by];
            const jr = VC.journey(w);
            const lit = VC.lit.has(r.id);
            return `<tr class="c-row" data-req="${r.id}"${lit ? ` data-lit="${r.id}"` : ''}>
              <td class="c-td-thumb">${VC.poster(w, 'c-thumb')}</td>
              <td class="c-td-w"><a href="#" class="c-trunc" data-go="work/${w.id}"><b>${esc(w.title)}</b> <span class="c-dim">${w.type === 'series' ? (r.seasons ? `Season ${r.seasons.join(', ')}` : 'Every season') : w.year}</span></a></td>
              <td><span class="c-who">${VC.avatar(a.id)}${esc(a.name)}</span></td>
              <td class="c-num c-dim">${esc(F.ago(r.at))}</td>
              <td class="c-td-note"><span class="c-trunc">${esc(r.note || r.reason || '')}</span></td>
              <td><span class="c-rstate" data-s="${r.state}">${esc(statusText[r.state])}${jr && r.state === 'downloading' ? ` <span class="c-num" data-rp="${r.id}">${F.pct(jr.p)}</span>` : ''}</span></td>
              <td class="c-td-a">${r.state === 'pending' ? `<button class="c-btn c-btn--s" data-act="deny" data-v="${r.id}">Deny</button><button class="c-btn c-btn--s c-btn--primary" data-act="approve" data-v="${r.id}">${I('check', 13)}Approve</button>` : ''}</td>
            </tr>${denying === r.id ? `<tr class="c-denyrow"><td></td><td colspan="6"><form class="c-deny" data-deny="${r.id}"><label>Reason for ${esc(a.name.split(' ')[0])}<input name="reason" type="text" placeholder="Already in the library, try Movies" autocomplete="off"></label><button type="button" class="c-btn c-btn--s" data-act="cancel-deny">Cancel</button><button class="c-btn c-btn--s c-btn--danger" type="submit">Deny request</button></form></td></tr>` : ''}`;
          }).join('') || `<tr><td colspan="7" class="c-empty">No ${esc(TABS.find(([k]) => k === tab)[1].toLowerCase())} requests. Members request from the portal; new ones wait here for you.</td></tr>`}
          </tbody></table></div>`;
        const form = $(el, '[data-deny]');
        if (form) {
          $(form, 'input').focus();
          form.addEventListener('submit', (e) => {
            e.preventDefault();
            const r = d.requests.find((x) => x.id === form.dataset.deny);
            VC.act.deny(r, form.reason.value.trim());
            denying = null;
            toast(`Denied ${d.W[r.work].title}`, `${d.A[r.by].name} sees your reason in the portal`);
            render();
          });
        }
      }
      render();
      return {
        second() { $$(el, '[data-rp]').forEach((n) => { const r = d.requests.find((x) => x.id === n.dataset.rp); const jr = VC.journey(d.W[r.work]); if (jr) VC.set(n, F.pct(jr.p)); }); },
        requests: render,
        act(name, ds) {
          if (name === 'tab') { tab = ds.v; denying = null; }
          if (name === 'approve') { const r = d.requests.find((x) => x.id === ds.v); VC.act.approve(r); toast(`Approved ${d.W[r.work].title}`, `${d.A[r.by].name} will be told when it arrives`); }
          if (name === 'deny') denying = ds.v;
          if (name === 'cancel-deny') denying = null;
          render();
        },
      };
    }

    go(ctx.view || 'home');
    shellSecond();
    return { destroy() { view?.destroy?.(); closePalette(); } };
  };
})();
