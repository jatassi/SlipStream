// PROTOTYPE. House Lights: the Member portal, one set of components from phone to desk.
(function () {
  const SS = window.SS, VB = window.VB;
  const D = SS.data, F = SS.fmt, ART = SS.art;
  const esc = F.esc;
  const icon = VB.icon;
  const ME = D.member;

  const parse = (v) => {
    const [a, b] = (v || 'home').split('/');
    if (a === 'work' && D.W[b]) return { name: 'work', id: b };
    if (a === 'request' && VB.reqs().some((r) => r.id === b)) return { name: 'request', id: b };
    return { name: ['home', 'search', 'library', 'requests'].includes(a) ? a : 'home' };
  };
  const viewOf = (r) => (r.id ? `${r.name}/${r.id}` : r.name);

  VB.portal = function (ctx) {
    const { root } = ctx;
    const wide = ctx.shell === 'portal-wide';
    root.innerHTML = `<div class="b-app b-portal ${wide ? 'b-portal--wide' : 'b-portal--ph'}">
      <div class="b-spillhost"></div>
      <main class="b-main" tabindex="-1"><div class="b-view"></div></main>
      <header class="b-pbar">
        <span class="b-pbar__lead"></span>
        ${wide ? `<button class="b-search b-search--portal" type="button" data-go="search">${icon('search', 16)}<span>Search movies and series</span></button>` : ''}
        <nav class="b-pbar__r" aria-label="Portal">
          ${wide ? '' : `<button class="b-iconbtn" type="button" data-go="search" aria-label="Search">${icon('search', 20)}</button>`}
          <a class="${wide ? 'b-pbar__link' : 'b-iconbtn'}" href="${VB.href('library')}" data-go="library" aria-label="Library">${wide ? 'Library' : icon('library-big', 20)}</a>
          ${wide ? `<a class="b-pbar__link" href="${VB.href('requests')}" data-go="requests">Requests</a>` : ''}
          <button class="b-iconbtn" type="button" data-act="bell" aria-label="Notifications" aria-expanded="false">${icon('bell', 20)}<i class="b-iconbtn__dot" aria-hidden="true" hidden></i></button>
          <button class="b-acct" type="button" data-act="settings" aria-label="Settings" aria-expanded="false">${VB.av(ME)}</button>
        </nav>
      </header>
      <div class="b-layer"></div>
    </div>`;
    const app = root.querySelector('.b-app');
    const main = root.querySelector('.b-main');
    const viewEl = root.querySelector('.b-view');
    const bar = root.querySelector('.b-pbar');
    const spill = VB.spill(root.querySelector('.b-spillhost'));
    const live = VB.live(ctx);
    const pop = VB.popover(ctx, root);
    let route = parse(ctx.view);
    const stack = [];
    const notes = [];
    const SRCH = { q: '' };
    const tW = wide ? 220 : 150;
    let rowTimer = 0;
    const now = VB.nowShowing(ctx, (h) => {
      if (route.name !== 'home') return;
      const slot = viewEl.querySelector('[data-mhero]');
      if (!slot) return;
      slot.classList.add('is-out');
      setTimeout(() => { slot.outerHTML = heroHTML(h); live.scan(viewEl); viewEl.querySelector('[data-mhero]').classList.add('is-in'); }, 260);
      spill.set(D.W[h.work]);
    });

    function lead() {
      const el = bar.querySelector('.b-pbar__lead');
      el.innerHTML = route.name === 'home' || wide
        ? `<a class="b-mark" href="${VB.href('home')}" data-go="home">${VB.mark()}SlipStream</a>`
        : `<button class="b-pback" type="button" data-back>${icon('chevron-left', 24)}<span>${esc(backLabel())}</span></button>`;
    }
    const backLabel = () => { const p = stack[stack.length - 1]; return p ? ({ home: 'Home', search: 'Search', library: 'Library', requests: 'Requests', work: 'Back', request: 'Back' }[parse(p.view).name]) : 'Home'; };

    function go(view, { kbd = false, el = null, back = false, y = 0 } = {}) {
      const next = parse(view);
      if (!back && viewOf(next) === viewOf(route)) return;
      if (!back) stack.push({ view: viewOf(route), y: main.scrollTop });
      const update = () => { route = next; ctx.setView(viewOf(route)); render(); main.scrollTop = back ? y : 0; };
      const morph = el && el.isConnected && next.name === 'work';
      if (morph) el.style.viewTransitionName = 'b-poster';
      document.documentElement.dataset.bvt = morph ? 'down' : 'nav';
      const t = VB.vt(update, { kbd });
      const clear = () => { delete document.documentElement.dataset.bvt; };
      if (t) t.finished.finally(clear); else clear();
    }
    function back(kbd) {
      const p = stack.pop() || { view: 'home', y: 0 };
      go(p.view, { kbd, back: true, y: p.y });
    }

    // ---------- Views ----------
    function heroHTML(h) {
      const w = D.W[h.work];
      const sub = h.episode ? `New episode · ${F.ep(h.episode)} ${w.seasons[h.episode[0] - 1].episodes[h.episode[1] - 1].title}` : `${w.year} · ${w.genres.join(', ')} · ${F.runtime(w.runtime)}`;
      return `<section class="b-mhero" data-mhero style="${VB.tint(w)}">
        <div class="b-mhero__frame"><img src="${ART.backdrop(w, wide ? { w: 1680, h: 720 } : { w: 786, h: 940 })}" alt=""><div class="b-mhero__shade"></div></div>
        <div class="b-mhero__body">
          ${VB.logo(w, { maxW: wide ? 560 : 330, max: wide ? 84 : 60, min: 30, tag: 'h1' })}
          <p class="b-meta"><span>${esc(sub)}</span></p>
          <p class="b-mhero__arr">New in the library <span data-lv="ago|${h.at.getTime()}">${esc(F.ago(h.at))}</span>.</p>
          <div class="b-mhero__act"><button class="b-btn b-btn--primary b-btn--lg" type="button" data-cmd="play" data-work="${w.id}">${icon('play', 18)}Play</button><a class="b-btn b-btn--veil b-btn--lg" href="${VB.href('work/' + w.id)}" data-open="${w.id}">More about it</a></div>
        </div>
      </section>`;
    }
    const myReqs = () => VB.reqs().filter((r) => r.by === ME).sort((a, b) => b.at - a.at);
    const onItsWay = () => VB.developing().filter((x) => x.j.state === 'running');
    const memberTile = ({ w, ep }) => `<a class="b-tile b-dtile" href="${VB.href('work/' + w.id)}" data-open="${w.id}" data-key="${w.id}|${VB.epKey(ep)}" style="${VB.tint(w)}">
        <div class="b-tile__art">${VB.dev(w, { ep })}</div>
        <div class="b-dtile__cap"><span class="b-tile__t">${esc(w.title)}${ep ? ` <span class="b-dim">${F.ep(ep)}</span>` : ''}</span><span class="b-dtile__s" data-lv="wmember|${w.id}${ep ? '|' + VB.epKey(ep) : ''}">${esc(VB.memberLine(VB.acq(w.id, ep)))}</span></div></a>`;
    const posterTile = (w, sub) => `<a class="b-tile b-tile--lit" href="${VB.href('work/' + w.id)}" data-open="${w.id}" style="${VB.tint(w)}"><div class="b-tile__art">${VB.poster(w)}</div><div class="b-dtile__cap"><span class="b-tile__t">${esc(w.title)}</span><span class="b-dtile__s">${esc(sub)}</span></div></a>`;

    function homeHTML() {
      const mine = myReqs().filter((r) => VB.reqState(r) !== 'denied');
      const way = onItsWay();
      const fresh = [];
      const seen = new Set();
      for (const x of D.history) if (x.kind === 'imported' && !seen.has(x.work)) { seen.add(x.work); fresh.push(x); }
      for (const w of D.works.filter((w) => VB.status(w).s === 'available').sort((a, b) => b.added - a.added)) if (!seen.has(w.id) && fresh.length < 10) { seen.add(w.id); fresh.push({ work: w.id, at: w.added }); }
      return `${heroHTML(now.get())}
        <div class="b-mcol">
          <section class="b-msec" aria-labelledby="b-tix-h"><header class="b-sec__h"><h2 class="b-h2" id="b-tix-h">Your tickets</h2><a class="b-link" href="${VB.href('requests')}" data-go="requests">All requests${icon('chevron-right', 16)}</a></header>
            ${mine.length ? `<div class="b-tix">${mine.map((r) => VB.ticket(r, { titleW: tW })).join('')}</div>` : `<p class="b-empty">No tickets yet. Search for a movie or series that is not here, and ask for it. You will see it develop here.</p>`}</section>
          <section class="b-msec" aria-labelledby="b-way-h"><header class="b-sec__h"><h2 class="b-h2" id="b-way-h">On its way</h2></header>
            <div class="b-drow b-house" data-wayrow>${way.length ? way.map(memberTile).join('') : '<p class="b-empty">Nothing on its way right now.</p>'}</div></section>
          <section class="b-msec" aria-labelledby="b-new-h"><header class="b-sec__h"><h2 class="b-h2" id="b-new-h">New in the library</h2><a class="b-link" href="${VB.href('library')}" data-go="library">Library${icon('chevron-right', 16)}</a></header>
            <div class="b-arow b-house">${fresh.slice(0, 10).map((x) => posterTile(D.W[x.work], VB.cap(F.ago(x.at)))).join('')}</div></section>
          <section class="b-msec" aria-labelledby="b-soon-h"><header class="b-sec__h"><h2 class="b-h2" id="b-soon-h">Coming soon</h2></header>
            <div class="b-cslist b-cslist--grid b-house">${VB.comingSoon(wide ? 6 : 4).map(VB.comingCard).join('')}</div></section>
        </div>`;
    }

    function libraryHTML(q, isSearch) {
      q = (q || '').trim().toLowerCase();
      const pool = D.works.filter((w) => !q || w.title.toLowerCase().includes(q) || w.genres.some((g) => g.toLowerCase().includes(q)));
      const here = pool.filter((w) => w.type === 'series' || VB.status(w).s === 'available' || VB.status(w).s === 'upgradable');
      const notYet = pool.filter((w) => !here.includes(w));
      const card = (w) => {
        const s = VB.status(w);
        const r = VB.reqs().find((x) => x.work === w.id && VB.reqState(x) !== 'denied');
        const sub = r ? (VB.reqState(r) === 'downloading' ? VB.memberLine(s.j) : r.by === ME ? 'You asked for it' : `${VB.first(r.by)} asked for it`) : s.j && s.j.state === 'running' ? VB.memberLine(s.j) : s.s === 'unreleased' ? `Out ${w.release ? w.release.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' }) : w.year}` : 'Not here yet';
        return `<a class="b-tile b-tile--req" href="${VB.href('work/' + w.id)}" data-open="${w.id}" style="${VB.tint(w)}"><div class="b-tile__art">${VB.art(w)}</div><div class="b-dtile__cap"><span class="b-tile__t">${esc(w.title)}</span><span class="b-dtile__s">${esc(sub)}</span></div></a>`;
      };
      return `${notYet.length && q ? `<section class="b-msec"><header class="b-sec__h"><h2 class="b-h2">Not here yet</h2><p class="b-sec__aside">Ask and Alex gets it</p></header><div class="b-mgrid b-house">${notYet.map(card).join('')}</div></section>` : ''}
        <section class="b-msec"><header class="b-sec__h"><h2 class="b-h2">${q ? 'In the library' : isSearch ? 'Browse the library' : 'Everything here'}</h2><p class="b-sec__aside">${F.plural(here.length, 'Work')}</p></header>
          ${here.length ? `<div class="b-mgrid b-house">${here.map((w) => posterTile(w, w.type === 'series' ? F.plural(w.seasons.length, 'season') : String(w.year))).join('')}</div>` : `<p class="b-empty">Nothing in the library matches “${esc(q)}”.</p>`}</section>
        ${notYet.length && !q ? `<section class="b-msec"><header class="b-sec__h"><h2 class="b-h2">Not here yet</h2><p class="b-sec__aside">Ask and Alex gets it</p></header><div class="b-mgrid b-house">${notYet.map(card).join('')}</div></section>` : ''}`;
    }
    function searchHTML() {
      return `<div class="b-mcol b-mcol--top">
        <h1 class="b-h1">${route.name === 'library' ? 'Library' : 'Search'}</h1>
        <label class="b-psearch b-psearch--portal">${icon('search', 18)}<span class="b-sr">Search movies and series</span><input type="search" enterkeyhint="search" placeholder="Movies, series, genres" value="${esc(SRCH.q)}" data-msearch autocomplete="off"></label>
        <div data-mresults>${libraryHTML(SRCH.q, route.name === 'search')}</div>
      </div>`;
    }

    function workHTML(r) {
      const w = D.W[r.id];
      const s = VB.status(w);
      const req = VB.reqs().find((x) => x.work === w.id && VB.reqState(x) !== 'denied');
      const meta = w.type === 'movie' ? [String(w.year), F.runtime(w.runtime), w.cert, w.genres.join(', ')] : [w.network, F.plural(w.seasons.length, 'season'), w.cert, w.genres.join(', ')];
      let action;
      if (req) action = VB.ticket(req, { big: true, title: false, art: false });
      else if (w.type === 'series' || s.s === 'available' || s.s === 'upgradable') action = `<div class="b-mact"><button class="b-btn b-btn--primary b-btn--lg" type="button" data-cmd="play" data-work="${w.id}">${icon('play', 18)}Play</button><p class="b-dim">${w.type === 'series' ? `${w.counts.available} episodes to watch` : `In the library in ${esc(VB.coverage(w).filter((x) => x.c === 'met').map((x) => x.t.name).join(' and ') || '1080p')}`}</p></div>`;
      else if (s.j && s.j.state === 'running') action = `<div class="b-mact"><p class="b-mact__live" data-lv="wmember|${w.id}">${esc(VB.memberLine(s.j))}</p></div>`;
      else action = requestBlock(w);
      return `<article class="b-mwork" style="${VB.tint(w)}">
        <div class="b-mwork__stage"><img src="${ART.backdrop(w, wide ? { w: 1680, h: 720 } : { w: 786, h: 640 })}" alt=""><div class="b-mwork__shade"></div></div>
        <div class="b-mwork__head">
          <div class="b-mwork__poster">${VB.art(w, { vt: 'b-poster' })}</div>
          <div class="b-mwork__id">${VB.logo(w, { maxW: wide ? 620 : 236, max: wide ? 80 : 50, min: 24 })}<p class="b-meta">${meta.filter(Boolean).slice(0, wide ? 4 : 3).map((m) => `<span>${esc(m)}</span>`).join('')}</p></div>
        </div>
        <div class="b-mcol">
          <div class="b-mwork__act" data-mact>${action}</div>
          <p class="b-mwork__ov">${esc(w.overview)}</p>
          <dl class="b-dl b-dl--inline">${(w.type === 'movie' ? [['Director', w.director], ['Starring', w.cast.slice(0, 3).join(', ')]] : [['Created by', w.creator], ['Starring', w.cast.slice(0, 3).join(', ')]]).map(([k, v]) => `<div><dt>${k}</dt><dd>${esc(v)}</dd></div>`).join('')}</dl>
        </div>
      </article>`;
    }
    function requestBlock(w) {
      const q = D.A[ME].quota[w.type];
      const quota = `You have used ${q[0]} of ${q[1]} ${w.type === 'movie' ? 'movie' : 'series'} requests this week.`;
      if (w.type === 'series') {
        return `<div class="b-mreq" data-mreq>
          <button class="b-btn b-btn--primary b-btn--lg" type="button" data-mreq-open aria-expanded="false">${icon('ticket', 18)}Request</button>
          <fieldset class="b-seasons" hidden><legend>Which seasons?</legend>
            <label class="b-check b-check--all"><input type="checkbox" data-allseasons><span class="b-check__box" aria-hidden="true">${icon('check', 14)}</span>All seasons, and new ones as they air</label>
            ${w.seasons.map((s) => `<label class="b-check"><input type="checkbox" value="${s.n}" data-season-pick ${s.n === w.seasons.length ? 'checked' : ''}><span class="b-check__box" aria-hidden="true">${icon('check', 14)}</span>Season ${s.n}<span class="b-dim">${s.year} · ${s.episodes.length} episodes</span></label>`).join('')}
            <button class="b-btn b-btn--primary" type="button" data-mreq-send="${w.id}">Request 1 season</button>
          </fieldset>
          <p class="b-dim">${quota}</p></div>`;
      }
      return `<div class="b-mreq" data-mreq><button class="b-btn b-btn--primary b-btn--lg" type="button" data-mreq-send="${w.id}">${icon('ticket', 18)}Request</button><p class="b-dim">${w.status === 'unreleased' ? `Out ${w.release ? esc(F.day(w.release)) : w.year}. Ask now and it arrives on release. ` : ''}${quota}</p></div>`;
    }

    // The ticket is the request's one status object; below it, what happened and when.
    function requestHTML(r) {
      const req = VB.reqs().find((x) => x.id === r.id);
      const w = D.W[req.work];
      const st = VB.reqState(req);
      const admin = VB.first(D.me);
      const when = (d) => `${F.day(d)} at ${F.time(d)}`;
      const events = [[`${req.by === ME ? 'You' : VB.name(req.by)} asked for it`, when(req.at)]];
      if (st === 'denied') events.push([`${admin} said not this time`, when(VB.decidedAt(req))]);
      else if (st !== 'pending') events.push([`${admin} approved it`, when(VB.decidedAt(req))]);
      if (st === 'available' && req.doneAt) events.push(['It arrived, ready to watch', when(req.doneAt)]);
      return `<article class="b-mreqpage" style="${VB.tint(w)}">
        <div class="b-mwork__stage"><img src="${ART.backdrop(w, wide ? { w: 1680, h: 720 } : { w: 786, h: 640 })}" alt=""><div class="b-mwork__shade"></div></div>
        <div class="b-mcol b-mreqpage__col">
          ${VB.ticket(req, { big: true, link: false, titleW: wide ? 400 : 150 })}
          <section class="b-msec" aria-labelledby="b-rh-h"><header class="b-sec__h"><h2 class="b-h2" id="b-rh-h">History</h2></header>
            <ol class="b-hist">${events.map(([t, at]) => `<li><span>${esc(t)}</span><span class="b-dim">${esc(at)}</span></li>`).join('')}</ol></section>
          <a class="b-link b-mreqpage__more" href="${VB.href('work/' + w.id)}" data-open="${w.id}">About ${esc(w.title)}${icon('chevron-right', 16)}</a>
        </div>
      </article>`;
    }
    function requestsHTML() {
      const mine = myReqs();
      const house = VB.reqs().filter((r) => r.by !== ME && VB.reqState(r) !== 'denied').sort((a, b) => b.at - a.at);
      return `<div class="b-mcol b-mcol--top">
        <h1 class="b-h1">Requests</h1>
        <section class="b-msec"><header class="b-sec__h"><h2 class="b-h2">Yours</h2></header>${mine.length ? `<div class="b-tix">${mine.map((r) => VB.ticket(r, { titleW: tW })).join('')}</div>` : '<p class="b-empty">You have not asked for anything yet. Find a Work in Search and press Request.</p>'}</section>
        <section class="b-msec"><header class="b-sec__h"><h2 class="b-h2">From the house</h2></header><div class="b-tix">${house.map((r) => VB.ticket(r, { titleW: tW })).join('')}</div></section>
      </div>`;
    }

    const VIEWS = {
      home: [homeHTML, () => D.W[now.get().work]],
      search: [searchHTML, () => null],
      library: [searchHTML, () => null],
      work: [workHTML, () => D.W[route.id]],
      request: [requestHTML, () => D.W[VB.reqs().find((x) => x.id === route.id).work]],
      requests: [requestsHTML, () => null],
    };
    function render() {
      live.reset();
      pop.close();
      clearTimeout(rowTimer);
      app.dataset.route = route.name;
      const [html, light] = VIEWS[route.name];
      viewEl.innerHTML = html(route);
      live.scan(viewEl);
      VB.fillStills(viewEl);
      spill.set(light());
      spill.dim(1);
      bar.classList.remove('is-solid');
      lead();
      if (route.name === 'search' && wide) viewEl.querySelector('[data-msearch]')?.focus({ preventScroll: true });
    }

    function sendRequest(w, seasons) {
      const r = { id: 'm' + (VB.local.mine.length + 1), work: w.id, by: ME, at: SS.sim.clock, state: 'pending', seasons, note: '' };
      VB.local.mine.push(r);
      const box = viewEl.querySelector('[data-mact]');
      box.innerHTML = VB.ticket(r, { big: true, title: false, art: false });
      live.scan(box);
      const t = box.firstElementChild;
      if (!VB.reduced()) t.animate([{ opacity: 0, transform: 'translateY(10px) scale(0.98)' }, { opacity: 1, transform: 'none' }], { duration: 420, easing: 'cubic-bezier(0.23, 1, 0.32, 1)' });
      VB.toast(root, `Asked for ${w.title}. Alex gets a notification`);
    }

    // ---------- Events ----------
    ctx.listen(root, 'click', (e) => {
      if (e.button !== 0) return;
      const kbd = e.detail === 0;
      const a = e.target.closest('a[href]');
      if (a && (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey)) return;
      const t = e.target.closest('[data-go],[data-open],[data-back],[data-cmd],[data-act],[data-mreq-open],[data-mreq-send]');
      if (!t || !root.contains(t)) return;
      if (a) e.preventDefault();
      const d = t.dataset;
      if (d.go) return go(d.go, { kbd });
      if (d.open) return go(`work/${d.open}`, { kbd, el: t.querySelector('.b-tile__art > *') });
      if ('back' in d) return back(kbd);
      if (d.cmd === 'play') { e.preventDefault(); return VB.toast(root, `Opening ${D.W[d.work].title} on Plex`); }
      if (d.act === 'bell') return pop.show(t, bellHTML(), { width: 340 });
      if (d.act === 'settings') return pop.show(t, `<div class="b-acctpop"><p class="b-acctpop__n">${esc(VB.name(ME))}</p><p class="b-dim">${esc(D.A[ME].email)}</p></div><div class="b-menu" role="menu"><button type="button" role="menuitem">${icon('key-round', 16)}PIN and passkey</button><button type="button" role="menuitem">${icon('bell', 16)}Notifications</button><hr><button type="button" role="menuitem">${icon('log-out', 16)}Sign out</button></div>`, { width: 260, cls: 'b-pop--menu' });
      if ('mreqOpen' in d) {
        const fs = t.parentElement.querySelector('.b-seasons');
        fs.hidden = !fs.hidden;
        t.setAttribute('aria-expanded', String(!fs.hidden));
        t.hidden = !fs.hidden;
        if (!fs.hidden && !VB.reduced()) fs.animate([{ opacity: 0, transform: 'translateY(-4px)' }, { opacity: 1, transform: 'none' }], { duration: 220, easing: 'cubic-bezier(0.23, 1, 0.32, 1)' });
        return;
      }
      if (d.mreqSend) {
        const w = D.W[d.mreqSend];
        const picks = [...viewEl.querySelectorAll('[data-season-pick]:checked')].map((x) => +x.value);
        if (w.type === 'series' && !picks.length) return;
        return sendRequest(w, w.type === 'series' ? picks : undefined);
      }
    });
    ctx.listen(root, 'change', (e) => {
      if (!e.target.matches('[data-season-pick],[data-allseasons]')) return;
      const fs = e.target.closest('.b-seasons');
      if (e.target.matches('[data-allseasons]')) for (const c of fs.querySelectorAll('[data-season-pick]')) c.checked = e.target.checked;
      else fs.querySelector('[data-allseasons]').checked = [...fs.querySelectorAll('[data-season-pick]')].every((c) => c.checked);
      const n = fs.querySelectorAll('[data-season-pick]:checked').length;
      const btn = fs.querySelector('[data-mreq-send]');
      btn.textContent = n ? `Request ${F.plural(n, 'season')}` : 'Pick a season';
      btn.disabled = !n;
    });
    ctx.listen(root, 'input', (e) => {
      if (!e.target.matches('[data-msearch]')) return;
      SRCH.q = e.target.value;
      const box = viewEl.querySelector('[data-mresults]');
      box.innerHTML = libraryHTML(SRCH.q, route.name === 'search');
      live.scan(box);
    });
    ctx.listen(main, 'scroll', () => {
      const y = main.scrollTop;
      bar.classList.toggle('is-solid', y > 8);
      spill.dim(1 - Math.min(1, y / 900) * 0.55);
    }, { passive: true });
    ctx.listen(window, 'keydown', (e) => {
      if (e.key === 'Escape' && !e.defaultPrevented && route.name !== 'home' && !e.target.closest('input')) back(true);
    });

    function bellHTML() {
      const base = [
        { t: 'Amber Protocol was approved', d: 'It starts as soon as a copy turns up', at: new Date(D.now.getTime() - 4.8 * 864e5) },
        { t: 'Paper Lanterns is ready to watch', d: 'Rosa asked for it. It is in 2160p HDR', at: new Date(D.now.getTime() - 12 * 6e4) },
      ];
      const all = notes.concat(base);
      return `<header class="b-pop__h"><h2 class="b-h3">Notifications</h2></header><ul class="b-notes" role="list">${all.map((n) => `<li><i class="b-dot b-dot--ink" aria-hidden="true"></i><div><span class="b-notes__t">${esc(n.t)}</span><span class="b-notes__d">${esc(n.d)}</span><span class="b-notes__a">${esc(F.ago(n.at))}</span></div></li>`).join('')}</ul>`;
    }

    // A request turning available tears its stub where it is on screen.
    ctx.on('request:update', (r) => {
      if (r.state !== 'available') return;
      const w = D.W[r.work];
      notes.unshift({ t: `${w.title} is ready to watch`, d: r.by === ME ? 'You asked for it' : `${VB.first(r.by)} asked for it`, at: SS.sim.clock });
      bar.querySelector('.b-iconbtn__dot').hidden = false;
      for (const el of viewEl.querySelectorAll(`[data-ticket="${r.id}"]`)) {
        const big = el.classList.contains('is-big');
        const link = el.querySelector('.b-ticket__main').tagName === 'A';
        el.outerHTML = VB.ticket(r, { big, link, title: !!el.querySelector('.b-ticket__title'), art: !!el.querySelector('.b-ticket__art'), titleW: big ? (wide ? 400 : 150) : tW });
        const fresh = viewEl.querySelector(`[data-ticket="${r.id}"]`);
        live.scan(fresh);
        VB.tear(fresh);
      }
      if (route.name === 'request') { const y = main.scrollTop; render(); main.scrollTop = y; VB.tear(viewEl.querySelector('.b-ticket')); }
      VB.toast(root, `${w.title} is ready to watch`);
    });
    ctx.on('job:done', (j) => {
      if (j.kind !== 'import' || route.name !== 'home') return;
      clearTimeout(rowTimer);
      rowTimer = setTimeout(async () => {
        const row = viewEl.querySelector('[data-wayrow]');
        if (!row) return;
        const keep = new Set(onItsWay().map((x) => x.key));
        await VB.leave([...row.children].filter((c) => c.dataset.key && !keep.has(c.dataset.key)));
        VB.flip(row, () => { const way = onItsWay(); row.innerHTML = way.length ? way.map(memberTile).join('') : '<p class="b-empty">Nothing on its way right now.</p>'; live.scan(row); });
      }, 4200);
    });

    render();
    return { unmount() { clearTimeout(rowTimer); } };
  };
})();
