(function () {
  const SS = window.SS;
  const d = SS.data;
  const fmt = SS.fmt;
  const VA = window.VA;
  const esc = fmt.esc;
  const icon = VA.icon;
  const ME = () => d.member;

  const mine = () => d.requests.filter((r) => r.by === ME());
  const requestFor = (w) => d.requests.find((r) => r.work === w.id && r.state !== 'denied' && (r.by === ME() || r.state !== 'available'));
  const seasonState = (w) => w.seasons.map((s) => {
    const aired = s.episodes.filter((e) => e.status !== 'unreleased').length;
    const have = s.episodes.filter((e) => e.status === 'available').length;
    return { n: s.n, year: s.year, total: s.episodes.length, aired, have, done: aired > 0 && have >= aired };
  });
  const inLibrary = (w) => (w.type === 'series' ? w.counts.available > 0 : w.status === 'available' || w.status === 'upgradable');
  const RANK = { downloading: 0, approved: 1, pending: 2, available: 3, denied: 4 };

  const journeyLabel = (r) => {
    const J = VA.journey(r);
    return J.step === 2 ? `On its way · ${J.detail.replace('Ready in about ', 'about ')}` : J.label;
  };
  const card = (r, by = false) => {
    const w = d.W[r.work];
    const J = VA.journey(r);
    return `<a class="a-preqc ${J.step === 3 ? 'is-ready' : ''}" href="#request/${r.id}" data-key="${r.id}">
      <span class="a-preqc__art">${VA.posterImg(w)}</span>
      <span class="a-preqc__body">
        <span class="a-preqc__t"><b>${esc(w.title)}</b><span>${by ? `Requested by ${esc(VA.firstName(r.by))}` : `${w.type === 'series' ? (r.seasons ? `Season ${r.seasons.join(', ')}` : 'Series') : w.year}`}</span></span>
        <span class="a-preqc__state" data-flip="state">${esc(J.denied ? 'Not added' : journeyLabel(r))}</span>
        ${J.denied ? `<span class="a-preqc__note">${esc(J.detail)}</span>` : ''}
      </span>
      ${J.denied ? '' : `<span class="a-preqc__strip" data-strip>${VA.journeyStrip(r)}</span>`}
    </a>`;
  };

  VA.mountPortal = (ctx) => {
    const root = ctx.root;
    root.classList.add('a-app', 'a-pt');
    const wide = ctx.shell === 'portal-wide';
    root.innerHTML = `<header class="a-pt__head">
        <a class="a-pt__brand" href="#home" aria-label="SlipStream home">${VA.mark(24)}<span>SlipStream</span></a>
        <nav class="a-pt__nav" aria-label="Portal"><a href="#home" data-nav="home">Home</a><a href="#library" data-nav="library">Library</a><a href="#requests" data-nav="requests">Your requests</a></nav>
        <label class="a-pt__field">${icon('search', { size: 16 })}<input type="search" placeholder="Search movies and series" aria-label="Search movies and series" data-hsearch enterkeyhint="search" autocomplete="off"></label>
        <div class="a-pt__icons">
          <a class="a-ibtn a-pt__phone" href="#search" aria-label="Search">${icon('search', { size: 20 })}</a>
          <a class="a-ibtn a-pt__phone" href="#library" aria-label="Library">${icon('library', { size: 20 })}</a>
          <button class="a-ibtn" data-bell aria-label="Notifications">${icon('bell', { size: 20 })}<span class="a-dot" data-bdot></span></button>
          <button class="a-ibtn" data-settings aria-label="Settings">${icon('settings', { size: 20 })}</button>
          <span class="a-pt__me">${VA.avatar('maya', 28)}</span>
        </div>
      </header>
      <main class="a-pt__main" data-view></main>
      <div class="a-pt__layer" data-layer></div>`;
    const viewEl = root.querySelector('[data-view]');
    const layer = root.querySelector('[data-layer]');
    let cur = null;
    const api = { root, ctx, wide, toast: (t) => VA.toast(root, t), go: (v) => go(v) };

    function go(v, { keepSearch = false } = {}) {
      v = v || 'home';
      const [name, arg] = v.split('/');
      const fn = VIEWS[name] ? name : 'home';
      cur?.destroy?.();
      layer.innerHTML = '';
      ctx.setView(v);
      viewEl.scrollTop = 0;
      viewEl.innerHTML = '';
      cur = VIEWS[fn](viewEl, arg, api) || {};
      root.querySelectorAll('[data-nav]').forEach((a) => (a.dataset.nav === fn ? a.setAttribute('aria-current', 'page') : a.removeAttribute('aria-current')));
      if (!keepSearch) root.querySelector('[data-hsearch]').value = fn === 'search' ? arg ? decodeURIComponent(arg) : '' : '';
      root.querySelector('[data-bdot]').hidden = !d.requests.some((r) => r.by === ME() && r.state === 'available' && !r.seen);
    }

    const hs = root.querySelector('[data-hsearch]');
    hs.addEventListener('input', () => {
      if (cur?.search) cur.search(hs.value);
      else { go('search', { keepSearch: true }); cur.search?.(hs.value); }
    });

    ctx.listen(root, 'click', (e) => {
      const t = e.target;
      if (t.closest('[data-bell]')) {
        if (layer.innerHTML) { layer.innerHTML = ''; return; }
        const items = mine().slice().sort((a, b) => RANK[a.state] - RANK[b.state]).map((r) => `<a class="a-note" href="#request/${r.id}"><span class="a-note__t"><b>${esc(d.W[r.work].title)}</b><span>${esc(VA.journey(r).label)}</span></span></a>`).join('');
        layer.innerHTML = `<div class="a-scrim" data-close></div><div class="a-pop a-pop--pt" role="dialog" aria-label="Notifications"><div class="a-pop__head"><b>Notifications</b></div>${items}</div>`;
        layer.querySelector('[data-close]').addEventListener('pointerdown', () => { layer.innerHTML = ''; });
        return;
      }
      if (t.closest('[data-settings]')) { api.toast('Your PIN, passkeys and email live in Settings.'); return; }
      if (cur?.click?.(e) === true) return;
      const a = t.closest('a[href^="#"]');
      if (a) { e.preventDefault(); go(a.getAttribute('href').slice(1)); }
    });
    ctx.on('second', () => cur?.live?.());
    for (const ev of ['job:new', 'job:done', 'job:update', 'request:update', 'work:update']) ctx.on(ev, (p) => cur?.refresh?.(ev, p));

    go(ctx.view || 'home');
    return { unmount() { cur?.destroy?.(); } };
  };

  const VIEWS = {};

  VIEWS.home = (el) => {
    const me = d.A[ME()];
    const h = SS.sim.clock.getHours();
    const greet = h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening';
    const fresh = d.history.filter((x) => (x.kind === 'imported' || x.kind === 'upgraded') && !x.episode).map((x) => d.W[x.work]);
    const newIn = [...new Set([...fresh, ...d.works.filter((w) => inLibrary(w) && w.type === 'movie').sort((a, b) => b.added - a.added)])].slice(0, 8);
    const soon = d.calendar.slice().sort((a, b) => a.at - b.at).slice(0, 5);
    el.innerHTML = `<div class="a-pt__page">
      <div class="a-pt__lede"><h1 class="a-pt__hello">${greet}, ${esc(me.name.split(' ')[0])}.</h1><p class="a-pt__sum" data-sum></p></div>
      <section class="a-pt__sec"><div class="a-section-head"><h2 class="a-h2">Your requests</h2><a class="a-link" href="#requests">All requests ${icon('chevron-right', { size: 14 })}</a></div><div class="a-pt__cards" data-mine></div></section>
      <section class="a-pt__sec" data-other-sec><div class="a-section-head"><h2 class="a-h2">Also on its way</h2></div><div class="a-pt__cards" data-other></div></section>
      <section class="a-pt__sec"><div class="a-section-head"><h2 class="a-h2">New in the library</h2><a class="a-link" href="#library">Library ${icon('chevron-right', { size: 14 })}</a></div><div class="a-pt__row">${newIn.map((w) => VA.tile(w, { quiet: true })).join('')}</div></section>
      <section class="a-pt__sec"><div class="a-section-head"><h2 class="a-h2">Coming soon</h2></div><ol class="a-pt__soon">${soon.map((c) => { const w = d.W[c.work]; return `<li><a href="#work/${w.id}">${VA.posterImg(w, 'a-poster--sm')}<span><b>${esc(w.title)}</b><span>${c.episode ? `${fmt.ep(c.episode)} · ${esc(c.title)}` : 'New release'}</span></span><span class="a-num">${esc(fmt.until(c.at))}</span></a></li>`; }).join('')}</ol></section>
    </div>`;
    const q = (s) => el.querySelector(s);
    const render = () => {
      const m = mine().filter((r) => r.state !== 'denied').sort((a, b) => RANK[a.state] - RANK[b.state]);
      VA.patch(q('[data-mine]'), m, (r) => r.id, (r) => card(r));
      VA.emptyState(q('[data-mine]'), !m.length, 'You have no requests yet.', 'Search for a movie or series and ask for it. You will see it here until it is ready to watch.');
      const others = d.requests.filter((r) => r.by !== ME() && (r.state === 'downloading' || VA.requestFlow(r).length));
      q('[data-other-sec]').hidden = !others.length;
      VA.patch(q('[data-other]'), others, (r) => r.id, (r) => card(r, true));
      const lines = m.slice(0, 2).map((r) => {
        const t = d.W[r.work].title;
        const J = VA.journey(r);
        return J.step === 3 ? `${t} is ready to watch.` : J.step === 2 ? `${t} is on its way.` : J.step === 1 ? `${t} is approved.` : `${t} is waiting for approval.`;
      });
      VA.flip(q('[data-sum]'), lines.join(' ') || 'Ask for anything that is missing. You will see it arrive here.');
    };
    render();
    return {
      live() { el.querySelectorAll('.a-preqc').forEach((c) => { const r = d.requests.find((x) => x.id === c.dataset.key); if (!r) return; const s = c.querySelector('[data-strip]'); if (s) s.innerHTML = VA.journeyStrip(r); const st = c.querySelector('[data-flip="state"]'); const txt = VA.journey(r).denied ? 'Not added' : journeyLabel(r); if (st.textContent.split(' · ')[0] !== txt.split(' · ')[0]) VA.flip(st, txt); else st.textContent = txt; c.classList.toggle('is-ready', VA.journey(r).step === 3); }); },
      refresh(ev) { if (ev === 'request:update' || ev === 'job:new' || ev === 'job:done') render(); },
    };
  };

  VIEWS.requests = (el) => {
    el.innerHTML = `<div class="a-pt__page"><h1 class="a-pt__title">Your requests</h1><div class="a-pt__cards a-pt__cards--list" data-list></div></div>`;
    const render = () => {
      const list = mine().slice().sort((a, b) => RANK[a.state] - RANK[b.state] || b.at - a.at);
      VA.patch(el.querySelector('[data-list]'), list, (r) => r.id, (r) => card(r));
      VA.emptyState(el.querySelector('[data-list]'), !list.length, 'You have no requests yet.', 'Search for a movie or series and ask for it.');
    };
    render();
    return { live() { el.querySelectorAll('.a-preqc [data-strip]').forEach((s) => { const r = d.requests.find((x) => x.id === s.closest('.a-preqc').dataset.key); s.innerHTML = VA.journeyStrip(r); }); }, refresh() { render(); } };
  };

  VIEWS.request = (el, id) => {
    const r = d.requests.find((x) => x.id === id);
    if (!r) { el.innerHTML = `<div class="a-pt__page"><div class="a-empty"><p>This request is no longer here.</p><p class="a-empty__sub">See your requests for everything you have asked for.</p></div></div>`; return {}; }
    const w = d.W[r.work];
    if (r.by === ME() && r.state === 'available') r.seen = true;
    el.innerHTML = `<div class="a-pt__reqpage">
      <div class="a-pt__bd"><img src="${SS.art.backdrop(w, { w: 1600, h: 900 })}" alt=""></div>
      <div class="a-pt__page a-pt__req">
        <a class="a-pt__poster" href="#work/${w.id}">${VA.posterImg(w)}</a>
        <div class="a-pt__reqbody">
          <h1 class="a-pt__say" data-say></h1>
          <p class="a-pt__sub">${r.by === ME() ? 'You asked' : `${esc(VA.firstName(r.by))} asked`} for it ${esc(fmt.ago(r.at))}${r.seasons ? ` · Season ${r.seasons.join(', ')}` : ''}</p>
          <div class="a-pt__journey" data-strip></div>
          <p class="a-pt__fine" data-fine></p>
          <p class="a-pt__ov">${esc(w.overview)}</p>
        </div>
      </div>
    </div>`;
    const q = (s) => el.querySelector(s);
    let lastStep = -1;
    const render = () => {
      const J = VA.journey(r);
      q('[data-strip]').innerHTML = J.denied ? '' : VA.journeyStrip(r);
      const say = J.denied ? `${w.title} was not added.` : J.step === 3 ? `${w.title} is ready to watch.` : J.step === 2 ? `${w.title} is on its way.` : J.step === 1 ? `${w.title} is approved.` : `${w.title} is waiting for approval.`;
      if (lastStep !== J.step) { if (lastStep >= 0) VA.flip(q('[data-say]'), say); else q('[data-say]').textContent = say; lastStep = J.step; }
      q('[data-say]').classList.toggle('is-ready', J.step === 3);
      q('[data-fine]').textContent = J.denied ? J.detail : J.step === 3 ? 'Open Plex to start watching.' : J.step === 2 ? `${J.detail}. We will let you know when it is ready.` : J.step === 1 ? 'SlipStream is looking for the best copy.' : 'Alex reviews requests most evenings.';
    };
    render();
    return { live: render, refresh: render };
  };

  VIEWS.work = (el, id, api) => {
    const w = d.W[id];
    if (!w) { el.innerHTML = `<div class="a-pt__page"><div class="a-empty"><p>We could not find that.</p><p class="a-empty__sub">Try searching for it by title.</p></div></div>`; return {}; }
    el.innerHTML = `<div class="a-pt__reqpage">
      <div class="a-pt__bd"><img src="${SS.art.backdrop(w, { w: 1600, h: 900 })}" alt=""></div>
      <div class="a-pt__page a-pt__req">
        <span class="a-pt__poster">${VA.posterImg(w)}</span>
        <div class="a-pt__reqbody">
          <h1 class="a-pt__title">${esc(w.title)}</h1>
          <p class="a-pt__sub a-pt__meta">${VA.workMeta(w)}</p>
          <p class="a-pt__ov">${esc(w.overview)}</p>
          <div class="a-pt__act" data-act></div>
        </div>
      </div>
    </div>`;
    const box = el.querySelector('[data-act]');
    let picking = false;
    const render = () => {
      const r = requestFor(w);
      const ss = w.type === 'series' ? seasonState(w) : [];
      const open = ss.filter((s) => !s.done);
      if (r && r.state !== 'available') { box.innerHTML = `<a class="a-pt__inline" href="#request/${r.id}"><span class="a-pt__inline-t">${esc(r.by === ME() ? journeyLabel(r) : `${VA.firstName(r.by)} asked for it · ${journeyLabel(r)}`)}</span>${VA.journeyStrip(r)}</a>`; return; }
      if (w.type === 'series' && picking) {
        box.innerHTML = `<form class="a-pt__seasons" data-pick><p class="a-pt__fine">Which seasons would you like?</p>${ss.map((s) => `<label class="a-pt__season ${s.done ? 'is-done' : ''}"><input type="checkbox" name="s" value="${s.n}" ${s.done ? 'disabled checked' : s.n === open[0]?.n ? 'checked' : ''}><span><b>Season ${s.n}</b><span>${s.done ? `All ${s.total} episodes are here` : s.have ? `${s.have} of ${s.total} episodes are here` : `${s.total} episodes · ${s.year}`}</span></span></label>`).join('')}<div class="a-pt__pickacts">${VA.btn('Request', { kind: 'primary', attrs: 'type="submit" data-pick-go' })}${VA.btn('Cancel', { kind: 'ghost', attrs: 'type="button" data-pick-cancel' })}</div></form>`;
        const f = box.querySelector('[data-pick]');
        const sync = () => { const n = f.querySelectorAll('input:checked:not(:disabled)').length; const b = f.querySelector('[data-pick-go]'); b.disabled = !n; b.querySelector('span').textContent = n ? `Request ${fmt.plural(n, 'season')}` : 'Choose a season'; };
        f.addEventListener('change', sync);
        f.addEventListener('submit', (e) => { e.preventDefault(); const seasons = [...f.querySelectorAll('input:checked:not(:disabled)')].map((i) => +i.value); ask(seasons); });
        sync();
        return;
      }
      if (inLibrary(w) && (w.type !== 'series' || !open.length)) {
        box.innerHTML = `<p class="a-pt__ready"><span class="a-pt__readydot"></span>Ready to watch on Plex</p>${w.type === 'series' ? `<p class="a-pt__fine">${w.counts.available} of ${w.counts.total} episodes are here${w.nextAir ? `. The next one airs ${esc(fmt.until(w.nextAir))}` : ''}.</p>` : ''}`;
        return;
      }
      if (inLibrary(w)) {
        box.innerHTML = `<p class="a-pt__ready"><span class="a-pt__readydot"></span>Ready to watch on Plex</p><p class="a-pt__fine">${w.counts.available} of ${w.counts.total} episodes are here. Ask for the rest and they arrive as they air.</p><div class="a-pt__more">${VA.btn('Request more episodes', { ic: icon('plus', { size: 16 }), attrs: 'data-ask' })}</div>`;
        return;
      }
      box.innerHTML = `${VA.btn(w.status === 'unreleased' ? 'Request for release day' : 'Request', { kind: 'primary', ic: icon('plus', { size: 16 }), attrs: 'data-ask' })}<p class="a-pt__fine">${w.status === 'unreleased' && w.release ? `Out ${esc(fmt.until(w.release))}. Alex approves requests before anything is added.` : 'Alex approves requests before anything is added.'}</p>`;
    };
    const ask = (seasons) => {
      const r = { id: 'rq' + Date.now().toString(36), work: w.id, by: ME(), at: SS.sim.clock, state: 'pending', seasons: seasons || undefined, note: '' };
      d.requests.push(r);
      SS.sim.emit('request:update', r);
      picking = false;
      api.toast(`Requested ${w.title}. Alex will take a look.`);
      render();
    };
    render();
    return {
      click(e) {
        if (e.target.closest('[data-ask]')) { if (w.type === 'series') { picking = true; render(); } else ask(); return true; }
        if (e.target.closest('[data-pick-cancel]')) { picking = false; render(); return true; }
        return false;
      },
      live() { const r = requestFor(w); if (r && r.state !== 'available') { const s = box.querySelector('.a-pt__inline'); if (s) s.innerHTML = `<span class="a-pt__inline-t">${esc(journeyLabel(r))}</span>${VA.journeyStrip(r)}`; } },
      refresh(ev, p) { if (!picking && (p?.work === w.id)) render(); },
    };
  };

  VIEWS.search = (el, arg) => {
    el.innerHTML = `<div class="a-pt__page"><label class="a-psearch a-pt__psearch">${icon('search', { size: 17 })}<input type="search" placeholder="Search movies and series" aria-label="Search movies and series" enterkeyhint="search" autocomplete="off" data-q value="${esc(arg ? decodeURIComponent(arg) : '')}"></label><div data-results></div></div>`;
    const input = el.querySelector('[data-q]');
    const out = el.querySelector('[data-results]');
    // One chip form for every state: the journey stop it has reached, or the ask itself.
    const badge = (w) => {
      const r = requestFor(w);
      if (inLibrary(w) && (!r || r.state === 'available')) return '<span class="a-pt__badge is-ready">In the library</span>';
      if (r) return `<span class="a-pt__badge">${esc(VA.JOURNEY[VA.journey(r).step])}</span>`;
      return `<span class="a-pt__badge is-ask">${icon('plus', { size: 13 })}Request</span>`;
    };
    const render = (qv = input.value) => {
      const qs = qv.trim().toLowerCase();
      const list = qs ? d.works.filter((w) => w.title.toLowerCase().includes(qs) || w.genres.some((g) => g.toLowerCase().includes(qs))) : d.works.filter((w) => !inLibrary(w)).slice(0, 12);
      out.innerHTML = `${qs ? '' : '<div class="a-section-head"><h2 class="a-h2">Not in the library yet</h2></div>'}${list.length ? `<div class="a-pt__results">${list.map((w) => `<a class="a-pt__res" href="#work/${w.id}">${VA.posterImg(w)}<b>${esc(w.title)}</b><span>${w.type === 'series' ? 'Series' : 'Movie'} · ${w.year}</span>${badge(w)}</a>`).join('')}</div>` : `<div class="a-empty"><p>Nothing matches “${esc(qv)}”.</p><p class="a-empty__sub">Try another title, or a genre such as Drama.</p></div>`}`;
    };
    input.addEventListener('input', () => render());
    render();
    return { search(v) { input.value = v; render(v); } };
  };

  VIEWS.library = (el) => {
    const list = d.works.filter(inLibrary).sort((a, b) => b.added - a.added);
    el.innerHTML = `<div class="a-pt__page"><h1 class="a-pt__title">Library</h1><p class="a-pt__sub">${list.length} movies and series ready to watch on Plex.</p><div class="a-pt__results">${list.map((w) => `<a class="a-pt__res" href="#work/${w.id}">${VA.posterImg(w)}<b>${esc(w.title)}</b><span>${w.type === 'series' ? 'Series' : 'Movie'} · ${w.year}</span></a>`).join('')}</div></div>`;
    return {};
  };
})();
