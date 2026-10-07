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

  const STOPS = ['Requested', 'Approved', 'On its way', 'Arrived'];
  const soon = (s) => (s < 60 ? 'less than a minute' : s < 3600 ? `about ${F.plural(Math.max(1, Math.round(s / 60)), 'minute')}` : `about ${F.plural(Math.round(s / 3600), 'hour')}`);
  const first = (id) => d.A[id].name.split(' ')[0];

  VC.reqView = (r) => {
    const w = d.W[r.work];
    const j = VC.journey(w);
    if (r.state === 'denied') return { key: 'denied', pos: 0, stop: 0, title: 'Not added', line: r.reason ? `Alex said: ${r.reason}` : 'Alex did not add this one' };
    if (r.state === 'available') return { key: 'arrived', pos: 3, stop: 3, title: 'Ready to watch', line: `Arrived ${F.ago(r.doneAt || VC.clock())}. It is in Plex now.` };
    if (j) return { key: 'moving', pos: 2 + Math.min(0.97, j.p), stop: 2, title: 'On its way', line: j.state === 'stalled' ? 'Taking longer than usual' : `Ready in ${soon(j.eta)}`, live: true };
    if (r.state === 'approved' || r.state === 'downloading') return { key: 'approved', pos: 1.3, stop: 1, title: 'Approved', line: w.status === 'unreleased' ? `Arrives once it is released ${F.until(w.release)}` : 'Looking for a good copy' };
    return { key: 'pending', pos: 0, stop: 0, title: 'Requested', line: 'Waiting for Alex to approve' };
  };
  const reqFor = (w) => d.requests.find((r) => r.work === w.id && r.state !== 'denied' && r.state !== 'available') || d.requests.find((r) => r.work === w.id && r.by === d.member);

  function strip(r, big = false) {
    const v = VC.reqView(r);
    const decided = r.decidedAt || (r.state !== 'pending' ? new Date(r.at.getTime() + 52 * 60e3) : null);
    const times = [F.ago(r.at), decided && v.stop >= 1 ? F.ago(decided) : '', v.key === 'moving' ? v.line.replace('Ready in ', 'in ') : '', r.doneAt && v.key === 'arrived' ? F.ago(r.doneAt) : ''];
    return `<div class="c-rq${big ? ' c-rq--big' : ''}" data-rq="${r.id}" data-k="${v.key}" style="--p:${v.pos.toFixed(4)}">
      <div class="c-rq__line"><span class="c-rq__fill"></span></div>
      ${STOPS.map((s, i) => `<span class="c-rq__stop" data-i="${i}" style="--i:${i}"${i <= v.stop ? ' data-on' : ''}></span>`).join('')}
      <span class="c-rq__train" aria-hidden="true"></span>
      <ol class="c-rq__labels">${STOPS.map((s, i) => `<li style="--i:${i}"${i <= v.stop ? ' data-on' : ''}${i === v.stop ? ' aria-current="step"' : ''}>${s}${big ? `<time data-t="${i}">${esc(times[i])}</time>` : ''}</li>`).join('')}</ol>
    </div>`;
  }

  VC.portal = (ctx) => {
    const root = ctx.root;
    const wide = !ctx.phone;
    root.classList.add('c-app', 'c-app--portal', wide ? 'c-portal--wide' : 'c-portal--phone');
    const me = d.A[d.member];
    root.innerHTML = `
      <header class="c-phead">
        <div class="c-phead__in">
          <button class="c-phead__back" data-back aria-label="Back" hidden>${I('chevron-left', 24)}</button>
          <a class="c-phead__brand" href="#" data-go="home">${VC.logo(20)}<span>SlipStream</span></a>
          ${wide ? `<nav class="c-phead__nav" aria-label="Main">${[['home', 'Home'], ['library', 'Library'], ['requests', 'Requests']].map(([k, l]) => `<a href="#" data-go="${k}" data-nav="${k}">${l}</a>`).join('')}</nav>
            <button class="c-psearchbtn" data-go="search">${I('search', 16)}<span>Search movies and series</span></button>` : ''}
          <div class="c-phead__r">
            ${wide ? '' : `<button class="c-icon-btn" data-go="search" aria-label="Search">${I('search', 21)}</button><button class="c-icon-btn" data-go="library" aria-label="Library">${I('library-big', 21)}</button>`}
            <button class="c-icon-btn c-bell" data-pp="bell" aria-label="Notifications">${I('bell', 21)}<span class="c-bell__dot"></span></button>
            <button class="c-meb" data-pp="me" aria-label="Settings for ${esc(me.name)}">${VC.avatar(me.id)}</button>
          </div>
        </div>
      </header>
      <main class="c-pmain"><div class="c-pview"></div></main>`;
    const main = $(root, '.c-pmain');
    let viewEl = $(root, '.c-pview');
    let view = null;
    let route = 'home';
    const hist = [];
    let pop = null;
    const toast = (t, s, tone) => VC.toast(root, t, { sub: s, tone });

    function go(to, { back = false } = {}) {
      closePop();
      const [name, id] = (to || 'home').split('/');
      const V = { home: homeView, search: searchView, library: libraryView, requests: requestsView, work: workView, request: requestView }[name];
      if (!V || (name === 'work' && !d.W[id]) || (name === 'request' && !d.requests.find((r) => r.id === id))) return go('home');
      if (!back && route !== to) hist.push(route);
      view?.destroy?.();
      const next = document.createElement('div');
      next.className = 'c-pview';
      viewEl.replaceWith(next);
      viewEl = next;
      route = to;
      view = V(viewEl, id) || {};
      ctx.setView(to);
      main.scrollTop = 0;
      $(root, '[data-back]').hidden = wide || name === 'home';
      $(root, '.c-phead__brand').hidden = !wide && name !== 'home';
      $$(root, '[data-nav]').forEach((a) => a.toggleAttribute('aria-current', a.dataset.nav === name || (name === 'request' && a.dataset.nav === 'requests')));
    }

    function reqCard(r) {
      const w = d.W[r.work];
      const v = VC.reqView(r);
      return `<a class="c-rqcard" href="#" data-go="request/${r.id}" data-card="${r.id}">
        ${VC.poster(w, 'c-rqcard__p')}
        <span class="c-rqcard__b">
          <span class="c-rqcard__h"><b class="c-trunc">${esc(w.title)}</b><span class="c-rqcard__k" data-k="${v.key}" data-f="title">${esc(v.title)}</span></span>
          ${strip(r)}
          <span class="c-rqcard__l" data-f="line">${esc(v.line)}</span>
        </span>
      </a>`;
    }
    const posterCard = (w, sub) => `<a class="c-pposter" href="#" data-go="work/${w.id}">${VC.poster(w)}<b class="c-trunc">${esc(w.title)}</b><span class="c-trunc">${esc(sub ?? (w.type === 'movie' ? String(w.year) : F.plural(w.seasons.length, 'season')))}</span></a>`;
    const badgeFor = (w) => {
      const r = reqFor(w);
      if (r && r.state !== 'available' && r.state !== 'denied') return VC.reqView(r).title;
      if (VC.journey(w)) return 'On its way';
      if (w.type === 'movie' ? ['available', 'upgradable'].includes(w.status) : w.counts.available) return 'In the library';
      return '';
    };
    function liveStrips() {
      for (const el of $$(viewEl, '.c-rq')) {
        const r = d.requests.find((x) => x.id === el.dataset.rq);
        if (!r) continue;
        const v = VC.reqView(r);
        if (el.dataset.k !== v.key) { el.outerHTML = strip(r, el.classList.contains('c-rq--big')); continue; }
        if (v.key === 'moving') el.style.setProperty('--p', v.pos.toFixed(4));
      }
    }
    function liveText() {
      for (const c of $$(viewEl, '[data-card]')) {
        const r = d.requests.find((x) => x.id === c.dataset.card);
        const v = VC.reqView(r);
        VC.set($(c, '[data-f="line"]'), v.line);
        const t = $(c, '[data-f="title"]');
        VC.set(t, v.title);
        t.dataset.k = v.key;
      }
    }

    function homeView(el) {
      const mine = d.requests.filter((r) => r.by === d.member && r.state !== 'denied');
      const hour = VC.clock().getHours();
      const greet = hour < 5 || hour >= 18 ? 'Good evening' : hour < 12 ? 'Good morning' : 'Good afternoon';
      const moving = d.requests.filter((r) => r.by !== d.member && VC.journey(d.W[r.work]) && r.state !== 'available');
      const arrivals = [...new Set(d.history.filter((h) => h.kind === 'imported' && !h.episode).map((h) => h.work))].map((id) => d.W[id]).concat(d.works.filter((w) => w.type === 'movie' && w.status === 'available').sort((a, b) => b.added - a.added)).filter((w, i, a) => a.indexOf(w) === i).slice(0, wide ? 7 : 6);
      const coming = d.calendar.filter((c) => c.at > VC.clock()).slice(0, 4);
      const inFlight = mine.filter((r) => r.state !== 'available').length;
      el.innerHTML = `<div class="c-pwrap">
        <section class="c-phero"><h1>${greet}, ${esc(first(me.id))}</h1><p>${inFlight ? `${F.plural(inFlight, 'request')} in progress.` : 'Nothing waiting on you.'} ${arrivals.length ? `${esc(arrivals[0].title)} arrived ${esc(F.ago(d.history.find((h) => h.work === arrivals[0].id)?.at || arrivals[0].added))}.` : ''}</p></section>
        <section class="c-psec"><header class="c-psec__h"><h2>Your requests</h2><a href="#" class="c-link" data-go="requests">All requests</a></header>
          ${mine.length ? `<div class="c-rqcards">${mine.map(reqCard).join('')}</div>` : '<p class="c-empty">Find a movie or series and ask for it. You will see it travel here until it is ready to watch.</p>'}
        </section>
        ${moving.map((r) => {
          const w = d.W[r.work];
          const v = VC.reqView(r);
          return `<section class="c-psec"><header class="c-psec__h"><h2>On its way to the library</h2></header>
            <a class="c-feature" href="#" data-go="request/${r.id}" data-card="${r.id}">
              <img class="c-feature__bg" src="${SS.art.backdrop(w, { w: 1200, h: 520 })}" alt="">
              <span class="c-feature__scrim"></span>
              <span class="c-feature__b">
                <span class="c-feature__h"><b>${esc(w.title)}</b><span class="c-rqcard__k" data-k="${v.key}" data-f="title">${esc(v.title)}</span></span>
                <span class="c-feature__s">${esc(VC.workMeta(w))} · requested by ${esc(first(r.by))}</span>
                ${strip(r, true)}
                <span class="c-feature__l" data-f="line">${esc(v.line)}</span>
              </span>
            </a></section>`;
        }).join('')}
        <section class="c-psec"><header class="c-psec__h"><h2>New in the library</h2><a href="#" class="c-link" data-go="library">Browse</a></header>
          <div class="c-prow">${arrivals.map((w) => posterCard(w)).join('')}</div></section>
        <section class="c-psec"><header class="c-psec__h"><h2>Coming soon</h2></header>
          <ul class="c-coming">${coming.map((c) => {
            const w = d.W[c.work];
            return `<li><a href="#" data-go="work/${w.id}"><img src="${SS.art.still(w, c.episode ? c.episode[1] : 2)}" alt="" loading="lazy"><span class="c-coming__b"><b class="c-trunc">${esc(w.title)}</b><span class="c-trunc">${c.episode ? `${F.ep(c.episode)} · ${esc(c.title)}` : esc(c.title)}</span></span><time class="c-num">${esc(F.until(c.at))}</time></a></li>`;
          }).join('')}</ul></section>
      </div>`;
      return { tick: liveStrips, second: liveText, requests: () => go(route, { back: true }) };
    }

    function requestsView(el) {
      const mine = d.requests.filter((r) => r.by === d.member);
      el.innerHTML = `<div class="c-pwrap c-pwrap--narrow">
        <header class="c-ptitle"><h1>Your requests</h1><p>${F.plural(mine.length, 'request')}. You get a notification when each one is ready.</p></header>
        ${mine.length ? `<div class="c-rqcards c-rqcards--list">${mine.map(reqCard).join('')}</div>` : '<p class="c-empty">No requests yet. Search for a movie or series and press Request.</p>'}
      </div>`;
      return { tick: liveStrips, second: liveText, requests: () => go(route, { back: true }) };
    }

    function requestView(el, id) {
      const r = d.requests.find((x) => x.id === id);
      const w = d.W[r.work];
      const render = () => {
        const v = VC.reqView(r);
        el.innerHTML = `<div class="c-pwrap c-pwrap--narrow c-preq" data-card="${r.id}">
          ${wide ? `<a href="#" class="c-link c-preq__back" data-go="requests">${I('chevron-left', 16)}Requests</a>` : ''}
          <div class="c-preq__head">${VC.poster(w, 'c-preq__p')}<div><p class="c-dim">${esc(VC.workMeta(w))}</p><h1>${esc(w.title)}</h1><p class="c-dim">Requested by ${r.by === d.member ? 'you' : esc(first(r.by))} ${esc(F.ago(r.at))}${r.seasons ? ` · Season ${r.seasons.join(', ')}` : ''}</p></div></div>
          <section class="c-preq__card">
            <h2 class="c-preq__t" data-f="title" data-k="${v.key}">${esc(v.title)}</h2>
            <p class="c-preq__l" data-f="line">${esc(v.line)}</p>
            ${strip(r, true)}
          </section>
          <p class="c-preq__foot">${v.key === 'arrived' ? `Open Plex on any device to watch ${esc(w.title)}.` : v.key === 'denied' ? 'You can ask Alex about it, or request something else.' : `We will notify ${r.by === d.member ? 'you' : esc(first(r.by))} the moment it is ready to watch.`}</p>
          ${v.key === 'arrived' ? `<button class="c-btn c-btn--primary c-btn--l" data-pa="watch">${I('play', 17)}Watch in Plex</button>` : `<a href="#" class="c-btn c-btn--l" data-go="work/${w.id}">About ${esc(w.title)}</a>`}
        </div>`;
      };
      render();
      return {
        tick: liveStrips,
        second() {
          const v = VC.reqView(r);
          const t = $(el, '[data-f="title"]');
          if (t.dataset.k !== v.key) return render();
          VC.set($(el, '[data-f="line"]'), v.line);
          const tm = $(el, '[data-t="2"]');
          if (tm && v.key === 'moving') VC.set(tm, v.line.replace('Ready in ', 'in '));
        },
        requests: render,
      };
    }

    function workView(el, id) {
      const w = d.W[id];
      let picking = false;
      const inLib = (s) => s.episodes.every((e) => e.status === 'available' || e.status === 'unreleased');
      let chosen = w.type === 'series' ? w.seasons.filter((s) => !inLib(s)).map((s) => s.n) : [];
      const render = () => {
        const r = reqFor(w);
        const live = r && r.state !== 'denied' && r.state !== 'available';
        const has = w.type === 'movie' ? ['available', 'upgradable'].includes(w.status) : w.counts.missing === 0;
        let cta;
        if (live) cta = `<div class="c-pwork__req" data-card="${r.id}"><span class="c-rqcard__h"><b data-f="title" class="c-rqcard__k" data-k="${VC.reqView(r).key}">${esc(VC.reqView(r).title)}</b><a href="#" class="c-link" data-go="request/${r.id}">Follow the request ${I('arrow-right', 14)}</a></span>${strip(r)}<span class="c-rqcard__l" data-f="line">${esc(VC.reqView(r).line)}</span></div>`;
        else if (has) cta = `<button class="c-btn c-btn--primary c-btn--l" data-pa="watch">${I('play', 17)}Watch in Plex</button><span class="c-pwork__note">In the library</span>`;
        else if (w.type === 'series' && picking) cta = `<div class="c-pick"><p class="c-pick__h">Which seasons?</p><ul>${w.seasons.map((s) => {
          const lib = inLib(s);
          const avail = s.episodes.filter((e) => e.status === 'available').length;
          return `<li><label class="c-pick__row${lib ? ' is-off' : ''}"><input type="checkbox" data-season="${s.n}"${chosen.includes(s.n) ? ' checked' : ''}${lib ? ' disabled' : ''}><span class="c-pick__box">${I('check', 14)}</span><span><b>Season ${s.n}</b><span class="c-dim">${s.episodes.length} episodes · ${lib ? 'In the library' : avail ? `${avail} in the library` : s.year}</span></span></label></li>`;
        }).join('')}</ul><div class="c-pick__acts"><button class="c-btn c-btn--l" data-pa="cancel">Cancel</button><button class="c-btn c-btn--primary c-btn--l" data-pa="send"${chosen.length ? '' : ' disabled'}>Request ${chosen.length ? F.plural(chosen.length, 'season') : ''}</button></div></div>`;
        else cta = `<button class="c-btn c-btn--primary c-btn--l" data-pa="request">${I('plus', 17)}Request</button><span class="c-pwork__note">${w.status === 'unreleased' ? `Out ${esc(F.until(w.release || new Date(w.year, 0, 1)))}` : 'Alex approves requests, usually within a day'}</span>`;
        el.innerHTML = `<article class="c-pwork">
          <div class="c-pwork__hero"><img src="${SS.art.backdrop(w, { w: 1600, h: 700 })}" alt=""><span class="c-pwork__scrim"></span></div>
          <div class="c-pwrap c-pwork__in">
            <div class="c-pwork__head">${VC.poster(w, 'c-pwork__p')}<div class="c-pwork__id"><h1>${esc(w.title)}</h1><p class="c-pwork__meta">${esc(VC.workMeta(w))} · ${esc(w.genres.join(', '))}</p><p class="c-pwork__tag">${esc(w.tagline)}</p></div></div>
            <div class="c-pwork__cta">${cta}</div>
            <p class="c-pwork__ov">${esc(w.overview)}</p>
            <dl class="c-pwork__facts"><dt>${w.type === 'movie' ? 'Director' : 'Created by'}</dt><dd>${esc(w.director || w.creator)}</dd><dt>Starring</dt><dd>${esc(w.cast.slice(0, 3).join(', '))}</dd>${w.type === 'series' ? `<dt>Seasons</dt><dd>${w.seasons.map((s) => `Season ${s.n}: ${inLib(s) ? 'in the library' : `${s.episodes.filter((e) => e.status === 'available').length} of ${s.episodes.length}`}`).join(' · ')}</dd>` : `<dt>Runtime</dt><dd>${esc(F.runtime(w.runtime))}</dd>`}</dl>
          </div>
        </article>`;
      };
      render();
      el.addEventListener('change', (e) => {
        const c = e.target.closest('[data-season]');
        if (!c) return;
        const n = +c.dataset.season;
        chosen = c.checked ? [...chosen, n].sort() : chosen.filter((x) => x !== n);
        const b = $(el, '[data-pa="send"]');
        b.disabled = !chosen.length;
        b.textContent = `Request ${chosen.length ? F.plural(chosen.length, 'season') : ''}`;
      });
      return {
        tick: liveStrips,
        second: liveText,
        act(a) {
          if (a === 'request' && w.type === 'series') { picking = true; render(); return; }
          if (a === 'cancel') { picking = false; render(); return; }
          if (a === 'request' || a === 'send') {
            const r = VC.act.request(w.id, w.type === 'series' ? chosen : undefined);
            picking = false;
            toast(`Requested ${w.title}`, 'Alex will see it in the queue');
            render();
            return r;
          }
        },
        requests: render,
      };
    }

    function libraryView(el) {
      let type = 'movie';
      const render = () => {
        const ws = d.works.filter((w) => w.type === type && (type === 'movie' ? ['available', 'upgradable'].includes(w.status) : w.counts.available)).sort((a, b) => b.added - a.added);
        el.innerHTML = `<div class="c-pwrap"><header class="c-ptitle c-ptitle--row"><h1>Library</h1><div class="c-seg" role="tablist"><button role="tab" aria-selected="${type === 'movie'}" data-type="movie">Movies</button><button role="tab" aria-selected="${type === 'series'}" data-type="series">Series</button></div></header>
          <div class="c-pgridp">${ws.map((w) => posterCard(w)).join('')}</div></div>`;
      };
      render();
      el.addEventListener('click', (e) => { const t = e.target.closest('[data-type]'); if (t) { type = t.dataset.type; render(); } });
      return {};
    }
    function searchView(el) {
      el.innerHTML = `<div class="c-pwrap"><header class="c-ptitle"><h1>Search</h1></header>
        <label class="c-pfield">${I('search', 19)}<input type="search" placeholder="Movies and series" enterkeyhint="search" autocomplete="off" autocapitalize="none" aria-label="Search movies and series"></label>
        <div data-res></div></div>`;
      const input = $(el, 'input');
      const res = $(el, '[data-res]');
      const render = () => {
        const q = input.value;
        const list = q ? VC.searchWorks(q) : d.works.filter((w) => !badgeFor(w) || w.status === 'unreleased').slice(0, 12);
        res.innerHTML = q && !list.length ? `<p class="c-empty">Nothing matches “${esc(q)}”. Try part of a title, or a genre such as drama.</p>`
          : `<h2 class="c-psec__t">${q ? `${F.plural(list.length, 'result')}` : 'Not in the library yet'}</h2><div class="c-pgridp">${list.map((w) => { const b = badgeFor(w); return `<a class="c-pposter" href="#" data-go="work/${w.id}">${VC.poster(w)}${b ? `<span class="c-pbadge" data-b="${b === 'In the library' ? 'lib' : 'req'}">${esc(b)}</span>` : ''}<b class="c-trunc">${esc(w.title)}</b><span>${w.year} · ${w.type === 'movie' ? 'Movie' : 'Series'}</span></a>`; }).join('')}</div>`;
      };
      input.addEventListener('input', render);
      render();
      if (wide) input.focus();
      return {};
    }

    function openPop(kind, anchor) {
      if (pop?.kind === kind) return closePop();
      closePop();
      const el = document.createElement('div');
      el.className = 'c-ppop';
      const mine = d.requests.filter((r) => r.by === d.member);
      if (kind === 'bell') {
        const items = [
          ...mine.filter((r) => r.state === 'available').map((r) => ({ t: `${d.W[r.work].title} is ready to watch`, at: r.doneAt })),
          ...mine.filter((r) => r.state === 'approved' || r.state === 'downloading').map((r) => ({ t: `Alex approved ${d.W[r.work].title}`, at: r.decidedAt || new Date(r.at.getTime() + 52 * 60e3) })),
          ...mine.filter((r) => r.state === 'denied').map((r) => ({ t: `${d.W[r.work].title} was not added`, at: r.decidedAt || r.at })),
          ...d.history.filter((h) => h.kind === 'imported' && !h.episode).slice(0, 2).map((h) => ({ t: `${d.W[h.work].title} is new in the library`, at: h.at })),
        ].sort((a, b) => b.at - a.at);
        el.innerHTML = `<header class="c-ppop__h">Notifications</header><ul>${items.map((x) => `<li><span class="c-ppop__dot"></span><span><b>${esc(x.t)}</b><time>${esc(F.ago(x.at))}</time></span></li>`).join('')}</ul>`;
      } else {
        el.innerHTML = `<header class="c-ppop__h c-ppop__me">${VC.avatar(me.id)}<span><b>${esc(me.name)}</b><span>${esc(me.email)}</span></span></header>
          <div class="c-ppop__menu">${['Sign-in with PIN and passkey', 'Notifications', 'Requests left this week'].map((x) => `<button data-pa="menu" data-v="${x}">${esc(x)}${x.startsWith('Requests') ? `<span class="c-num">${me.quota.movie[1] - me.quota.movie[0]} movies</span>` : ''}</button>`).join('')}<hr><button data-pa="menu" data-v="Sign out">Sign out</button></div>`;
      }
      root.appendChild(el);
      const r = anchor.getBoundingClientRect(), rr = root.getBoundingClientRect(), k = rr.width / root.offsetWidth || 1;
      el.style.top = `${(r.bottom - rr.top) / k + 8}px`;
      el.style.right = `${Math.max(8, (rr.right - r.right) / k)}px`;
      anchor.setAttribute('aria-expanded', 'true');
      pop = { kind, el, anchor };
    }
    function closePop() {
      if (!pop) return;
      pop.anchor.removeAttribute('aria-expanded');
      const el = pop.el;
      el.classList.add('is-out');
      setTimeout(() => el.remove(), 140);
      pop = null;
    }

    ctx.listen(root, 'click', (e) => {
      const t = e.target;
      if (pop && !t.closest('.c-ppop') && !t.closest('[data-pp]')) closePop();
      const pp = t.closest('[data-pp]');
      if (pp) { e.preventDefault(); return openPop(pp.dataset.pp, pp); }
      if (t.closest('[data-back]')) { const to = hist.pop() || 'home'; return go(to, { back: true }); }
      const pa = t.closest('[data-pa]');
      if (pa && !pa.disabled) {
        e.preventDefault();
        if (pa.dataset.pa === 'watch') return toast('Opening Plex', 'Prototype: playback happens in Plex', 'muted');
        if (pa.dataset.pa === 'menu') { closePop(); return toast(pa.dataset.v, 'Prototype: account settings are not part of this direction', 'muted'); }
        return view?.act?.(pa.dataset.pa);
      }
      const g = t.closest('[data-go]');
      if (g) { e.preventDefault(); go(g.dataset.go); }
    });
    ctx.listen(window, 'keydown', (e) => { if (e.key === 'Escape') closePop(); });
    ctx.on('tick', () => view?.tick?.());
    ctx.on('second', () => view?.second?.());
    ctx.on('request:update', () => view?.requests?.());
    ctx.on('job:done', () => view?.second?.());

    go(ctx.view || 'home', { back: true });
    return { destroy() { closePop(); } };
  };
})();
