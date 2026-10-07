// PROTOTYPE. House Lights: the admin desk.
(function () {
  const SS = window.SS, VB = window.VB;
  const D = SS.data, F = SS.fmt, ART = SS.art;
  const esc = F.esc;
  const icon = VB.icon;

  const NAV = [['home', 'Tonight'], ['library', 'Movies'], ['series', 'Series'], ['requests', 'Requests'], ['activity', 'Activity'], ['nodes', 'Nodes']];
  const parse = (v) => {
    const [a, b, c] = (v || 'home').split('/');
    if (a === 'work' && D.W[b]) return { name: 'work', id: b, sub: c || '' };
    return { name: NAV.some((n) => n[0] === a) ? a : 'home' };
  };
  const navOf = (r) => (r.name === 'work' ? (D.W[r.id].type === 'series' ? 'series' : 'library') : r.name);
  const viewOf = (r) => (r.name === 'work' ? `work/${r.id}${r.sub ? '/' + r.sub : ''}` : r.name);
  const LABEL = Object.fromEntries(NAV);

  VB.local.lib = VB.local.lib || {
    movie: { filter: 'all', sort: 'added', view: 'grid', size: 'm' },
    series: { filter: 'all', sort: 'added', view: 'grid', size: 'm' },
  };
  const FILTERS = {
    movie: [['all', 'All'], ['available', 'Available'], ['downloading', 'Downloading'], ['missing', 'Missing'], ['upgradable', 'Upgradable'], ['unreleased', 'Unreleased'], ['failed', 'Failed']],
    series: [['all', 'All'], ['continuing', 'Continuing'], ['ended', 'Ended'], ['missing', 'Missing episodes'], ['downloading', 'Downloading']],
  };
  const inFilter = (w, f) => {
    if (f === 'all') return true;
    if (w.type === 'series') return f === 'continuing' ? w.continuing : f === 'ended' ? !w.continuing : f === 'missing' ? w.counts.missing > 0 : !!VB.acq(w.id);
    return VB.status(w).s === f;
  };
  const bare = (t) => t.replace(/^The /, '');
  const SORTS = {
    added: ['Recently added', (a, b) => b.added - a.added],
    title: ['Title', (a, b) => bare(a.title).localeCompare(bare(b.title))],
    year: ['Year', (a, b) => b.year - a.year || bare(a.title).localeCompare(bare(b.title))],
    rating: ['Rating', (a, b) => (b.rating || 0) - (a.rating || 0)],
  };
  const SIZES = { s: 128, m: 164, l: 212 };
  const EMPTY = {
    available: 'No Works are available yet. When a release meets a Target it is imported and shows here.',
    downloading: 'Nothing is downloading. Missing Works start here the moment a release meets their Profile.',
    missing: 'Nothing is missing. Every monitored Work has a file for each Target it wants.',
    upgradable: 'Nothing to upgrade. Every file meets its Target cutoff.',
    unreleased: 'Nothing unreleased. Works you add before release wait here until their date.',
    failed: 'Nothing has failed. A failed grab or import shows here with its reason and a fix.',
    continuing: 'No continuing series.', ended: 'No ended series.',
  };

  VB.wide = function (ctx) {
    const { root } = ctx;
    root.innerHTML = `<div class="b-app b-desk">
      <div class="b-spillhost"></div>
      <main class="b-main" tabindex="-1"><div class="b-view"></div></main>
      <header class="b-bar">${barHTML()}</header>
      <div class="b-layer"></div>
    </div>`;
    const app = root.querySelector('.b-app');
    const main = root.querySelector('.b-main');
    const viewEl = root.querySelector('.b-view');
    const bar = root.querySelector('.b-bar');
    const spill = VB.spill(root.querySelector('.b-spillhost'));
    const live = VB.live(ctx);
    const pop = VB.popover(ctx, root);
    let route = parse(ctx.view);
    const stack = [];
    let hoverT = 0, devTimer = 0;

    const now = VB.nowShowing(ctx, (h) => { if (route.name === 'home') swapHero(h); });

    function barHTML() {
      return `<a class="b-mark" href="${VB.href('home')}" data-go="home"><span class="b-mark__glyph" aria-hidden="true"></span>SlipStream</a>
        <nav class="b-nav" aria-label="Primary">${NAV.map(([k, l]) => `<a class="b-nav__i" href="${VB.href(k)}" data-go="${k}" data-nav="${k}">${l}${k === 'requests' ? ' <span class="b-count b-count--amber" data-reqcount></span>' : k === 'activity' ? ' <span class="b-count" data-lv="active"></span>' : ''}</a>`).join('')}<span class="b-nav__ind" aria-hidden="true"></span></nav>
        <div class="b-bar__r">
          <button class="b-search" type="button" data-act="palette">${icon('search', 16)}<span>Search</span><kbd>⌘K</kbd></button>
          <button class="b-iconbtn" type="button" data-act="tasks" aria-label="Scheduled tasks" aria-expanded="false">${icon('list-checks', 18)}</button>
          <button class="b-iconbtn" type="button" data-act="bell" aria-label="Notifications" aria-expanded="false">${icon('bell', 18)}<i class="b-iconbtn__dot" aria-hidden="true"></i></button>
          <button class="b-acct" type="button" data-act="account" aria-label="Account" aria-expanded="false">${VB.av(D.me)}</button>
        </div>`;
    }

    function updateBar() {
      const k = navOf(route);
      let act = null;
      for (const a of bar.querySelectorAll('[data-nav]')) {
        const on = a.dataset.nav === k;
        a.toggleAttribute('aria-current', on);
        if (on) a.setAttribute('aria-current', 'page');
        if (on) act = a;
      }
      const ind = bar.querySelector('.b-nav__ind');
      if (act) {
        ind.style.opacity = '1';
        ind.style.transform = `translateX(${act.offsetLeft + act.offsetWidth / 2 - 9}px)`;
      } else ind.style.opacity = '0';
      const n = VB.reqs().filter((r) => VB.reqState(r) === 'pending').length;
      const rc = bar.querySelector('[data-reqcount]');
      rc.textContent = n || '';
      rc.hidden = !n;
      bar.querySelector('[data-lv="active"]').textContent = VB.LV.active();
    }

    // ---------- Navigation ----------
    // Opening a Work is house lights down: the poster carries into the detail while the room dims.
    function go(view, { kbd = false, el = null, back = false, y = 0 } = {}) {
      const next = parse(view);
      if (!back && viewOf(next) === viewOf(route)) return;
      if (!back) stack.push({ view: viewOf(route), y: main.scrollTop });
      const prev = route;
      const update = () => {
        route = next;
        ctx.setView(viewOf(route));
        render();
        main.scrollTop = back ? y : 0;
        if (back && prev.name === 'work') {
          const t = viewEl.querySelector(`[data-open="${prev.id}"] .b-tile__art > *`);
          if (t) t.style.viewTransitionName = 'b-poster';
        }
      };
      const morph = (el && el.isConnected && next.name === 'work') || (back && prev.name === 'work');
      if (morph && el) el.style.viewTransitionName = 'b-poster';
      document.documentElement.dataset.bvt = morph ? (back ? 'up' : 'down') : 'nav';
      const t = VB.vt(update, { kbd });
      const clear = () => {
        for (const x of viewEl.querySelectorAll('.b-tile__art > *')) x.style.viewTransitionName = '';
        delete document.documentElement.dataset.bvt;
      };
      if (t) t.finished.finally(clear); else { clear(); main.focus({ preventScroll: true }); }
    }
    function back(kbd) {
      const prev = stack.pop() || { view: route.name === 'work' ? navOf(route) : 'home', y: 0 };
      go(prev.view, { kbd, back: true, y: prev.y });
    }

    // ---------- Views ----------
    const VIEWS = {
      home: { html: homeHTML, after: afterHome, light: () => D.W[now.get().work] },
      library: { html: () => libraryHTML('movie'), after: afterLibrary, light: () => null },
      series: { html: () => libraryHTML('series'), after: afterLibrary, light: () => null },
      work: { html: workHTML, after: afterWork, light: () => D.W[route.id] },
      activity: { html: activityHTML, after: () => {}, light: () => null },
      nodes: { html: nodesHTML, after: () => {}, light: () => null },
      requests: { html: requestsHTML, after: () => {}, light: () => null },
    };
    function render() {
      clearTimeout(devTimer);
      live.reset();
      pop.close();
      const v = VIEWS[route.name];
      app.dataset.route = route.name;
      viewEl.innerHTML = v.html(route);
      v.after(viewEl);
      live.scan(viewEl);
      VB.fillStills(viewEl);
      spill.set(v.light());
      spill.dim(1);
      bar.classList.remove('is-solid');
      updateBar();
      document.title = route.name === 'work' ? `${D.W[route.id].title} · SlipStream` : `${LABEL[route.name]} · SlipStream`;
    }

    // Home: the latest arrival on the screen, its light on the house.
    function heroCopy(h) {
      const w = D.W[h.work];
      const t = D.targets.find((x) => x.id === h.target);
      const req = D.requests.find((r) => r.work === w.id && r.state === 'available');
      let meta;
      if (h.episode) {
        const e = w.seasons[h.episode[0] - 1].episodes[h.episode[1] - 1];
        meta = [`${F.ep(h.episode)} · ${e.title}`, w.network, `${e.runtime} min`];
      } else meta = [String(w.year), w.genres.join(', '), F.runtime(w.runtime), w.cert];
      return `<div class="b-hero__copy" style="${VB.tint(w)}">
        ${VB.logo(w, { maxW: 600, max: 88, min: 40 })}
        <p class="b-meta">${meta.filter(Boolean).map((m) => `<span>${esc(m)}</span>`).join('')}</p>
        <p class="b-hero__ov">${esc(h.episode ? w.tagline : w.overview)}</p>
        <p class="b-hero__arr">Now showing. Arrived <span data-lv="ago|${h.at.getTime()}">${esc(F.ago(h.at))}</span> in ${esc(t.name)}${req ? `, for ${esc(VB.name(req.by))}` : ''}.</p>
        <div class="b-hero__act">
          <button class="b-btn b-btn--primary b-btn--lg" type="button" data-cmd="play" data-work="${w.id}">${icon('play', 18)}Play on Plex</button>
          <a class="b-btn b-btn--veil b-btn--lg" href="${VB.href('work/' + w.id)}" data-open="${w.id}">Details</a>
        </div>
      </div>`;
    }
    const heroImg = (h) => `<img class="b-hero__img is-on" src="${ART.backdrop(h.work, { w: 1680, h: 720 })}" alt="">`;
    function homeHTML() {
      const h = now.get();
      const arrivals = [];
      const seenA = new Set();
      for (const x of D.history) if ((x.kind === 'imported' || x.kind === 'upgraded') && !seenA.has(x.work)) { seenA.add(x.work); arrivals.push(x); }
      return `<section class="b-hero" aria-label="Now showing">
          <div class="b-hero__frame">${heroImg(h)}<div class="b-hero__shade"></div></div>
          <div class="b-hero__body">
            <div class="b-hero__slot">${heroCopy(h)}</div>
            <aside class="b-hero__aisle" aria-labelledby="b-aisle-h"><h2 class="b-h3" id="b-aisle-h">Needs you</h2><div data-aisle>${VB.aisleList(VB.aisle())}</div></aside>
          </div>
        </section>
        <div class="b-home__lower">
          <section class="b-sec" aria-labelledby="b-dev-h">
            <header class="b-sec__h"><h2 class="b-h2" id="b-dev-h">Developing</h2><p class="b-sec__aside"><span data-lv="down">${F.rate(SS.sim.totals().down)}</span> down</p><a class="b-link" href="${VB.href('activity')}" data-go="activity">Activity${icon('chevron-right', 16)}</a></header>
            <div class="b-drow b-house" data-devrow>${devRow()}</div>
          </section>
          <section class="b-sec" aria-labelledby="b-cs-h">
            <header class="b-sec__h"><h2 class="b-h2" id="b-cs-h">Coming soon</h2></header>
            <div class="b-cslist b-house">${VB.comingSoon(4).map(VB.comingCard).join('')}</div>
          </section>
        </div>
        <section class="b-sec b-sec--pad" aria-labelledby="b-arr-h">
          <header class="b-sec__h"><h2 class="b-h2" id="b-arr-h">Arrived lately</h2></header>
          <div class="b-arow b-house">${arrivals.slice(0, 8).map((x) => arrivedTile(x)).join('')}</div>
        </section>
        <footer class="b-nodesline"><p>${VB.nodesLine()} <a class="b-link" href="${VB.href('nodes')}" data-go="nodes">Nodes${icon('chevron-right', 16)}</a></p></footer>`;
    }
    const arrivedTile = (x) => {
      const w = D.W[x.work];
      const what = x.episode ? F.ep(x.episode) : x.kind === 'upgraded' ? 'Upgraded' : D.targets.find((t) => t.id === x.target).short;
      return `<a class="b-tile b-tile--lit" href="${VB.href('work/' + w.id)}" data-open="${w.id}" data-key="${w.id}" style="${VB.tint(w)}"><div class="b-tile__art">${VB.poster(w)}</div><div class="b-dtile__cap"><span class="b-tile__t">${esc(w.title)}</span><span class="b-dtile__s">${esc(what)} · <span data-lv="ago|${x.at.getTime()}">${esc(F.ago(x.at))}</span></span></div></a>`;
    };
    const devRow = () => {
      const items = VB.developing();
      return items.length ? items.map(VB.devTile).join('') : `<p class="b-empty">Nothing is developing. New grabs appear here in monochrome and fill with colour as they download.</p>`;
    };
    function afterHome() {}
    // An Item that finishes stays lit in the row for a moment, then the row closes the gap.
    ctx.on('job:done', (j) => {
      if (route.name !== 'home' || j.kind !== 'import') return;
      clearTimeout(devTimer);
      devTimer = setTimeout(() => refreshDevRow(viewEl.querySelector('[data-devrow]')), 4200);
    });
    async function refreshDevRow(row) {
      if (!row || !row.isConnected) return;
      const keep = new Set(VB.developing().map((x) => x.key));
      await VB.leave([...row.children].filter((c) => c.dataset.key && !keep.has(c.dataset.key)));
      VB.flip(row, () => { row.innerHTML = devRow(); live.scan(row); });
    }
    function swapHero(h) {
      const frame = viewEl.querySelector('.b-hero__frame');
      const slot = viewEl.querySelector('.b-hero__slot');
      if (!frame) return;
      const old = frame.querySelector('.b-hero__img.is-on');
      frame.insertAdjacentHTML('afterbegin', heroImg(h).replace('is-on', ''));
      const img = frame.firstElementChild;
      requestAnimationFrame(() => requestAnimationFrame(() => { img.classList.add('is-on'); old?.classList.remove('is-on'); }));
      setTimeout(() => old?.remove(), 900);
      const cur = slot.firstElementChild;
      cur.classList.add('is-out');
      setTimeout(() => { slot.innerHTML = heroCopy(h); live.scan(slot); slot.firstElementChild.classList.add('is-in'); }, 260);
      spill.set(D.W[h.work]);
    }

    // Library: poster wall with the house dimming around focus, or a table.
    function libraryHTML(type) {
      const st = VB.local.lib[type];
      const all = D.works.filter((w) => w.type === type);
      const list = all.filter((w) => inFilter(w, st.filter)).sort(SORTS[st.sort][1]);
      const counts = Object.fromEntries(FILTERS[type].map(([k]) => [k, all.filter((w) => inFilter(w, k)).length]));
      const title = type === 'movie' ? 'Movies' : 'Series';
      return `<div class="b-page">
        <header class="b-page__h"><h1 class="b-h1">${title}</h1><p class="b-page__sum">${F.plural(all.length, type === 'movie' ? 'movie' : 'series', type === 'movie' ? 'movies' : 'series')} · ${type === 'movie' ? `${counts.available} available, ${counts.downloading} downloading` : `${all.reduce((s, w) => s + w.counts.available, 0)} episodes`}</p></header>
        <div class="b-toolbar">
          <div class="b-seg" role="radiogroup" aria-label="Status">${FILTERS[type].map(([k, l]) => `<button type="button" role="radio" aria-checked="${st.filter === k}" data-filter="${k}">${l}<span class="b-seg__n">${counts[k]}</span></button>`).join('')}</div>
          <div class="b-toolbar__r">
            <label class="b-select"><span class="b-sr">Sort by</span><select data-sort>${Object.entries(SORTS).map(([k, [l]]) => `<option value="${k}"${st.sort === k ? ' selected' : ''}>${l}</option>`).join('')}</select>${icon('chevron-down', 14)}</label>
            ${st.view === 'grid' ? `<div class="b-seg b-seg--icons" role="radiogroup" aria-label="Poster size">${['s', 'm', 'l'].map((k) => `<button type="button" role="radio" aria-checked="${st.size === k}" data-size="${k}" aria-label="${{ s: 'Small', m: 'Medium', l: 'Large' }[k]} posters"><i class="b-sz b-sz--${k}"></i></button>`).join('')}</div>` : ''}
            <div class="b-seg b-seg--icons" role="radiogroup" aria-label="View">
              <button type="button" role="radio" aria-checked="${st.view === 'grid'}" data-viewmode="grid" aria-label="Grid">${icon('layout-grid', 16)}</button>
              <button type="button" role="radio" aria-checked="${st.view === 'table'}" data-viewmode="table" aria-label="Table">${icon('list', 16)}</button>
            </div>
          </div>
        </div>
        ${!list.length ? `<p class="b-empty b-empty--page">${EMPTY[st.filter]}</p>` : st.view === 'grid'
          ? `<div class="b-grid b-house" style="--pw:${SIZES[st.size]}px">${list.map((w) => VB.tile(w)).join('')}</div>`
          : tableHTML(type, list)}
      </div>`;
    }
    function tableHTML(type, list) {
      const cols = type === 'movie' ? ['Work', 'Status', 'Targets', 'Profile', 'Size', 'Added', ''] : ['Work', 'Network', 'Episodes', 'Next', 'Profile', 'Size', ''];
      const sortable = { Work: 'title', Added: 'added' };
      const st = VB.local.lib[type];
      const row = (w) => {
        const s = VB.status(w);
        const prof = D.profiles.find((p) => p.id === w.profile).name;
        const lead = `<a class="b-trow__w" href="${VB.href('work/' + w.id)}" data-open="${w.id}"><span class="b-trow__art">${VB.art(w)}</span><span><span class="b-trow__t">${esc(w.title)}</span><span class="b-dim">${w.year}</span></span></a>`;
        const acts = `<span class="b-trow__acts"><button class="b-iconbtn b-iconbtn--sm" type="button" data-cmd="search" data-work="${w.id}" aria-label="Search for ${esc(w.title)}">${icon('search', 16)}</button><button class="b-iconbtn b-iconbtn--sm" type="button" data-cmd="delete" data-work="${w.id}" aria-label="Delete ${esc(w.title)}">${icon('trash-2', 16)}</button></span>`;
        if (type === 'movie') {
          const cov = VB.coverage(w).filter((x) => x.c !== 'unwanted').map((x) => `<span class="b-tchip" data-c="${x.c}">${esc(x.t.short)}</span>`).join('');
          const size = VB.files(w).reduce((a, f) => a + f.size, 0);
          const status = s.j ? `<span class="b-trow__live" data-lv="wline|${w.id}">${esc(VB.line(s.j))}</span>` : `<span data-s="${s.s}">${VB.statusLabel[s.s]}</span>`;
          return `<div class="b-trow" role="row">${lead}<span role="cell">${status}</span><span role="cell" class="b-tchips">${cov}</span><span role="cell">${prof}</span><span role="cell" class="b-num">${size ? F.bytes(size) : ''}</span><span role="cell" class="b-dim">${esc(F.ago(w.added))}</span>${acts}</div>`;
        }
        const live = s.j ? `<span class="b-trow__live" data-lv="wline|${w.id}|${VB.epKey(s.j.episode)}">${esc(VB.line(s.j))}</span>` : `${w.counts.available} of ${w.counts.aired}${w.counts.missing ? ` · <span>${w.counts.missing} missing</span>` : ''}`;
        return `<div class="b-trow b-trow--series" role="row">${lead}<span role="cell">${esc(w.network)}</span><span role="cell">${live}</span><span role="cell" class="b-dim">${w.nextAir ? esc(VB.cap(F.until(w.nextAir))) : w.continuing ? 'Not scheduled' : 'Ended'}</span><span role="cell">${prof}</span><span role="cell" class="b-num">${F.bytes(w.size)}</span>${acts}</div>`;
      };
      return `<div class="b-table b-table--${type}" role="table">
        <div class="b-trow b-trow--h${type === 'series' ? ' b-trow--series' : ''}" role="row">${cols.map((c) => sortable[c] ? `<button type="button" role="columnheader" class="b-th" data-sort-col="${sortable[c]}" aria-sort="${st.sort === sortable[c] ? (c === 'Work' ? 'ascending' : 'descending') : 'none'}">${c}${st.sort === sortable[c] ? icon('chevron-down', 14) : ''}</button>` : `<span role="columnheader" class="b-th">${c}</span>`).join('')}</div>
        ${list.map(row).join('')}
      </div>`;
    }
    function afterLibrary(el) {
      const grid = el.querySelector('.b-house');
      if (!grid) return;
      const focusOn = (tile) => {
        clearTimeout(hoverT);
        hoverT = setTimeout(() => spill.set(tile ? D.W[tile.dataset.open] : null), tile ? 90 : 420);
      };
      grid.addEventListener('pointerover', (e) => { const t = e.target.closest('.b-tile'); if (t) focusOn(t); });
      grid.addEventListener('pointerleave', () => focusOn(null));
      grid.addEventListener('focusin', (e) => { const t = e.target.closest('.b-tile'); if (t) focusOn(t); });
      grid.addEventListener('focusout', () => focusOn(null));
    }

    // Work detail: house lights down, the Work's own light up.
    function workHTML(r) {
      const w = D.W[r.id];
      const st = VB.status(w);
      const prof = D.profiles.find((p) => p.id === w.profile);
      const mon = VB.local.monitored[w.id] ?? true;
      const meta = w.type === 'movie'
        ? [String(w.year), F.runtime(w.runtime), w.cert, w.genres.join(', '), w.rating ? `Rated ${w.rating}` : '']
        : [w.endYear ? `${w.year} to ${w.endYear}` : `Since ${w.year}`, w.network, F.plural(w.seasons.length, 'season'), w.cert, w.genres.join(', ')];
      const from = stack.length ? parse(stack[stack.length - 1].view) : { name: navOf(route) };
      const backTo = from.name === 'work' ? D.W[from.id].title : LABEL[from.name] || 'Back';
      return `<article class="b-work" data-work="${w.id}" style="${VB.tint(w)}">
        <div class="b-work__stage"><img class="b-work__bd" src="${ART.backdrop(w, { w: 1680, h: 720 })}" alt=""><div class="b-work__shade"></div></div>
        <button class="b-back" type="button" data-back>${icon('chevron-left', 18)}<span>${esc(backTo)}</span></button>
        <div class="b-work__head">
          <div class="b-work__poster">${w.type === 'movie' ? VB.art(w, { vt: 'b-poster' }) : VB.poster(w, { vt: 'b-poster' })}</div>
          <div class="b-work__id">
            ${VB.logo(w, { maxW: 680, max: 92, min: 40 })}
            <p class="b-meta">${meta.filter(Boolean).map((m) => `<span>${esc(m)}</span>`).join('')}</p>
            <p class="b-work__ov">${esc(w.overview)}</p>
            <div class="b-work__act">
              <button class="b-switch" type="button" role="switch" aria-checked="${mon}" data-cmd="monitor" data-work="${w.id}"><span class="b-switch__t" aria-hidden="true"></span>Monitored</button>
              <button class="b-btn b-btn--primary" type="button" data-cmd="search" data-work="${w.id}">${icon('search', 16)}<span>${w.type === 'series' ? 'Search missing' : 'Search'}</span></button>
              <button class="b-btn" type="button" data-cmd="manual" aria-expanded="${r.sub === 'manual'}">${icon('list', 16)}Manual search</button>
              <button class="b-iconbtn" type="button" data-cmd="more" data-work="${w.id}" aria-label="More actions for ${esc(w.title)}" aria-expanded="false">${icon('ellipsis', 18)}</button>
            </div>
          </div>
          <aside class="b-work__state" data-state-panel>${statePanel(w)}</aside>
        </div>
        <section class="b-work__manual" ${r.sub === 'manual' ? '' : 'hidden'} aria-label="Manual search">${VB.releases(w)}</section>
        <div class="b-work__body">
          <div class="b-work__main">
            ${w.type === 'series' ? seasonsHTML(w) : ''}
            <section class="b-sec"><header class="b-sec__h"><h2 class="b-h2">Targets and Editions</h2><p class="b-sec__aside">${esc(prof.name)} Profile · ${F.plural(w.editions.length, 'Edition')}</p></header>${VB.covMatrix(w)}</section>
            ${w.type === 'movie' ? `<section class="b-sec"><header class="b-sec__h"><h2 class="b-h2">Files</h2></header>${VB.fileList(w)}</section>` : ''}
          </div>
          <aside class="b-work__side">
            <section><h2 class="b-h3">Details</h2>${detailsDL(w)}</section>
            <section><h2 class="b-h3">History</h2>${historyList(w)}</section>
          </aside>
        </div>
      </article>`;
    }
    function statePanel(w) {
      if (w.type === 'series') {
        const j = VB.acq(w.id);
        const next = D.calendar.find((c) => c.work === w.id);
        return `<p class="b-state__k">${w.continuing ? 'Continuing' : 'Ended'}</p>
          <p class="b-state__v">${w.counts.available} of ${w.counts.aired} aired episodes${w.counts.missing ? `, ${w.counts.missing} missing` : ''}</p>
          ${j ? `<p class="b-state__live"><span>${F.ep(j.episode)} </span><span data-lv="wline|${w.id}|${VB.epKey(j.episode)}">${esc(VB.line(j))}</span></p>` : ''}
          ${next ? `<p class="b-state__v b-dim">Next: ${F.ep(next.episode)} ${esc(next.title)}, ${esc(F.until(next.at))}</p>` : ''}`;
      }
      const { s, j } = VB.status(w);
      if (j) {
        if (j.state === 'failed') return `<p class="b-state__k b-exit">Import failed</p><p class="b-state__v">${esc(j.message)}.</p><button class="b-btn b-btn--sm" type="button" data-cmd="blocklist" data-job="${j.id}">Blocklist and search</button>`;
        if (j.state === 'stalled') return `<p class="b-state__k">Stalled on ${esc(j.node)}</p><p class="b-state__v">${esc(j.message)}. ${Math.floor(j.progress * 100)}% downloaded.</p><button class="b-btn b-btn--sm" type="button" data-cmd="manual">Find another release</button>`;
        const steps = [['download', 'Download', `on ${j.kind === 'download' ? j.node : 'ams-1'}`], ['transfer', 'Transfer', 'to vault'], ['import', 'Import', 'on vault']];
        const at = steps.findIndex((x) => x[0] === j.kind);
        return `<ol class="b-pipe">${steps.map(([k, l, where], i) => `<li data-s="${i < at ? 'done' : i === at ? 'cur' : 'todo'}"><span class="b-pipe__k">${l} <span class="b-dim">${where}</span></span><span class="b-pipe__v">${i < at ? 'Done' : i === at ? `<span data-lv="jpct|${j.id}">${Math.floor(j.progress * 100)}%</span> · <span data-lv="jrate|${j.id}">${esc(VB.LV.jrate(j.id))}</span>` : ''}</span></li>`).join('')}</ol>
          <p class="b-state__ready"><span data-lv="wmember|${w.id}">${esc(VB.memberLine(j))}</span></p>`;
      }
      const playing = D.playing.find((p) => p.work === w.id);
      const lines = {
        available: `<p class="b-state__k">In the library</p><p class="b-state__v">${esc(VB.coverage(w).filter((x) => x.c === 'met').map((x) => x.t.name).join(', '))}</p>`,
        upgradable: `<p class="b-state__k">Upgradable</p><p class="b-state__v">1080p is below its Blu-ray cutoff. SlipStream keeps looking.</p>`,
        missing: `<p class="b-state__k">Missing</p><p class="b-state__v">Monitored. Searching every 15 min on 4 indexers.</p>`,
        unreleased: `<p class="b-state__k">Unreleased</p><p class="b-state__v">Out ${w.release ? esc(F.day(w.release)) : w.year}. Searching starts on release.</p>`,
      }[s] || '';
      return lines + (playing ? `<p class="b-state__v b-dim">${esc(VB.first(playing.who))} is watching on ${esc(playing.device)}</p>` : '');
    }
    function detailsDL(w) {
      const rows = w.type === 'movie'
        ? [['Director', w.director], ['Cast', w.cast.slice(0, 3).join(', ')], ['Studio', w.studio], ['Added', F.ago(w.added)], ['Requested by', w.addedBy ? VB.name(w.addedBy) : ''], ['Path', `/mnt/media/Movies/${w.title} (${w.year})`]]
        : [['Creator', w.creator], ['Cast', w.cast.slice(0, 3).join(', ')], ['Network', w.network], ['Added', F.ago(w.added)], ['Path', `/mnt/media/Series/${w.title}`]];
      return `<dl class="b-dl">${rows.filter((x) => x[1]).map(([k, v]) => `<div><dt>${k}</dt><dd${k === 'Path' ? ' class="b-mono"' : ''}>${esc(v)}</dd></div>`).join('')}</dl>`;
    }
    function historyList(w) {
      const h = D.history.filter((x) => x.work === w.id);
      if (!h.length) return '<p class="b-empty b-empty--quiet">No history yet. Grabs, imports and upgrades for this Work are listed here.</p>';
      return `<ol class="b-hist">${h.map((x) => `<li><span>${esc(x.text)}</span><span class="b-dim">${esc(F.ago(x.at))} · ${esc(x.node)}</span></li>`).join('')}</ol>`;
    }
    function seasonsHTML(w) {
      const cur = VB.local.season?.[w.id] || w.seasons.length;
      return `<section class="b-sec" aria-labelledby="b-seasons-h"><header class="b-sec__h"><h2 class="b-h2" id="b-seasons-h">Seasons</h2>
        <div class="b-seg" role="tablist" aria-label="Season">${w.seasons.map((s) => `<button type="button" role="tab" aria-selected="${s.n === cur}" data-season="${s.n}">Season ${s.n}</button>`).join('')}</div></header>
        <div data-eps>${VB.episodes(w, cur)}</div></section>`;
    }
    function afterWork() {}
    const refreshState = (j) => {
      if (route.name !== 'work' || j.work !== route.id) return;
      const p = viewEl.querySelector('[data-state-panel]');
      if (p) { p.innerHTML = statePanel(D.W[route.id]); live.scan(p); }
    };
    ctx.on('job:new', refreshState);
    ctx.on('job:done', refreshState);

    // Activity: every job on every Node, live.
    const ACT = { filter: 'all' };
    function activityHTML() {
      const t = SS.sim.totals();
      const needs = D.jobs.filter((j) => (j.state === 'failed' || j.state === 'stalled') && !VB.local.resolved.has(j.id));
      const groups = [
        ['needs', 'Needs you', needs],
        ['running', 'Running', D.jobs.filter((j) => j.state === 'running')],
        ['queued', 'Queued', D.jobs.filter((j) => j.state === 'queued')],
      ].filter(([k]) => ACT.filter === 'all' || ACT.filter === k);
      const fl = [['all', 'All'], ['running', 'Running'], ['queued', 'Queued'], ['needs', 'Needs you']];
      return `<div class="b-page">
        <header class="b-page__h"><h1 class="b-h1">Activity</h1><p class="b-page__sum"><span data-lv="active">${t.active}</span> running, ${t.queued} queued · <span data-lv="down">${F.rate(t.down)}</span> down across ${D.nodes.filter((n) => n.state !== 'offline').length} Nodes</p></header>
        <div class="b-toolbar"><div class="b-seg" role="radiogroup" aria-label="Show">${fl.map(([k, l]) => `<button type="button" role="radio" aria-checked="${ACT.filter === k}" data-actfilter="${k}">${l}</button>`).join('')}</div></div>
        <div class="b-jobs" role="table" aria-label="Jobs">
          <div class="b-job b-job--h" role="row"><span></span><span role="columnheader">Work</span><span role="columnheader">Node</span><span role="columnheader" class="b-job__n">Progress</span><span role="columnheader" class="b-job__n">Rate</span><span role="columnheader" class="b-job__n">Left</span><span></span></div>
          ${groups.map(([k, l, js]) => `<div class="b-jobs__g" role="rowgroup"><h2 class="b-h3 b-jobs__gh">${l} <span class="b-dim">${js.length}</span></h2>${js.length ? js.map(VB.jobRow).join('') : `<p class="b-empty b-empty--quiet">${k === 'needs' ? 'Nothing needs you. Failed and stalled jobs land here with a fix.' : k === 'queued' ? 'Nothing is waiting. Jobs queue here when a Node is busy.' : 'Nothing is running.'}</p>`}</div>`).join('')}
        </div>
        <div class="b-act2">
          <section class="b-sec"><header class="b-sec__h"><h2 class="b-h2">Recently</h2></header><ol class="b-hist b-hist--wide">${D.history.slice(0, 8).map((x) => `<li><span class="b-hist__w">${esc(D.W[x.work].title)}${x.episode ? ' ' + F.ep(x.episode) : ''}</span><span>${esc(x.text)}</span><span class="b-dim">${esc(x.node)} · ${esc(F.ago(x.at))}</span></li>`).join('')}</ol></section>
          <section class="b-sec"><header class="b-sec__h"><h2 class="b-h2">Scheduled tasks</h2></header>${tasksList()}</section>
        </div>
      </div>`;
    }
    const tasksList = () => `<ul class="b-tasks" role="list">${VB.tasks.map((t) => `<li><span class="b-tasks__n">${t.state === 'running' ? '<i class="b-spin" aria-hidden="true"></i>' : icon('clock', 14)}${esc(t.name)}</span><span class="b-dim">${esc(t.note)}</span></li>`).join('')}</ul>`;

    function nodesHTML() {
      const on = D.nodes.filter((n) => n.state === 'online').length;
      return `<div class="b-page">
        <header class="b-page__h"><h1 class="b-h1">Nodes</h1><p class="b-page__sum">${F.plural(D.nodes.length, 'Node')} · ${on} online, 1 behind, 1 offline · atlas is the Controller</p>
          <div class="b-page__act"><button class="b-btn" type="button" data-cmd="joincode">${icon('plus', 16)}Pair a Node</button></div></header>
        <div class="b-nodes">${D.nodes.map(VB.nodeRow).join('')}</div>
      </div>`;
    }

    const REQ = { tab: 'pending' };
    function requestsHTML() {
      const all = VB.reqs();
      const tabs = [['pending', 'Pending', (s) => s === 'pending'], ['approved', 'Approved', (s) => s === 'approved' || s === 'downloading'], ['available', 'Available', (s) => s === 'available'], ['denied', 'Denied', (s) => s === 'denied']];
      const cur = tabs.find((t) => t[0] === REQ.tab);
      const list = all.filter((r) => cur[2](VB.reqState(r))).sort((a, b) => b.at - a.at);
      const quota = (a, k) => `${a.quota[k][0]} of ${a.quota[k][1]}`;
      return `<div class="b-page">
        <header class="b-page__h"><h1 class="b-h1">Requests</h1><p class="b-page__sum">Members ask, you decide. Approved requests are searched right away.</p></header>
        <div class="b-reqgrid">
          <div>
            <div class="b-toolbar"><div class="b-seg" role="tablist" aria-label="Requests">${tabs.map(([k, l, f]) => { const n = all.filter((r) => f(VB.reqState(r))).length; return `<button type="button" role="tab" aria-selected="${REQ.tab === k}" data-reqtab="${k}">${l}<span class="b-seg__n${k === 'pending' && n ? ' b-amber' : ''}">${n}</span></button>`; }).join('')}</div></div>
            <div class="b-reqs" data-reqs>${list.length ? list.map(VB.reqRow).join('') : `<p class="b-empty b-empty--page">${REQ.tab === 'pending' ? 'No requests waiting. When a Member asks for a Work it waits here for you.' : 'Nothing here yet.'}</p>`}</div>
          </div>
          <aside class="b-quota" aria-labelledby="b-quota-h"><h2 class="b-h3" id="b-quota-h">This week</h2>
            <ul role="list">${D.accounts.filter((a) => !a.admin).map((a) => `<li>${VB.av(a.id)}<span class="b-quota__n">${esc(a.name)}</span><span class="b-quota__v">${quota(a, 'movie')} movies · ${quota(a, 'series')} series</span></li>`).join('')}</ul>
            <p class="b-dim">Quotas reset on Monday. Change them under Accounts.</p></aside>
        </div>
      </div>`;
    }

    // ---------- Palette ----------
    const PAL = { el: null, sel: 0, items: [] };
    const COMMANDS = [
      { id: 'missing-eps', label: 'Search for missing episodes', hint: '3 episodes in 3 series', run: () => VB.toast(root, 'Searching 4 indexers for 3 missing episodes') },
      { id: 'missing-movies', label: 'Search for missing movies', hint: '4 movies', run: () => VB.toast(root, 'Searching 4 indexers for 4 missing movies') },
      { id: 'pause-ams', label: 'Pause ams-1', hint: '3 downloads', run: () => VB.toast(root, 'ams-1 paused. 3 downloads hold their place') },
      { id: 'approve-all', label: 'Approve every pending request', hint: 'From Maya, Theo and Rosa', run: () => { for (const r of VB.reqs()) if (VB.reqState(r) === 'pending') VB.local.req[r.id] = 'approved'; VB.toast(root, 'Approved 3 requests. Searching now'); render(); } },
      { id: 'refresh', label: 'Refresh all metadata', hint: '58 Works', run: () => VB.toast(root, 'Refreshing metadata for 58 Works') },
      { id: 'restart', label: 'Restart SlipStream', hint: 'All Nodes reconnect in about 20 s', run: () => VB.toast(root, 'Restart scheduled after the current imports finish') },
    ];
    function palItems(q) {
      q = q.trim().toLowerCase();
      const works = D.works.filter((w) => !q || w.title.toLowerCase().includes(q)).slice(0, q ? 6 : 0).map((w) => ({ kind: 'work', w, label: w.title, hint: `${w.type === 'movie' ? 'Movie' : 'Series'} · ${w.year} · ${VB.statusLabel[VB.status(w).s]}`, run: (kbd) => go(`work/${w.id}`, { kbd }) }));
      const views = NAV.filter(([, l]) => !q || l.toLowerCase().includes(q) || 'go to'.includes(q)).map(([k, l]) => ({ kind: 'view', label: `Go to ${l}`, hint: '', run: (kbd) => go(k, { kbd }) }));
      const cmds = COMMANDS.filter((c) => !q || c.label.toLowerCase().includes(q)).map((c) => ({ kind: 'cmd', label: c.label, hint: c.hint, run: c.run }));
      const recent = q ? [] : [D.W[now.get().work], ...VB.developing().slice(0, 2).map((x) => x.w)].map((w) => ({ kind: 'work', w, label: w.title, hint: `${w.type === 'movie' ? 'Movie' : 'Series'} · ${VB.statusLabel[VB.status(w).s]}`, run: (kbd) => go(`work/${w.id}`, { kbd }) }));
      return [['Works', [...recent, ...works]], ['Go to', views], ['Commands', cmds]].filter((g) => g[1].length);
    }
    function palRender(q) {
      const groups = palItems(q);
      PAL.items = groups.flatMap((g) => g[1]);
      PAL.sel = Math.min(PAL.sel, Math.max(0, PAL.items.length - 1));
      let i = 0;
      const list = PAL.el.querySelector('.b-pal__list');
      list.innerHTML = groups.length ? groups.map(([g, xs]) => `<div class="b-pal__g" role="group" aria-label="${g}"><p class="b-pal__gh">${g}</p>${xs.map((x) => { const k = i++; return `<div class="b-pal__i" role="option" id="b-pal-${k}" data-i="${k}" aria-selected="${k === PAL.sel}">${x.kind === 'work' ? `<img src="${ART.poster(x.w)}" alt="">` : x.kind === 'view' ? `<span class="b-pal__ic">${icon('arrow-right', 16)}</span>` : `<span class="b-pal__ic">${icon('command', 16)}</span>`}<span class="b-pal__l">${esc(x.label)}</span><span class="b-pal__h">${esc(x.hint)}</span></div>`; }).join('')}</div>`).join('') : `<p class="b-empty b-empty--quiet">Nothing matches “${esc(q)}”. Try a Work's title, a view such as Activity, or a command such as Pause.</p>`;
      PAL.el.querySelector('input').setAttribute('aria-activedescendant', PAL.items.length ? `b-pal-${PAL.sel}` : '');
    }
    function palOpen() {
      if (PAL.el) return;
      pop.close();
      const host = root.querySelector('.b-layer');
      host.insertAdjacentHTML('beforeend', `<div class="b-pal" role="dialog" aria-modal="true" aria-label="Search and commands">
        <div class="b-pal__scrim" data-palclose></div>
        <div class="b-pal__panel">
          <label class="b-pal__in">${icon('search', 18)}<span class="b-sr">Search</span><input type="text" role="combobox" aria-expanded="true" aria-controls="b-pal-list" placeholder="Search Works, jump to a view, run a command" autocomplete="off" spellcheck="false"><kbd>esc</kbd></label>
          <div class="b-pal__list" id="b-pal-list" role="listbox"></div>
          <div class="b-pal__foot"><span><kbd>${icon('arrow-up', 12)}</kbd><kbd>${icon('arrow-down', 12)}</kbd>Move</span><span><kbd>${icon('corner-down-left', 12)}</kbd>Open</span></div>
        </div></div>`);
      PAL.el = host.lastElementChild;
      PAL.sel = 0;
      palRender('');
      const input = PAL.el.querySelector('input');
      input.focus();
      input.addEventListener('input', () => { PAL.sel = 0; palRender(input.value); });
      input.addEventListener('keydown', (e) => {
        if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
          e.preventDefault();
          PAL.sel = (PAL.sel + (e.key === 'ArrowDown' ? 1 : -1) + PAL.items.length) % PAL.items.length;
          for (const el of PAL.el.querySelectorAll('[data-i]')) el.setAttribute('aria-selected', String(+el.dataset.i === PAL.sel));
          PAL.el.querySelector(`[data-i="${PAL.sel}"]`)?.scrollIntoView({ block: 'nearest' });
          input.setAttribute('aria-activedescendant', `b-pal-${PAL.sel}`);
        } else if (e.key === 'Enter') { e.preventDefault(); const x = PAL.items[PAL.sel]; palClose(); x?.run(true); }
        else if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); palClose(); }
      });
      PAL.el.addEventListener('pointermove', (e) => { const it = e.target.closest('[data-i]'); if (it && +it.dataset.i !== PAL.sel) { PAL.sel = +it.dataset.i; for (const el of PAL.el.querySelectorAll('[data-i]')) el.setAttribute('aria-selected', String(+el.dataset.i === PAL.sel)); } });
      PAL.el.addEventListener('click', (e) => { if (e.target.closest('[data-palclose]')) return palClose(); const it = e.target.closest('[data-i]'); if (it) { const x = PAL.items[+it.dataset.i]; palClose(); x.run(false); } });
    }
    function palClose() { if (!PAL.el) return; PAL.el.remove(); PAL.el = null; }
    ctx.cleanup(palClose);

    // ---------- Popovers ----------
    function bellHTML() {
      const al = D.alerts;
      return `<header class="b-pop__h"><h2 class="b-h3">Notifications</h2><button class="b-btn b-btn--quiet b-btn--sm" type="button" data-cmd="markread">Mark all read</button></header>
        <ul class="b-notes" role="list">${al.map((a) => `<li data-level="${a.level}"><i class="b-dot b-dot--${a.level === 'error' ? 'exit' : a.level === 'warning' && a.work === 'velvet-hour' ? 'amber' : 'ink'}" aria-hidden="true"></i><div><span class="b-notes__t">${esc(a.title)}</span><span class="b-notes__d">${esc(a.detail)}</span><span class="b-notes__a">${esc(F.ago(a.at))}</span></div></li>`).join('')}</ul>`;
    }
    function moreMenu(w) {
      return `<div class="b-menu" role="menu">${[['refresh-cw', 'Refresh metadata', 'refresh'], ['sliders-horizontal', 'Change Profile', 'profile'], ['folder', 'Show folder on vault', 'folder'], ['ban', 'Blocklist a release', 'blocklist-menu']].map(([ic, l, c]) => `<button type="button" role="menuitem" data-cmd="${c}" data-work="${w.id}">${VB.icon(ic, 16)}${l}</button>`).join('')}<hr><button type="button" role="menuitem" data-cmd="delete" data-work="${w.id}">${icon('trash-2', 16)}Delete ${esc(w.title)}</button></div>`;
    }

    // ---------- Commands ----------
    function command(btn, kbd) {
      const c = btn.dataset.cmd;
      const w = btn.dataset.work ? D.W[btn.dataset.work] : route.name === 'work' ? D.W[route.id] : null;
      const job = btn.dataset.job ? VB.job(btn.dataset.job) : null;
      switch (c) {
        case 'play': VB.toast(root, `Opening ${w.title} on Plex`); break;
        case 'monitor': {
          const on = btn.getAttribute('aria-checked') !== 'true';
          btn.setAttribute('aria-checked', String(on));
          VB.local.monitored[w.id] = on;
          VB.toast(root, on ? `${w.title} is monitored. SlipStream searches for it` : `${w.title} is unmonitored. Files stay, searches stop`);
          break;
        }
        case 'search': {
          if (btn.dataset.busy) return;
          btn.dataset.busy = '1';
          btn.setAttribute('aria-busy', 'true');
          btn.disabled = true;
          setTimeout(() => {
            btn.disabled = false; btn.removeAttribute('aria-busy'); delete btn.dataset.busy;
            VB.toast(root, w.type === 'series' ? `Searched 4 indexers. Nothing new for ${w.title}` : `Searched 4 indexers. 5 releases, none better than what ${w.title} has`);
          }, 1400);
          break;
        }
        case 'manual': {
          const panel = viewEl.querySelector('.b-work__manual');
          if (!panel) break;
          panel.hidden = !panel.hidden;
          viewEl.querySelector('[data-cmd="manual"]').setAttribute('aria-expanded', String(!panel.hidden));
          ctx.setView(`work/${route.id}${panel.hidden ? '' : '/manual'}`);
          if (!panel.hidden) panel.scrollIntoView({ block: 'nearest', behavior: kbd || VB.reduced() ? 'auto' : 'smooth' });
          break;
        }
        case 'grab': btn.textContent = 'Grabbed'; btn.disabled = true; VB.toast(root, 'Sent to ams-1. It appears in Developing in a moment'); break;
        case 'more': pop.show(btn, moreMenu(w), { width: 248, cls: 'b-pop--menu' }); break;
        case 'tasks': break;
        case 'blocklist': VB.local.resolved.add(job.id); VB.toast(root, `Blocklisted the sample. Searching 4 indexers for ${job.label}`); render(); break;
        case 'another': go(`work/${job.work}/manual`, { kbd }); break;
        case 'search-ep': VB.toast(root, `Searching for ${w.title} ${F.ep(btn.dataset.ep.split('x').map(Number))}`); break;
        case 'jobmenu': pop.show(btn, `<div class="b-menu" role="menu"><button type="button" role="menuitem" data-cmd="noop">${icon('pause', 16)}Pause</button><button type="button" role="menuitem" data-cmd="noop">${icon('arrow-up', 16)}Move to the front</button><hr><button type="button" role="menuitem" data-cmd="noop">${icon('x', 16)}Cancel and blocklist</button></div>`, { width: 220, cls: 'b-pop--menu' }); break;
        case 'delete': pop.close(); VB.toast(root, `Delete is disabled in the prototype`); break;
        case 'joincode': VB.toast(root, 'Join Code 4K7-Q2M, valid for 10 min'); break;
        case 'markread': bar.querySelector('.b-iconbtn__dot').hidden = true; pop.close(); break;
        default: pop.close(); VB.toast(root, 'Done'); break;
      }
    }

    // ---------- Events ----------
    ctx.listen(root, 'click', (e) => {
      if (e.defaultPrevented || e.button !== 0) return;
      const kbd = e.detail === 0;
      const a = e.target.closest('a[href]');
      if (a && (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey)) return;
      const t = e.target.closest('[data-go],[data-open],[data-back],[data-cmd],[data-act],[data-aisle-act],[data-filter],[data-viewmode],[data-size],[data-sort-col],[data-season],[data-actfilter],[data-reqtab],[data-req-approve],[data-req-deny],[data-req-cancel],[data-req-confirm]');
      if (!t || !root.contains(t)) return;
      if (a) e.preventDefault();
      const d = t.dataset;
      if (d.go) return go(d.go, { kbd });
      if (d.open) return go(`work/${d.open}`, { kbd, el: t.querySelector('.b-tile__art > *, .b-job__art > *, .b-req__art > *') });
      if ('back' in d) return back(kbd);
      if (d.cmd) return command(t, kbd);
      if (d.act === 'palette') return palOpen();
      if (d.act === 'bell') return pop.show(t, bellHTML(), { width: 380 });
      if (d.act === 'tasks') return pop.show(t, `<header class="b-pop__h"><h2 class="b-h3">Scheduled tasks</h2></header>${tasksList()}`, { width: 360 });
      if (d.act === 'account') return pop.show(t, `<div class="b-acctpop"><p class="b-acctpop__n">${esc(VB.name(D.me))}</p><p class="b-dim">${esc(D.A[D.me].email)} · Admin</p></div><div class="b-menu" role="menu"><button type="button" role="menuitem" data-cmd="noop">${icon('user', 16)}Account and passkeys</button><button type="button" role="menuitem" data-cmd="noop">${icon('settings', 16)}Settings</button><button type="button" role="menuitem" data-cmd="noop">${icon('rotate-ccw', 16)}Restart SlipStream</button><hr><button type="button" role="menuitem" data-cmd="noop">${icon('log-out', 16)}Log out</button></div>`, { width: 260, cls: 'b-pop--menu' });
      if (d.aisleAct) {
        const item = VB.aisle().find((x) => x.id === d.aisleAct);
        if (!item) return;
        if (item.cmd === 'blocklist') { VB.local.resolved.add(item.job); VB.toast(root, 'Blocklisted the sample. Searching 4 indexers again'); const box = viewEl.querySelector('[data-aisle]'); if (box) { box.innerHTML = VB.aisleList(VB.aisle()); } refreshDevRow(viewEl.querySelector('[data-devrow]') || document.createElement('div')); return; }
        return go(item.job ? `work/${VB.job(item.job).work}/manual` : item.go, { kbd });
      }
      const lib = route.name === 'library' ? VB.local.lib.movie : VB.local.lib.series;
      if (d.filter) { lib.filter = d.filter; return render(); }
      if (d.viewmode) { lib.view = d.viewmode; return render(); }
      if (d.size) { lib.size = d.size; const g = viewEl.querySelector('.b-grid'); if (g) g.style.setProperty('--pw', SIZES[d.size] + 'px'); for (const b of viewEl.querySelectorAll('[data-size]')) b.setAttribute('aria-checked', String(b === t)); return; }
      if (d.sortCol) { lib.sort = d.sortCol; return render(); }
      if (d.season) {
        const w = D.W[route.id];
        (VB.local.season ||= {})[w.id] = +d.season;
        for (const b of viewEl.querySelectorAll('[data-season]')) b.setAttribute('aria-selected', String(b === t));
        const box = viewEl.querySelector('[data-eps]');
        box.innerHTML = VB.episodes(w, +d.season);
        live.scan(box); VB.fillStills(box);
        return;
      }
      if (d.actfilter) { ACT.filter = d.actfilter; return render(); }
      if (d.reqtab) { REQ.tab = d.reqtab; return render(); }
      const row = t.closest('.b-req');
      if (d.reqApprove) { VB.local.req[d.reqApprove] = 'approved'; settleReq(row, 'Approved. Searching now'); return; }
      if (d.reqDeny) { row.querySelector('.b-req__deny').hidden = false; row.querySelector('.b-req__act').hidden = true; row.querySelector('.b-req__deny input').focus(); return; }
      if (d.reqCancel) { row.querySelector('.b-req__deny').hidden = true; row.querySelector('.b-req__act').hidden = false; return; }
      if (d.reqConfirm) { VB.local.req[d.reqConfirm] = 'denied'; settleReq(row, 'Denied. They will see your reason'); }
    });
    function settleReq(row, msg) {
      row.dataset.state = 'settling';
      VB.toast(root, msg);
      updateBar();
      const box = row.parentElement;
      const h = row.offsetHeight;
      const done = () => { row.remove(); if (!box.children.length) box.innerHTML = '<p class="b-empty b-empty--page">No requests waiting. When a Member asks for a Work it waits here for you.</p>'; render(); };
      if (VB.reduced()) return done();
      row.animate([{ opacity: 1, height: h + 'px' }, { opacity: 0, height: '0px', paddingTop: '0px', paddingBottom: '0px', marginTop: '0px' }], { duration: 320, delay: 420, easing: 'cubic-bezier(0.23, 1, 0.32, 1)', fill: 'forwards' }).finished.then(done);
    }
    ctx.listen(root, 'change', (e) => {
      if (e.target.matches('[data-sort]')) { (route.name === 'library' ? VB.local.lib.movie : VB.local.lib.series).sort = e.target.value; render(); }
    });
    ctx.listen(main, 'scroll', () => {
      const y = main.scrollTop;
      bar.classList.toggle('is-solid', y > 8);
      spill.dim(1 - Math.min(1, y / 1100) * 0.6);
    }, { passive: true });
    ctx.listen(window, 'keydown', (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); PAL.el ? palClose() : palOpen(); return; }
      if (e.key === 'Escape' && !e.defaultPrevented && !PAL.el && route.name === 'work' && !e.target.closest('input')) back(true);
    });
    ctx.on('second', () => updateBar());
    ctx.on('job:new', () => { if (route.name === 'activity') rerenderKeep(); });
    ctx.on('job:done', () => { if (route.name === 'activity') rerenderKeep(); });
    ctx.on('request:update', () => { updateBar(); if (route.name === 'requests') rerenderKeep(); });
    function rerenderKeep() { const y = main.scrollTop; render(); main.scrollTop = y; }

    render();
    requestAnimationFrame(() => bar.querySelector('.b-nav__ind').classList.add('is-ready'));
    return { unmount() { clearTimeout(hoverT); clearTimeout(devTimer); } };
  };
})();
