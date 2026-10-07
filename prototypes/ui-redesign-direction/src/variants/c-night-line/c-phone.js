(function () {
  'use strict';
  const SS = window.SS;
  const d = SS.data;
  const F = SS.fmt;
  const esc = F.esc;
  const VC = window.VC;
  const I = (name, size = 20) => SS.icon(name, { size, stroke: 1.75 });
  const $ = (el, s) => el.querySelector(s);
  const $$ = (el, s) => [...el.querySelectorAll(s)];

  // Apple's momentum projection: where a flick would come to rest.
  const project = (v, rate = 0.998) => ((v / 1000) * rate) / (1 - rate);

  function tracker() {
    const pts = [];
    return {
      add(x, t) { pts.push([x, t]); while (pts.length > 5) pts.shift(); },
      v() {
        if (pts.length < 2) return 0;
        const [x0, t0] = pts[0], [x1, t1] = pts[pts.length - 1];
        return t1 - t0 > 0 ? ((x1 - x0) / (t1 - t0)) * 1000 : 0;
      },
    };
  }

  VC.phone = (ctx) => {
    const root = ctx.root;
    root.classList.add('c-app', 'c-app--phone');
    const TABS = [['home', 'route', 'Network'], ['library', 'film', 'Library'], ['activity', 'activity', 'Activity'], ['search', 'search', 'Search'], ['more', 'ellipsis', 'More']];
    root.innerHTML = `<div class="c-stack"></div>
      <nav class="c-tabbar" aria-label="Tabs">${TABS.map(([id, icon, label]) => `<button class="c-tab" data-tab="${id}" aria-label="${label}">${SS.icon(icon, { size: 24, stroke: 1.75 })}<span>${label}</span>${id === 'activity' ? '<b class="c-tab__badge c-num" data-badge></b>' : ''}</button>`).join('')}</nav>
      <div class="c-sheets"></div>`;
    const stackEl = $(root, '.c-stack');
    const sheetsEl = $(root, '.c-sheets');
    let stack = [];
    let tab = 'home';
    let sheet = null;
    let busy = false;
    const W = () => root.offsetWidth;
    const scaleK = () => root.getBoundingClientRect().width / root.offsetWidth || 1;
    const toast = (t, s, tone) => VC.toast(root, t, { sub: s, tone });

    function makePage(route) {
      const [name, id] = route.split('/');
      const def = name === 'work' ? workPage(id) : name === 'nodes' ? nodesPage() : { home: homePage, library: libraryPage, activity: activityPage, search: searchPage, more: morePage }[name]();
      const el = document.createElement('section');
      el.className = `c-pg ${def.cls || ''}`;
      const back = stack.length ? stack[stack.length - 1].title : '';
      el.innerHTML = `
        <header class="c-nb${def.over ? ' c-nb--over' : ''}">
          <div class="c-nb__in">
            <span class="c-nb__l">${stack.length ? `<button class="c-nb__back" data-back aria-label="Back to ${esc(back)}">${I('chevron-left', 24)}<span>${esc(back)}</span></button>` : ''}</span>
            <span class="c-nb__t">${esc(def.title)}</span>
            <span class="c-nb__r">${def.right || ''}</span>
          </div>
        </header>
        <div class="c-pg__scroll"><div class="c-pg__body">${def.large === false ? '' : `<h1 class="c-lt">${esc(def.title)}</h1>`}${def.html()}</div></div>
        <div class="c-pg__dim"></div>`;
      const page = { route, el, def, title: def.title, scroll: $(el, '.c-pg__scroll') };
      const nb = $(el, '.c-nb');
      const onScroll = () => {
        const lt = $(el, '.c-lt');
        const bar = nb.getBoundingClientRect().bottom;
        const edge = lt ? lt.getBoundingClientRect().bottom : bar + 40 - page.scroll.scrollTop;
        nb.classList.toggle('is-collapsed', edge <= bar);
        if (def.over) nb.classList.toggle('is-solid', edge <= bar + 90 * scaleK());
      };
      page.scroll.addEventListener('scroll', onScroll, { passive: true });
      page.sync = onScroll;
      def.mount?.(el, page);
      return page;
    }

    function setTab(name, keepRoute) {
      closeSheet(true);
      for (const p of stack) { p.def.destroy?.(); p.el.remove(); }
      stack = [];
      tab = name;
      const p = makePage(name);
      stackEl.appendChild(p.el);
      stack.push(p);
      p.sync();
      $$(root, '[data-tab]').forEach((b) => b.toggleAttribute('aria-current', b.dataset.tab === name));
      if (!keepRoute) ctx.setView(name);
    }

    function push(route, animate = true) {
      if (busy) return;
      const prev = stack[stack.length - 1];
      const p = makePage(route);
      stackEl.appendChild(p.el);
      stack.push(p);
      p.sync();
      ctx.setView(route);
      if (!animate) { prev.el.classList.add('is-under'); return; }
      busy = true;
      if (VC.reduced()) {
        p.el.style.opacity = '0';
        p.el.offsetWidth;
        p.el.style.transition = 'opacity 200ms ease';
        p.el.style.opacity = '1';
        setTimeout(() => { prev.el.classList.add('is-under'); p.el.style.transition = ''; busy = false; }, 210);
        return;
      }
      p.el.style.transform = 'translateX(100%)';
      p.el.offsetWidth;
      slide(p, prev, 0, { easing: 'cubic-bezier(0.32, 0.72, 0, 1)', duration: 440 }, () => { prev.el.classList.add('is-under'); busy = false; });
    }

    function slide(top, under, x, { easing, duration }, done) {
      const w = W();
      const t = `transform ${duration}ms ${easing}`;
      top.el.style.transition = t;
      under.el.style.transition = t;
      $(under.el, '.c-pg__dim').style.transition = `opacity ${duration}ms ${easing}`;
      top.el.style.transform = `translateX(${x}px)`;
      under.el.style.transform = `translateX(${(-0.28 * (w - x)).toFixed(1)}px)`;
      $(under.el, '.c-pg__dim').style.opacity = String(0.42 * (1 - x / w));
      top.el.classList.add('is-moving');
      setTimeout(() => {
        top.el.style.transition = under.el.style.transition = '';
        top.el.classList.remove('is-moving');
        done?.();
      }, duration + 20);
    }

    function pop(animate = true, spring) {
      if (stack.length < 2 || busy) return;
      const top = stack[stack.length - 1];
      const under = stack[stack.length - 2];
      under.el.classList.remove('is-under');
      const finish = () => {
        top.def.destroy?.();
        top.el.remove();
        stack.pop();
        under.el.style.transform = '';
        $(under.el, '.c-pg__dim').style.opacity = '';
        ctx.setView(under.route);
        busy = false;
      };
      if (!animate) return finish();
      busy = true;
      if (VC.reduced()) {
        top.el.style.transition = 'opacity 200ms ease';
        top.el.style.opacity = '0';
        return setTimeout(finish, 210);
      }
      if (!top.el.style.transform) { under.el.style.transform = 'translateX(-28%)'; $(under.el, '.c-pg__dim').style.opacity = '0.42'; under.el.offsetWidth; }
      slide(top, under, W(), spring || { easing: 'cubic-bezier(0.32, 0.72, 0, 1)', duration: 380 }, finish);
    }

    ctx.listen(stackEl, 'pointerdown', (e) => {
      if (stack.length < 2 || busy || sheet || e.button > 0) return;
      const rect = root.getBoundingClientRect();
      const k = scaleK();
      if ((e.clientX - rect.left) / k > 28) return;
      e.preventDefault();
      const top = stack[stack.length - 1];
      const under = stack[stack.length - 2];
      const x0 = e.clientX;
      const vt = tracker();
      let x = 0, dragging = false;
      under.el.classList.remove('is-under');
      const move = (ev) => {
        const dx = (ev.clientX - x0) / k;
        if (!dragging && dx < 6) return;
        dragging = true;
        x = Math.max(0, dx);
        vt.add(x, ev.timeStamp);
        top.el.style.transform = `translateX(${x}px)`;
        under.el.style.transform = `translateX(${(-0.28 * (W() - x)).toFixed(1)}px)`;
        $(under.el, '.c-pg__dim').style.opacity = String(0.42 * (1 - x / W()));
        top.el.classList.add('is-moving');
      };
      const up = () => {
        stackEl.removeEventListener('pointermove', move);
        stackEl.removeEventListener('pointerup', up);
        stackEl.removeEventListener('pointercancel', up);
        if (!dragging) { under.el.classList.add('is-under'); return; }
        const v = vt.v();
        const w = W();
        const goBack = x + project(v) > w / 2;
        const target = goBack ? w : 0;
        const dist = target - x || 1;
        const sp = VC.spring({ velocity: v / dist, response: 0.36 });
        if (goBack) pop(true, sp);
        else { busy = true; slide(top, under, 0, sp, () => { under.el.classList.add('is-under'); busy = false; }); }
      };
      stackEl.setPointerCapture?.(e.pointerId);
      stackEl.addEventListener('pointermove', move);
      stackEl.addEventListener('pointerup', up);
      stackEl.addEventListener('pointercancel', up);
    });

    function openSheet(html, { route, onClose, cls = '' } = {}) {
      closeSheet(true);
      const wrap = document.createElement('div');
      wrap.className = 'c-sheetwrap';
      wrap.innerHTML = `<div class="c-scrim" data-dismiss></div><div class="c-sheet ${cls}" role="dialog" aria-modal="true"><div class="c-sheet__grab" aria-hidden="true"></div>${html}</div>`;
      sheetsEl.appendChild(wrap);
      const el = $(wrap, '.c-sheet');
      const scrim = $(wrap, '.c-scrim');
      sheet = { wrap, el, scrim, route, onClose };
      if (route) ctx.setView(route);
      if (VC.reduced()) {
        el.style.opacity = '0';
        el.offsetWidth;
        el.style.transition = 'opacity 200ms ease';
        el.style.opacity = '1';
      } else {
        el.style.transform = 'translateY(100%)';
        scrim.style.opacity = '0';
        el.offsetWidth;
        el.style.transition = 'transform 420ms cubic-bezier(0.32, 0.72, 0, 1)';
        scrim.style.transition = 'opacity 420ms cubic-bezier(0.32, 0.72, 0, 1)';
        el.style.transform = 'translateY(0)';
        scrim.style.opacity = '1';
      }
      dragSheet(sheet);
      return sheet;
    }
    function closeSheet(instant, spring) {
      if (!sheet) return;
      const s = sheet;
      sheet = null;
      s.onClose?.();
      if (instant) return s.wrap.remove();
      const h = s.el.offsetHeight;
      const sp = spring || { easing: 'cubic-bezier(0.32, 0.72, 0, 1)', duration: 320 };
      s.el.style.transition = `transform ${sp.duration}ms ${sp.easing}`;
      s.scrim.style.transition = `opacity ${sp.duration}ms ${sp.easing}`;
      s.el.style.transform = `translateY(${h}px)`;
      s.scrim.style.opacity = '0';
      setTimeout(() => s.wrap.remove(), sp.duration + 30);
    }
    function dragSheet(s) {
      let y0 = 0, y = 0, dragging = false, vt = null, k = 1;
      s.el.addEventListener('pointerdown', (e) => {
        if (e.target.closest('button, input, a') && !e.target.closest('.c-sheet__grab')) return;
        e.preventDefault();
        y0 = e.clientY; y = 0; dragging = true; vt = tracker(); k = scaleK();
        s.el.setPointerCapture(e.pointerId);
        s.el.style.transition = 'none';
        s.scrim.style.transition = 'none';
      });
      s.el.addEventListener('pointermove', (e) => {
        if (!dragging) return;
        const dy = (e.clientY - y0) / k;
        y = dy < 0 ? -Math.sqrt(-dy) * 2 : dy;
        vt.add(y, e.timeStamp);
        s.el.style.transform = `translateY(${y}px)`;
        s.scrim.style.opacity = String(Math.max(0, 1 - y / s.el.offsetHeight));
      });
      const end = () => {
        if (!dragging) return;
        dragging = false;
        const v = vt.v();
        const h = s.el.offsetHeight;
        if (y + project(v, 0.99) > h * 0.4 || v > 700) {
          const sp = VC.spring({ velocity: Math.max(0, v) / Math.max(1, h - y), response: 0.32 });
          closeAndRoute(sp);
        } else {
          const sp = VC.spring({ velocity: -v / Math.max(1, Math.abs(y)), response: 0.34, damping: 0.86 });
          s.el.style.transition = `transform ${sp.duration}ms ${sp.easing}`;
          s.scrim.style.transition = `opacity ${sp.duration}ms ${sp.easing}`;
          s.el.style.transform = 'translateY(0)';
          s.scrim.style.opacity = '1';
        }
      };
      s.el.addEventListener('pointerup', end);
      s.el.addEventListener('pointercancel', end);
    }
    function closeAndRoute(spring) {
      const top = stack[stack.length - 1];
      closeSheet(false, spring);
      ctx.setView(top.route);
    }

    function workActions(w) {
      const job = VC.workJob(w);
      openSheet(`<div class="c-sheet__head">${VC.poster(w, 'c-sheet__poster')}<span><b>${esc(w.title)}</b><span>${esc(VC.workMeta(w))}</span></span></div>
        <div class="c-sheet__group">
          <button class="c-sheet__i" data-sa="search">${I('search')}Search now</button>
          <button class="c-sheet__i" data-sa="manual">${I('list-filter')}Manual search</button>
          <button class="c-sheet__i" data-sa="monitor">${I('eye')}Monitored<span class="c-sheet__v">On</span></button>
          <button class="c-sheet__i" data-sa="profile">${I('sliders-horizontal')}Profile<span class="c-sheet__v">${esc(VC.profile(w).name)}</span></button>
        </div>
        <div class="c-sheet__group">
          ${job?.state === 'failed' ? `<button class="c-sheet__i" data-sa="blocklist">${I('ban')}Blocklist and search again</button>` : ''}
          ${job?.state === 'stalled' ? `<button class="c-sheet__i" data-sa="find">${I('refresh-cw')}Find another release</button>` : ''}
          <button class="c-sheet__i" data-sa="plex">${I('external-link')}Open in Plex</button>
          <button class="c-sheet__i c-sheet__i--danger" data-sa="delete">${I('trash-2')}Delete files</button>
        </div>
        <button class="c-sheet__cancel" data-sa="cancel">Cancel</button>`, { route: `work/${w.id}/actions`, cls: 'c-sheet--actions' });
      sheet.work = w;
    }
    function sheetAction(a) {
      const w = sheet?.work;
      if (a === 'cancel') return closeAndRoute();
      if (!w) return;
      if (a === 'search') {
        const job = VC.workJob(w);
        if (job) toast(`${w.title} is already on its way`, `${VC.jobStage(job)} at ${VC.jobPath(job).join(' to ')}`);
        else if (w.status === 'missing' && w.type === 'movie') { VC.act.grab(w.id); toast(`Grabbed ${w.title}`, 'Downloading on ams-1'); }
        else toast(`Searched 4 indexers for ${w.title}`, 'Nothing passes the cutoff yet');
      } else if (a === 'blocklist') { VC.act.blocklistAndSearch(w.id); toast(`${w.title} is on the Blocklist`, 'A new release is downloading'); }
      else if (a === 'find') { VC.act.findRelease(w.id); toast(`New release for ${w.title}`, 'Kestrel · 64 seeders'); }
      else toast({ manual: 'Manual search', monitor: 'Monitoring', profile: 'Profile', plex: 'Open in Plex', delete: 'Delete files' }[a], 'Prototype: this action is not wired', 'muted');
      closeAndRoute();
    }

    function homePage() {
      let strip = null;
      let el;
      const status = () => {
        const net = VC.net();
        const html = net.sentence.map((p) => (typeof p === 'string' ? `<span>${esc(p)}</span>` : `<span class="c-tone-${p.tone}">${esc(p.text)}</span>`)).join('<span class="c-status__sep">·</span>');
        const t = $(el, '[data-status]');
        if (t.innerHTML !== html) t.innerHTML = html;
      };
      const disr = () => {
        const list = VC.disruptions();
        const html = list.map((x) => `<li class="c-pdisr" data-kind="${x.kind}"><span class="c-disr__mark"></span>${VC.bullet(x.line)}<span class="c-pdisr__b"><b>${esc(x.title)}</b><span>${esc(x.detail)}</span></span>
          ${x.kind === 'failed' ? `<button class="c-btn c-btn--s" data-pa="blocklist" data-work="${x.work}">Blocklist</button>` : x.job ? `<button class="c-btn c-btn--s" data-pa="find" data-work="${x.work}">Retry</button>` : ''}</li>`).join('') || '<li class="c-empty">Good service on every line.</li>';
        const host = $(el, '[data-disr]');
        if (host.dataset.v !== html) { host.dataset.v = html; host.innerHTML = html; }
        VC.set($(el, '[data-dc]'), String(list.length));
      };
      const dep = () => {
        const body = $(el, '[data-dep]');
        const list = VC.departures(5);
        const ids = list.map((x) => x.id).join();
        if (body.dataset.ids !== ids) {
          body.dataset.ids = ids;
          body.innerHTML = list.map((x) => `<tr data-dep="${x.id}" data-push="work/${x.work}"><td class="c-board__t c-num" data-f="t"></td><td class="c-td-b">${VC.bullet(x.line)}</td><td class="c-board__d"><span class="c-trunc"><b>${esc(x.title)}</b></span><span class="c-trunc c-dim">${esc(x.sub)} · ${esc(x.platform)}</span></td><td class="c-board__s" data-f="s"></td></tr>`).join('');
        }
        for (const x of list) {
          const tr = $(body, `[data-dep="${x.id}"]`);
          VC.flap($(tr, '[data-f="t"]'), VC.boardTime(x.at));
          const s = $(tr, '[data-f="s"]');
          VC.flap(s, x.status);
          s.dataset.tone = x.tone;
        }
      };
      return {
        title: 'Network',
        html: () => `<section class="c-pstatus"><div class="c-status__lines">${VC.LINES.map((l) => VC.bullet(l.id)).join('')}</div><p data-status></p></section>
          <section class="c-pcard c-pcard--strip"><div class="c-pstrip"></div></section>
          <h2 class="c-ph">Disruptions<span class="c-count c-num" data-dc></span></h2>
          <ul class="c-pcard c-plist" data-disr></ul>
          <h2 class="c-ph">Departures</h2>
          <div class="c-pcard"><table class="c-board c-board--ph"><tbody data-dep></tbody></table></div>`,
        mount(pageEl) {
          el = pageEl;
          strip = VC.stripMap($(el, '.c-pstrip'), ctx, { onStation: (name) => openSheet(`<div class="c-card c-card--sheet">${VC.nodeCard(name)}</div>`, { cls: 'c-sheet--station' }) });
          status(); disr(); dep();
        },
        tick: () => strip.tick(),
        second() { status(); dep(); strip.second(); },
        jobs() { strip.sync(); dep(); disr(); },
        refresh() { strip.render(); disr(); status(); },
      };
    }

    function libraryPage() {
      let type = 'movie';
      let filter = 'all';
      const FILTERS = [['all', 'All'], ['moving', 'In progress'], ['missing', 'Missing'], ['upgradable', 'Upgradable'], ['unreleased', 'Unreleased']];
      const GROUP = { moving: 'moving', stalled: 'moving', paused: 'moving', failed: 'missing', missing: 'missing', upgradable: 'upgradable', unreleased: 'unreleased' };
      const list = () => d.works.filter((w) => w.type === type && (filter === 'all' || GROUP[VC.statusOf(w).key] === filter)).sort((a, b) => b.added - a.added);
      let el;
      const label = (w, s) => (s.key === 'available' ? String(w.year) : s.label);
      const tile = (w) => {
        const s = VC.statusOf(w);
        return `<button class="c-tile" data-push="work/${w.id}" data-work="${w.id}"><span class="c-tile__art">${VC.poster(w)}</span><span class="c-tile__t">${esc(w.title)}</span><span class="c-tile__m"><span class="c-tile__s c-st-${s.key}" data-f="s">${esc(label(w, s))}</span></span>${s.key === 'moving' ? `<span class="c-tile__track" style="--ink:${VC.L[s.line].ink}"><span class="c-tile__fill" style="transform:scaleX(${s.p.toFixed(4)})"></span><span class="c-tile__train" style="left:${(s.p * 100).toFixed(2)}%"></span></span>` : ''}</button>`;
      };
      const render = () => {
        const ws = list();
        $(el, '[data-grid]').innerHTML = ws.length ? ws.map(tile).join('') : `<p class="c-empty">Nothing ${esc(FILTERS.find(([k]) => k === filter)[1].toLowerCase())} in ${type === 'movie' ? 'Movies' : 'Series'}.</p>`;
        $$(el, '[data-type]').forEach((b) => b.setAttribute('aria-selected', b.dataset.type === type));
        $$(el, '[data-filter]').forEach((b) => b.setAttribute('aria-pressed', b.dataset.filter === filter));
      };
      return {
        title: 'Library',
        html: () => `<div class="c-seg c-seg--full" role="tablist"><button role="tab" data-type="movie">Movies<span class="c-num">${d.works.filter((w) => w.type === 'movie').length}</span></button><button role="tab" data-type="series">Series<span class="c-num">${d.works.filter((w) => w.type === 'series').length}</span></button></div>
          <div class="c-chips">${FILTERS.map(([k, l]) => `<button class="c-chip" data-filter="${k}">${l}</button>`).join('')}</div>
          <div class="c-pgrid" data-grid></div>`,
        mount(pageEl) {
          el = pageEl;
          render();
          el.addEventListener('click', (e) => {
            const t = e.target.closest('[data-type]');
            if (t) { type = t.dataset.type; render(); }
            const f = e.target.closest('[data-filter]');
            if (f) { filter = f.dataset.filter; render(); }
          });
        },
        second() {
          for (const t of $$(el, '[data-work]')) {
            const w = d.W[t.dataset.work];
            const s = VC.statusOf(w);
            VC.set($(t, '[data-f="s"]'), label(w, s));
            const fill = $(t, '.c-tile__fill');
            if (fill && s.p != null) { fill.style.transform = `scaleX(${s.p.toFixed(4)})`; $(t, '.c-tile__train').style.left = (s.p * 100).toFixed(2) + '%'; }
          }
        },
        jobs: () => render(),
      };
    }

    const STATE_ORDER = ['running', 'paused', 'stalled', 'queued', 'failed', 'done'];
    function jobItem(j) {
      const line = VC.jobLine(j);
      const w = d.W[j.work];
      return `<li class="c-jrow" data-push="work/${w.id}" data-job="${j.id}" data-state="${j.state}"${VC.lit.has(j.id) ? ` data-lit="${j.id}"` : ''}>
        <div class="c-jrow__top"><b class="c-trunc">${esc(w.title)}${j.episode ? ` <span class="c-dim">${F.ep(j.episode)}</span>` : ''}</b><span class="c-num" data-f="pct">${j.state === 'failed' ? '' : F.pct(j.progress)}</span></div>
        <div class="c-jrow__mid"><span class="c-tone-${{ failed: 'failed', stalled: 'delays', paused: 'paused', queued: 'muted' }[j.state] || ''}" data-f="stage">${esc(VC.jobStage(j))}</span> · ${esc(VC.jobPath(j).join(' to '))} · ${esc(VC.target(j.target)?.short || '')}</div>
        ${j.state === 'failed' ? `<div class="c-jrow__msg c-tone-failed">${esc(j.message)}</div>` : VC.bar(j.progress, line)}
        ${j.state === 'running' ? `<div class="c-jrow__bot c-num"><span data-f="rate">${VC.jobRate(j)}</span><span data-f="eta">${VC.jobEta(j) ? VC.jobEta(j) + ' left' : ''}</span></div>` : ''}
      </li>`;
    }
    function activityPage() {
      let el;
      const render = () => {
        const groups = VC.LINES.map((l) => ({ l, jobs: d.jobs.filter((j) => VC.jobLine(j) === l.id && (j.state !== 'done' || VC.lit.has(j.id))) })).filter((g) => g.jobs.length);
        const net = VC.net();
        $(el, '[data-acts]').innerHTML = groups.map((g) => `<h2 class="c-ph c-ph--line">${VC.bullet(g.l.id)}${g.l.name}<span class="c-ph__s c-tone-${net.lines[g.l.id].state === 'good' ? '' : 'delays'}">${esc(net.lines[g.l.id].short)}</span></h2>
          <ul class="c-pcard c-plist">${g.jobs.sort((a, b) => STATE_ORDER.indexOf(a.state) - STATE_ORDER.indexOf(b.state)).map(jobItem).join('')}</ul>`).join('') || '<p class="c-empty">Nothing is moving. Trains appear here the moment a job starts.</p>';
      };
      return {
        title: 'Activity',
        html: () => '<div data-acts></div>',
        mount(pageEl) { el = pageEl; render(); },
        tick() {
          for (const li of $$(el, '[data-job]')) {
            const j = d.jobs.find((x) => x.id === li.dataset.job);
            const f = $(li, '.c-bar__fill');
            if (j && f && j.state === 'running') f.style.transform = `scaleX(${j.progress.toFixed(4)})`;
          }
        },
        second() {
          for (const li of $$(el, '[data-job]')) {
            const j = d.jobs.find((x) => x.id === li.dataset.job);
            if (!j) continue;
            VC.set($(li, '[data-f="pct"]'), j.state === 'failed' ? '' : F.pct(j.progress));
            VC.set($(li, '[data-f="rate"]'), VC.jobRate(j));
            VC.set($(li, '[data-f="eta"]'), VC.jobEta(j) ? VC.jobEta(j) + ' left' : '');
            VC.flap($(li, '[data-f="stage"]'), VC.jobStage(j));
          }
        },
        jobs: () => render(),
      };
    }

    function searchPage() {
      let el, q = '';
      const row = (w) => {
        const s = VC.statusOf(w);
        return `<li class="c-srow" data-push="work/${w.id}">${VC.poster(w, 'c-srow__p')}<span class="c-srow__b"><b class="c-trunc">${esc(w.title)}</b><span class="c-trunc c-dim">${w.type === 'movie' ? 'Movie' : 'Series'} · ${w.year} · <span class="c-st-${s.key}">${esc(s.key === 'available' ? 'In the library' : s.label)}</span></span></span>${I('chevron-right', 18)}</li>`;
      };
      const render = () => {
        const res = VC.searchWorks(q);
        const nodes = q ? d.nodes.filter((n) => n.name.includes(q.toLowerCase())) : [];
        const box = $(el, '[data-res]');
        if (!q) {
          const moving = d.works.filter((w) => VC.workJob(w));
          box.innerHTML = `<h2 class="c-ph">On the network</h2><ul class="c-pcard c-plist">${moving.map(row).join('')}</ul>
            <h2 class="c-ph">Recently added</h2><ul class="c-pcard c-plist">${[...d.works].sort((a, b) => b.added - a.added).slice(0, 4).map(row).join('')}</ul>`;
        } else if (!res.length && !nodes.length) {
          box.innerHTML = `<p class="c-empty">No Work or Node matches “${esc(q)}”. Search by title, genre, or a Node such as forge.</p>`;
        } else {
          box.innerHTML = `${res.length ? `<h2 class="c-ph">Works<span class="c-count c-num">${res.length}</span></h2><ul class="c-pcard c-plist">${res.slice(0, 12).map(row).join('')}</ul>` : ''}
            ${nodes.length ? `<h2 class="c-ph">Nodes</h2><ul class="c-pcard c-plist">${nodes.map((n) => `<li class="c-srow" data-station="${n.id}">${VC.bullets(VC.nodeLines(n))}<span class="c-srow__b"><b>${esc(n.name)}</b><span class="c-dim">${esc(n.hardware)}</span></span></li>`).join('')}</ul>` : ''}`;
        }
      };
      return {
        title: 'Search',
        html: () => `<label class="c-psearch">${I('search', 18)}<input type="search" placeholder="Works, Nodes, genres" enterkeyhint="search" autocomplete="off" autocapitalize="none" aria-label="Search"></label><div data-res></div>`,
        mount(pageEl) {
          el = pageEl;
          const input = $(el, 'input');
          input.addEventListener('input', () => { q = input.value; render(); });
          el.addEventListener('click', (e) => { const n = e.target.closest('[data-station]'); if (n) openSheet(`<div class="c-card c-card--sheet">${VC.nodeCard(n.dataset.station)}</div>`, { cls: 'c-sheet--station' }); });
          render();
        },
      };
    }

    function morePage() {
      const missingMovies = d.works.filter((w) => w.type === 'movie' && w.status === 'missing').length;
      const missingEps = VC.act.missingEpisodes().length;
      const pending = d.requests.filter((r) => r.state === 'pending').length;
      const suspended = d.nodes.filter((n) => VC.nodeState(n) === 'suspended').length;
      const row = (icon, label, value = '', act = '') => `<li><button class="c-mrow" ${act}><span class="c-mrow__i">${SS.icon(icon, { size: 20, stroke: 1.75 })}</span><span class="c-mrow__l">${label}</span><span class="c-mrow__v">${value}</span>${I('chevron-right', 17)}</button></li>`;
      return {
        title: 'More',
        html: () => `
          <h2 class="c-ph">Discover</h2>
          <ul class="c-pcard c-mlist">${row('compass', 'Discover movies', '', 'data-ma="discover"')}${row('tv', 'Discover series', '', 'data-ma="discover"')}${row('inbox', 'Requests', pending ? `<b class="c-mbadge c-num">${pending}</b>` : '', 'data-ma="requests"')}</ul>
          <h2 class="c-ph">Missing</h2>
          <ul class="c-pcard c-mlist">${row('film', 'Missing movies', `<span class="c-num">${missingMovies}</span>`, 'data-ma="missing"')}${row('list', 'Missing episodes', `<span class="c-num">${missingEps}</span>`, 'data-ma="missing"')}</ul>
          <h2 class="c-ph">Settings</h2>
          <ul class="c-pcard c-mlist">${row('sliders-horizontal', 'Profiles and Targets', `${d.profiles.length} and ${d.targets.length}`, 'data-ma="settings"')}${row('search', 'Indexers', '<span class="c-tone-delays">Halide limited</span>', 'data-ma="settings"')}${row('download', 'Download Clients', 'Built-in', 'data-ma="settings"')}${row('monitor-play', 'Media Servers', 'Plex', 'data-ma="settings"')}${row('users', 'Accounts', String(d.accounts.length), 'data-ma="settings"')}${row('bell', 'Notifications', 'Push and email', 'data-ma="settings"')}</ul>
          <h2 class="c-ph">System</h2>
          <ul class="c-pcard c-mlist">${row('server', 'Nodes', `<span class="${suspended ? 'c-dim' : ''}">${d.nodes.length} · ${suspended} suspended</span>`, 'data-push="nodes"')}${row('refresh-cw', 'Updates', '<span class="c-tone-arrived">2.3.1 is ready</span>', 'data-ma="settings"')}${row('scroll-text', 'Logs', '', 'data-ma="settings"')}</ul>
          <p class="c-mfoot">SlipStream 2.3.0 · atlas is the Controller<br>Signed in as Alex Rowan, Admin<br>Synthetic data for the prototype</p>`,
        mount(el) {
          el.addEventListener('click', (e) => {
            const a = e.target.closest('[data-ma]');
            if (!a) return;
            if (a.dataset.ma === 'missing') { setTab('library'); return; }
            toast($(a, '.c-mrow__l').textContent, 'Prototype: this page lives on the desk', 'muted');
          });
        },
      };
    }

    function nodesPage() {
      return {
        title: 'Nodes',
        html: () => `<ul class="c-pcard c-plist">${d.nodes.map((n) => {
          const s = VC.nodeState(n);
          return `<li class="c-nrow" data-station="${n.id}"><span class="c-srow__b"><b>${esc(n.name)}</b><span class="c-dim c-trunc">${esc(n.hardware)}</span></span>${VC.bullets(VC.nodeLines(n))}<span class="c-nstate" data-s="${s}">${{ good: 'Online', delays: 'Delays', suspended: 'Suspended', paused: 'Paused' }[s]}</span></li>`;
        }).join('')}</ul>`,
        mount(el) { el.addEventListener('click', (e) => { const n = e.target.closest('[data-station]'); if (n) openSheet(`<div class="c-card c-card--sheet">${VC.nodeCard(n.dataset.station)}</div>`, { cls: 'c-sheet--station' }); }); },
      };
    }

    function workPage(id) {
      const w = d.W[id];
      let el;
      let season = w.type === 'series' ? (VC.workJob(w)?.episode?.[0] || w.seasons.length) : 0;
      const nowLine = () => {
        const j = VC.workJob(w);
        if (j) return `${VC.bullet(VC.jobLine(j))}<span><b class="c-tone-${{ failed: 'failed', stalled: 'delays' }[j.state] || ''}">${esc(VC.jobStage(j))}${j.episode ? ' ' + F.ep(j.episode) : ''}</b> <span class="c-dim">${j.state === 'failed' ? esc(j.message) : `${F.pct(j.progress)} · ${esc(VC.jobPath(j).join(' to '))}`}</span></span>`;
        const s = VC.statusOf(w);
        return `<span class="c-tone-${{ available: 'arrived', upgradable: 'delays' }[s.key] || 'muted'}">${esc(s.label)}</span>${w.type === 'series' && w.nextAir ? `<span class="c-dim"> · next episode ${esc(F.until(w.nextAir))}</span>` : ''}`;
      };
      const coverage = () => VC.coverage(w).filter((c) => c.inProfile).map((c) => `<li class="c-covrow" data-s="${c.state}"><span class="c-covrow__t"><b>${esc(c.target.name)}</b><span class="c-dim">${c.target.kind === 'derived' ? 'Derived from ' + esc(VC.targetName(c.target.from)) : 'Downloaded'}</span></span><span class="c-covrow__v">${c.state === 'moving' ? `${VC.bar(c.p, VC.jobLine(c.job))}<span class="c-num">${F.pct(c.job.progress)}</span>` : esc(F.coverageLabel[c.raw] || c.raw)}</span></li>`).join('');
      const seasons = () => {
        const S = w.seasons[season - 1];
        return `<div class="c-seg c-seg--full">${w.seasons.map((x) => `<button role="tab" aria-selected="${x.n === season}" data-season="${x.n}">Season ${x.n}</button>`).join('')}</div>
          <ul class="c-pcard c-plist">${S.episodes.map((e) => {
            const j = d.jobs.find((x) => x.work === w.id && x.episode && x.episode[0] === S.n && x.episode[1] === e.n && x.state !== 'done');
            const st = j ? `<span class="c-num">${esc(VC.jobVerb(j))} ${F.pct(j.progress)}</span>` : { available: '<span class="c-tone-arrived">Available</span>', missing: '<span class="c-dim">Missing</span>', failed: '<span class="c-tone-failed">Failed</span>', unreleased: `<span class="c-dim">${esc(F.until(e.air))}</span>`, downloading: '<span class="c-dim">Waiting</span>' }[e.status];
            return `<li class="c-eprow"><span class="c-num c-dim">${e.n}</span><b class="c-trunc">${esc(e.title)}</b>${st}</li>`;
          }).join('')}</ul>`;
      };
      return {
        title: w.title,
        cls: 'c-pg--work',
        over: true,
        large: false,
        right: `<button class="c-nb__btn" data-actions aria-label="Actions for ${esc(w.title)}">${I('ellipsis', 22)}</button>`,
        html: () => `<div class="c-wkhero"><img src="${SS.art.backdrop(w, { w: 800, h: 640 })}" alt=""><div class="c-wkhero__scrim"></div></div>
          <div class="c-wkhead">${VC.poster(w, 'c-wkhead__p')}<div class="c-wkhead__b"><h1 class="c-lt">${esc(w.title)}</h1><p class="c-dim">${esc(VC.workMeta(w))}</p></div></div>
          <p class="c-wknow" data-now>${nowLine()}</p>
          <div class="c-wkacts"><button class="c-btn c-btn--primary" data-wa="search">${I('search', 17)}Search</button><button class="c-btn" data-actions>${I('ellipsis', 17)}Actions</button></div>
          <section class="c-pcard c-pcard--route" data-route>${VC.routeHTML(w)}</section>
          ${w.type === 'series' ? `<h2 class="c-ph">Seasons</h2><div data-seasons>${seasons()}</div>` : ''}
          <h2 class="c-ph">Targets</h2><ul class="c-pcard c-plist" data-cov>${coverage()}</ul>
          ${w.type === 'movie' && w.files.length ? `<h2 class="c-ph">Files</h2><ul class="c-pcard c-plist">${w.files.map((f) => `<li class="c-frow"><span class="c-frow__t"><b>${esc(VC.targetName(f.target))}</b><span class="c-dim c-num">${F.bytes(f.size)} · ${esc(f.resolution)}</span></span><span class="c-trunc c-dim">${esc(f.video)} · ${esc(f.audio)}</span></li>`).join('')}</ul>` : ''}
          <p class="c-wkover">${esc(w.overview)}</p>`,
        mount(pageEl) {
          el = pageEl;
          el.addEventListener('click', (e) => {
            if (e.target.closest('[data-actions]')) { workActions(w); return; }
            const s = e.target.closest('[data-season]');
            if (s) { season = +s.dataset.season; $(el, '[data-seasons]').innerHTML = seasons(); }
            const a = e.target.closest('[data-wa]');
            if (a && !a.disabled) {
              a.classList.add('is-loading');
              a.disabled = true;
              setTimeout(() => {
                a.classList.remove('is-loading');
                a.disabled = false;
                const job = VC.workJob(w);
                if (job) toast(`${w.title} is already on its way`, `${VC.jobStage(job)} at ${VC.jobPath(job).join(' to ')}`);
                else if (w.status === 'missing' && w.type === 'movie') { VC.act.grab(w.id); toast(`Grabbed ${w.title}`, 'Downloading on ams-1'); }
                else toast('Searched 4 indexers', 'Nothing passes the cutoff yet');
              }, 800);
            }
          });
        },
        second() {
          const now = $(el, '[data-now]');
          const html = nowLine();
          if (now.innerHTML !== html) now.innerHTML = html;
          const r = VC.route(w);
          const routeEl = $(el, '[data-route]');
          const stops = $$(routeEl, '.c-stop');
          if (stops.map((s) => s.dataset.state).join() !== r.stops.map((s) => s.state).join()) routeEl.innerHTML = VC.routeHTML(w);
          else r.stops.forEach((s, i) => VC.set($(stops[i], '[data-f="detail"]'), s.detail || ''));
          const cov = coverage();
          const c = $(el, '[data-cov]');
          if (c.dataset.v !== cov) { c.dataset.v = cov; c.innerHTML = cov; }
        },
      };
    }

    ctx.listen(root, 'click', (e) => {
      const t = e.target;
      if (t.closest('[data-back]')) return pop();
      const tb = t.closest('[data-tab]');
      if (tb) {
        if (tb.dataset.tab !== tab) setTab(tb.dataset.tab);
        else if (stack.length > 1) { while (stack.length > 1) pop(false); ctx.setView(tab); }
        else stack[0].scroll.scrollTo({ top: 0, behavior: VC.reduced() ? 'auto' : 'smooth' });
        return;
      }
      if (t.closest('[data-dismiss]')) return closeAndRoute();
      const sa = t.closest('[data-sa]');
      if (sa) return sheetAction(sa.dataset.sa);
      const pa = t.closest('[data-pa]');
      if (pa) {
        const w = d.W[pa.dataset.work];
        if (pa.dataset.pa === 'blocklist') { VC.act.blocklistAndSearch(w.id); toast(`${w.title} is on the Blocklist`, 'A new release is downloading'); }
        else { VC.act.findRelease(w.id); toast(`New release for ${w.title}`, 'Kestrel · 64 seeders'); }
        return;
      }
      const p = t.closest('[data-push]');
      if (p) push(p.dataset.push);
    });
    ctx.listen(root, 'pointerover', (e) => { const lit = e.target.closest('[data-lit]'); if (lit) VC.notice(lit); });

    const top = () => stack[stack.length - 1];
    const badge = () => {
      const n = d.jobs.filter((j) => j.state === 'running').length;
      const b = $(root, '[data-badge]');
      VC.set(b, String(n));
      b.hidden = !n;
    };
    ctx.on('tick', () => top()?.def.tick?.());
    ctx.on('second', () => { top()?.def.second?.(); badge(); });
    const jobs = (j) => { if (j) VC.light(j.id); for (const p of stack) p.def.jobs?.(); badge(); };
    ctx.on('job:new', jobs);
    ctx.on('job:done', jobs);
    ctx.on('job:update', () => jobs());
    ctx.on('alerts:update', () => { for (const p of stack) { p.def.refresh?.(); p.def.jobs?.(); } });

    const [name, id, extra] = (ctx.view || 'home').split('/');
    if (name === 'work' && d.W[id]) {
      setTab('library', true);
      push(`work/${id}`, false);
      if (extra === 'actions') workActions(d.W[id]);
    } else if (name === 'nodes') { setTab('more', true); push('nodes', false); }
    else setTab(TABS.some(([t]) => t === name) ? name : 'home', true);
    badge();
    return { destroy() { for (const p of stack) p.def.destroy?.(); } };
  };
})();
