// PROTOTYPE. House Lights: components shared by the desk, the phone and the portal.
(function () {
  const SS = window.SS, VB = window.VB;
  const D = SS.data, F = SS.fmt, ART = SS.art;
  const esc = F.esc;
  const icon = VB.icon;

  const bestMet = (w) => {
    const met = VB.coverage(w).filter((x) => x.c === 'met');
    return met.length ? met[0].t.name : '';
  };
  const releaseDay = (w) => (w.release ? w.release.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' }) : String(w.year));

  // Quiet caption under a library tile, revealed on focus. In-progress Works keep a live line.
  VB.caption = (w) => {
    const { s, j } = VB.status(w);
    if (w.type === 'series') {
      const live = j ? `<span class="b-tile__s is-live" data-lv="wline|${w.id}|${VB.epKey(j.episode)}">${esc(VB.line(j))}</span>` : '';
      return live || `<span class="b-tile__s">${F.plural(w.seasons.length, 'season')} · ${w.counts.available} of ${w.counts.aired} episodes</span>`;
    }
    if (j) return `<span class="b-tile__s is-live" data-lv="wline|${w.id}">${esc(VB.line(j))}</span>`;
    if (s === 'available') return `<span class="b-tile__s">${w.year} · ${esc(bestMet(w) || 'Available')}</span>`;
    if (s === 'unreleased') return `<span class="b-tile__s">Out ${esc(releaseDay(w))}</span>`;
    return `<span class="b-tile__s">${w.year} · ${VB.statusLabel[s]}</span>`;
  };

  VB.tile = (w, { vtid } = {}) => {
    const { j } = VB.status(w);
    const live = !!j;
    return `<a class="b-tile${live ? ' is-live' : ''}" href="${VB.href('work/' + w.id)}" data-open="${w.id}" style="${VB.tint(w)}">
      <div class="b-tile__art">${VB.art(w, { vt: vtid })}</div>
      <div class="b-tile__cap"><span class="b-tile__t">${esc(w.title)}</span>${VB.caption(w)}</div>
    </a>`;
  };

  // Every Item with an acquisition job that is not finished, most urgent light first.
  VB.developing = () => {
    const seen = new Set(), out = [];
    for (const j of D.jobs) {
      if (!VB.isAcq(j) || j.state === 'done' || VB.local.resolved.has(j.id)) continue;
      const key = j.work + '|' + VB.epKey(j.episode);
      if (seen.has(key)) continue;
      seen.add(key);
      const best = VB.acq(j.work, j.episode);
      out.push({ w: D.W[j.work], ep: j.episode, j: best, key });
    }
    const rank = { running: 0, stalled: 1, queued: 2, failed: 3 };
    return out.sort((a, b) => rank[a.j.state] - rank[b.j.state] || (VB.readyIn(a.j) || 0) - (VB.readyIn(b.j) || 0));
  };

  VB.devTile = ({ w, ep, j }) => `<a class="b-tile b-dtile" href="${VB.href('work/' + w.id)}" data-open="${w.id}" data-key="${w.id}|${VB.epKey(ep)}" style="${VB.tint(w)}">
      <div class="b-tile__art">${VB.dev(w, { ep })}</div>
      <div class="b-dtile__cap">
        <span class="b-tile__t">${esc(w.title)}${ep ? ` <span class="b-dim">${F.ep(ep)}</span>` : ''}</span>
        <span class="b-dtile__s" data-lv="wline|${w.id}${ep ? '|' + VB.epKey(ep) : ''}">${esc(VB.line(j))}</span>
      </div>
    </a>`;

  VB.aisleList = (items, { cls = '' } = {}) => {
    if (!items.length) return `<p class="b-empty b-empty--quiet">Nothing needs you. Every request is answered and every job is moving.</p>`;
    return `<ul class="b-aisle ${cls}" role="list">${items.map((a) => `<li class="b-aisle__i" data-tone="${a.tone}" data-aisle="${a.id}">
        <span class="b-dot b-dot--${a.tone}" aria-hidden="true"></span>
        <div class="b-aisle__txt"><span class="b-aisle__t">${esc(a.title)}</span><span class="b-aisle__s">${esc(a.sub)}</span></div>
        <button class="b-btn b-btn--quiet b-btn--sm" type="button" data-aisle-act="${a.id}">${esc(a.act)}</button>
      </li>`).join('')}</ul>`;
  };

  VB.comingSoon = (n = 6) => D.calendar.filter((c) => c.at > SS.sim.clock).slice(0, n);
  VB.comingCard = (c) => {
    const w = D.W[c.work];
    const label = c.episode ? `${F.ep(c.episode)} · ${c.title}` : c.title;
    return `<a class="b-cs" href="${VB.href('work/' + w.id)}" data-open="${w.id}" style="${VB.tint(w)}">
      <div class="b-cs__art"><img alt="" data-still="${w.id}|${c.episode ? c.episode[1] % 5 + 1 : 2}" src=""></div>
      <div class="b-cs__cap"><span class="b-cs__t">${esc(w.title)}</span><span class="b-cs__s">${esc(label)}</span><span class="b-cs__w">${esc(cap(F.until(c.at)))}</span></div>
    </a>`;
  };
  const cap = (s) => s.charAt(0).toUpperCase() + s.slice(1);
  VB.cap = cap;

  // Stills are drawn on demand; fill them a couple per frame so a list never stalls the page.
  VB.fillStills = (root) => {
    const imgs = [...root.querySelectorAll('img[data-still]')];
    const step = () => {
      for (let k = 0; k < 2 && imgs.length; k++) {
        const el = imgs.shift();
        if (!el.isConnected) continue;
        const [id, n] = el.dataset.still.split('|');
        el.src = ART.still(id, +n);
        el.removeAttribute('data-still');
      }
      if (imgs.length) requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
  };

  VB.nodesLine = () => {
    const ok = D.nodes.filter((n) => n.state === 'online').map((n) => n.name);
    const deg = D.nodes.filter((n) => n.state === 'degraded');
    const off = D.nodes.filter((n) => n.state === 'offline');
    const parts = [`<span>${esc(F.list(ok))} are running well.</span>`];
    for (const n of deg) parts.push(`<span><i class="b-dot b-dot--ink" aria-hidden="true"></i>${esc(n.name)} is 2 h behind on transcodes.</span>`);
    for (const n of off) parts.push(`<span><i class="b-dot b-dot--exit" aria-hidden="true"></i>${esc(n.name)} has been offline since ${F.time(n.lastSeen)}.</span>`);
    return parts.join(' ');
  };

  // ---------- Targets and Editions ----------
  const CELL = {
    met: ['circle-check', 'Met'], upgradable: ['circle-arrow-up', 'Upgradable'], missing: ['circle-dashed', 'Missing'],
    downloading: ['arrow-down', 'Downloading'], deriving: ['split', 'Deriving'], waiting: ['split', 'Derives after download'],
    stalled: ['pause', 'Stalled'], failed: ['circle-x', 'Failed'], queued: ['clock', 'Queued'], unreleased: ['calendar', 'Not out yet'],
    unwanted: ['minus', 'Not in Profile'],
  };
  const cellIcon = (c) => ({
    met: icon('circle-check', 16), upgradable: icon('circle-arrow-up', 16), missing: icon('circle-dashed', 16), downloading: icon('arrow-down', 16),
    deriving: icon('split', 16), waiting: icon('split', 16), stalled: icon('pause', 16), failed: icon('circle-x', 16), queued: icon('clock', 16),
    unreleased: icon('calendar', 16), unwanted: icon('minus', 16),
  }[c]);
  VB.files = (w) => {
    const files = (w.files || []).slice();
    if (w.type !== 'movie') return files;
    for (const t of D.targets) {
      if (w.coverage[t.id] === 'met' && !files.some((f) => f.target === t.id)) {
        const dl = D.jobs.find((j) => j.work === w.id && j.kind === 'download' && j.target === t.id);
        files.push({ target: t.id, size: dl?.size || 40e9, name: dl?.release || `${w.title.replace(/[^A-Za-z0-9]+/g, '.')}.${w.year}.${t.id === 'uhd' ? '2160p.HDR' : '1080p'}`, video: t.id === 'uhd' ? 'HEVC · Dolby Vision P7 + HDR10' : 'AVC · SDR', audio: t.id === 'uhd' ? 'TrueHD Atmos 7.1' : 'DTS-HD MA 5.1', resolution: t.id === 'uhd' ? '3840×2160' : '1920×1080', node: 'vault', fresh: true });
      }
    }
    return files;
  };
  VB.covMatrix = (w) => {
    const cov = VB.coverage(w);
    const files = VB.files(w);
    const eds = w.editions;
    const head = cov.map(({ t }) => `<div class="b-cov__th" role="columnheader"><span>${esc(t.name)}</span><small>${t.kind === 'derived' ? `Derived from ${esc(D.targets.find((x) => x.id === t.from).name)}` : 'Downloaded'}</small></div>`).join('');
    const detail = (x) => {
      const f = files.find((ff) => ff.target === x.t.id);
      if (x.c === 'met') return f ? F.bytes(f.size) : '';
      if (x.c === 'downloading' || x.c === 'deriving') return `<span data-lv="jpct|${x.j.id}">${Math.floor(x.j.progress * 100)}%</span>`;
      if (x.c === 'upgradable') return 'Below the Blu-ray cutoff';
      if (x.c === 'waiting') return x.t.from ? `After ${esc(D.targets.find((t) => t.id === x.t.from).name)} arrives` : 'After the download';
      if (x.c === 'queued') return `On ${esc(x.j.node)}`;
      if (x.c === 'failed') return 'Sample release';
      if (x.c === 'stalled') return 'No seeders';
      if (x.c === 'missing') return 'Wanted';
      return '';
    };
    const rows = eds.map((ed, i) => {
      const cells = cov.map((x) => {
        let c = x.c;
        if (i > 0) c = c === 'met' || c === 'upgradable' || c === 'downloading' ? 'missing' : c;
        const y = i > 0 && c !== x.c ? { ...x, c, j: null } : x;
        return `<div class="b-cov__cell" data-c="${c}" role="cell">${cellIcon(c)}<span class="b-cov__l">${CELL[c][1]}</span><small>${detail(y)}</small></div>`;
      }).join('');
      return `<div class="b-cov__ed" role="rowheader">${esc(ed)}</div>${cells}`;
    }).join('');
    return `<div class="b-cov" role="table" style="--cols:${cov.length}"><div class="b-cov__corner" role="columnheader">Edition</div>${head}${rows}</div>`;
  };
  VB.fileList = (w) => {
    const files = VB.files(w);
    if (!files.length) return `<p class="b-empty">No files yet. When a release meets one of this Work's Targets it is imported here, and the Targets above turn to Met.</p>`;
    return `<ul class="b-files" role="list">${files.map((f) => `<li class="b-file">
      <span class="b-chip">${esc(D.targets.find((t) => t.id === f.target).short)}</span>
      <div class="b-file__main"><span class="b-file__n">${esc(f.name)}</span><span class="b-file__m">${esc(f.video)} · ${esc(f.audio)} · ${esc(f.resolution)}${f.derived ? ' · derived on forge' : ''}</span></div>
      <span class="b-file__s">${F.bytes(f.size)}</span><span class="b-file__node">${esc(f.node)}</span>
    </li>`).join('')}</ul>`;
  };

  // ---------- Series ----------
  VB.epStatus = (w, s, e) => {
    const j = VB.acq(w.id, [s.n, e.n]);
    if (j) return { st: j.state === 'failed' ? 'failed' : 'downloading', j };
    return { st: e.status, j: null };
  };
  VB.episodes = (w, sn) => {
    const s = w.seasons[sn - 1];
    return `<ol class="b-eps" role="list">${s.episodes.map((e) => {
      const { st, j } = VB.epStatus(w, s, e);
      const art = j ? VB.dev(w, { ep: [s.n, e.n], still: (e.n % 5) + 1 }) : `<img alt="" src="" data-still="${w.id}|${(e.n % 5) + 1}">`;
      const when = e.air > SS.sim.clock ? `Airs ${F.until(e.air)}` : `Aired ${e.air.toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })}`;
      const right = {
        available: `${icon('circle-check', 16)}<span>Available</span>`,
        downloading: j ? `<span data-lv="wline|${w.id}|${s.n}x${e.n}">${esc(VB.line(j))}</span>` : '',
        missing: `${icon('circle-dashed', 16)}<span>Missing</span>`,
        failed: `${icon('circle-x', 16)}<span>Failed</span>`,
        unreleased: `${icon('calendar', 16)}<span>${esc(cap(F.until(e.air)))}</span>`,
      }[st] || '';
      return `<li class="b-ep" data-st="${st}">
        <div class="b-ep__art${e.air > SS.sim.clock ? ' is-future' : ''}">${art}</div>
        <div class="b-ep__main"><span class="b-ep__t"><span class="b-ep__n">${e.n}</span>${esc(e.title)}</span><span class="b-ep__m">${esc(when)} · ${e.runtime} min</span></div>
        <div class="b-ep__st">${right}</div>
        ${st === 'missing' || st === 'failed' ? `<button class="b-btn b-btn--quiet b-btn--sm" type="button" data-cmd="search-ep" data-ep="${s.n}x${e.n}">Search</button>` : '<span></span>'}
      </li>`;
    }).join('')}</ol>`;
  };

  // ---------- Activity ----------
  const KIND = { download: 'Download', transfer: 'Transfer', import: 'Import', transcode: 'Derive', subtitles: 'Subtitles' };
  VB.jobRow = (j) => {
    const w = D.W[j.work];
    const tname = D.targets.find((t) => t.id === j.target)?.name || '';
    const where = j.kind === 'transfer' ? `${j.from} → ${j.to}` : j.node;
    const sub = j.release || (j.kind === 'transcode' ? `${D.targets.find((t) => t.id === j.from).name} to ${tname} · ${j.encoder || 'NVENC HEVC'}` : j.step || '');
    const reason = j.state === 'failed' || j.state === 'stalled' ? `<span class="b-job__why">${esc(j.message)}</span>` : '';
    const act = j.state === 'failed' ? `<button class="b-btn b-btn--sm" type="button" data-cmd="blocklist" data-job="${j.id}">Blocklist and search</button>`
      : j.state === 'stalled' ? `<button class="b-btn b-btn--sm" type="button" data-cmd="another" data-job="${j.id}">Find another release</button>`
      : `<button class="b-iconbtn b-iconbtn--sm" type="button" data-cmd="jobmenu" data-job="${j.id}" aria-label="More for ${esc(j.label)}">${icon('ellipsis', 16)}</button>`;
    return `<div class="b-job" data-st="${j.state}" data-job-row="${j.id}" role="row" style="${VB.tint(w)}">
      <a class="b-job__art" href="${VB.href('work/' + w.id)}" data-open="${w.id}" aria-label="${esc(w.title)}">${VB.dev(w, { job: j })}</a>
      <div class="b-job__main" role="cell">
        <span class="b-job__t">${esc(j.label)}</span>
        <span class="b-job__k">${KIND[j.kind]}${tname ? ' · ' + esc(tname) : ''}${reason ? '' : ''}</span>
        ${reason || `<span class="b-job__r">${esc(sub)}</span>`}
      </div>
      <span class="b-job__node" role="cell">${esc(where)}</span>
      <span class="b-job__n" role="cell" data-lv="jpct|${j.id}">${j.state === 'queued' ? '0%' : Math.floor(j.progress * 100) + '%'}</span>
      <span class="b-job__n b-job__rate" role="cell" data-lv="jrate|${j.id}">${esc(VB.LV.jrate(j.id))}</span>
      <span class="b-job__n b-job__eta" role="cell" data-lv="jeta|${j.id}">${esc(VB.LV.jeta(j.id))}</span>
      <span class="b-job__act" role="cell">${act}</span>
    </div>`;
  };

  // ---------- Requests ----------
  VB.seasonsLabel = (w, ss) => (ss.length > 1 && ss.length === w.seasons.length ? 'All seasons' : `${ss.length > 1 ? 'Seasons' : 'Season'} ${F.list(ss.map(String))}`);
  VB.reqRow = (r) => {
    const w = D.W[r.work];
    const st = VB.reqState(r);
    const seasons = r.seasons ? `<span class="b-req__seasons">${VB.seasonsLabel(w, r.seasons)}</span>` : '';
    const acts = st === 'pending'
      ? `<button class="b-btn b-btn--quiet" type="button" data-req-deny="${r.id}">Deny</button><button class="b-btn b-btn--primary" type="button" data-req-approve="${r.id}">Approve</button>`
      : st === 'downloading' ? `<span class="b-req__live" data-lv="wline|${w.id}">${esc(VB.line(VB.acq(w.id)))}</span>`
      : st === 'available' ? `<span class="b-req__done">${icon('circle-check', 16)}Available ${r.doneAt ? esc(F.ago(r.doneAt)) : ''}</span>`
      : st === 'approved' ? `<span class="b-req__done">${icon('search', 16)}Approved, searching</span>`
      : `<span class="b-req__done">${icon('circle-x', 16)}Denied</span>`;
    return `<article class="b-req" data-state="${st}" data-req="${r.id}" style="${VB.tint(w)}">
      <a class="b-req__art" href="${VB.href('work/' + w.id)}" data-open="${w.id}" aria-label="${esc(w.title)}">${VB.art(w)}</a>
      <div class="b-req__main">
        <h3 class="b-req__t">${esc(w.title)} <span class="b-dim">${w.year}</span>${seasons}</h3>
        <p class="b-req__by">${VB.av(r.by)}<span>${esc(VB.name(r.by))} · ${esc(F.ago(r.at))}</span></p>
        ${r.note ? `<p class="b-req__note">“${esc(r.note)}”</p>` : ''}
        ${st === 'denied' && r.reason ? `<p class="b-req__note">${esc(r.reason)}</p>` : ''}
        <div class="b-req__deny" hidden><label class="b-field"><span class="b-sr">Reason for ${esc(VB.first(r.by))}</span><input class="b-input" type="text" placeholder="Tell ${esc(VB.first(r.by))} why, optional"></label><button class="b-btn b-btn--quiet" type="button" data-req-cancel="${r.id}">Keep it</button><button class="b-btn" type="button" data-req-confirm="${r.id}">Deny request</button></div>
      </div>
      <div class="b-req__act">${acts}</div>
    </article>`;
  };

  // ---------- Ticket stub: a Member's request ----------
  const STEP = { pending: 1, approved: 2, downloading: 3, available: 4, denied: 0 };
  VB.ticket = (r, { big = false, link = true, title = true, titleW } = {}) => {
    const w = D.W[r.work];
    const st = VB.reqState(r);
    const step = STEP[st];
    const j = st === 'downloading' ? VB.acq(w.id) : null;
    const mine = r.by === D.member;
    const who = mine ? `You asked ${F.ago(r.at)}` : `${VB.first(r.by)} asked ${F.ago(r.at)}`;
    const line = {
      pending: 'Waiting for Alex to approve',
      approved: 'Approved. It starts as soon as a copy turns up',
      downloading: VB.memberLine(j),
      available: `Arrived ${r.doneAt ? F.ago(r.doneAt) : 'today'}`,
      denied: r.reason || 'Not this time',
    }[st];
    const seg = ['Requested', 'Approved', 'Developing'].map((s, i) => {
      const on = i < step - 1 || (i === step - 1 && st !== 'downloading') || st === 'available';
      const cur = i === step - 1 && st !== 'available';
      return `<li class="b-tseg" data-on="${on}" data-cur="${cur}"><span class="b-tseg__bar">${cur && st === 'downloading' ? `<i data-lm="wfrac|${w.id}" style="transform:scaleX(${VB.frac(j).toFixed(3)})"></i>` : ''}</span><span class="b-tseg__l">${s}</span></li>`;
    }).join('');
    const art = st === 'downloading' ? VB.dev(w) : st === 'available' ? VB.poster(w) : `<div class="b-dev" data-st="queued"><img class="b-dev__m" src="${ART.poster(w)}" alt=""></div>`;
    const Tag = link ? 'a' : 'div';
    return `<article class="b-ticket${big ? ' is-big' : ''}" data-state="${st}" data-ticket="${r.id}" style="${VB.tint(w)}">
      <${Tag} class="b-ticket__main"${link ? ` href="${VB.href('request/' + r.id)}" data-go="request/${r.id}"` : ''}>
        <div class="b-ticket__art">${art}</div>
        <div class="b-ticket__head">
          ${title ? VB.logo(w, { tag: 'h3', maxW: titleW || (big ? 400 : 200), max: big ? 46 : 26, min: big ? 20 : 15, cls: 'b-ticket__title' }) : ''}
          <p class="b-ticket__who">${esc(who)}${r.seasons ? ` · ${VB.seasonsLabel(w, r.seasons).toLowerCase()}` : ''}</p>
        </div>
        <ol class="b-tsegs" role="list">${seg}</ol>
        <p class="b-ticket__line"${st === 'downloading' ? ` data-lv="wmember|${w.id}"` : ''}>${esc(line)}</p>
      </${Tag}>
      <div class="b-ticket__stub">
        <div class="b-stub__face b-stub__wait"><span class="b-stub__k">Showing</span><span class="b-stub__v">${st === 'denied' ? 'Cancelled' : st === 'downloading' ? 'Soon' : 'Not yet'}</span></div>
        <div class="b-stub__face b-stub__ready"><span class="b-stub__k">Ready to watch</span><button class="b-btn b-btn--primary b-btn--sm" type="button" data-cmd="play" data-work="${w.id}">${icon('play', 14)}Play</button></div>
      </div>
    </article>`;
  };
  // Tear the stub when a request turns available while it is on screen.
  VB.tear = (el) => {
    const stub = el.querySelector('.b-ticket__stub');
    if (!stub || VB.reduced()) return;
    const s = VB.spring({ response: 0.55, damping: 0.62 });
    stub.animate([{ transform: 'translate(0,0) rotate(0deg)' }, { transform: 'translate(7px, 4px) rotate(4deg)' }], { duration: s.duration, easing: s.easing });
  };

  // ---------- Nodes ----------
  const STATE = { online: 'Online', degraded: 'Behind', offline: 'Offline' };
  VB.nodeRow = (n) => {
    const groups = D.roleGroups.map((g) => ({ g, rs: g.roles.filter((r) => n.roles.includes(r)) })).filter((x) => x.rs.length);
    const meter = (k, label, v) => `<div class="b-meter"><span class="b-meter__l">${label}</span><span class="b-meter__t"><i data-lm="${k}|${n.id}" style="transform:scaleX(${v.toFixed(3)})"></i></span><span class="b-meter__v" data-lv="${k}|${n.id}">${Math.floor(v * 100)}%</span></div>`;
    const disks = n.disks.map((d) => `<div class="b-meter"><span class="b-meter__l">${esc(d.path)}</span><span class="b-meter__t"><i style="transform:scaleX(${(d.used / d.total).toFixed(3)})"></i></span><span class="b-meter__v">${F.bytes(d.total - d.used)} free</span></div>`).join('');
    const tone = n.state === 'offline' ? 'exit' : n.state === 'degraded' ? 'amber' : 'ok';
    return `<article class="b-node" data-state="${n.state}" data-node="${n.id}">
      <header class="b-node__h">
        <h3 class="b-node__n">${esc(n.name)}${n.controller ? '<span class="b-tag">Controller</span>' : ''}</h3>
        <span class="b-node__st"><i class="b-dot b-dot--${tone}" aria-hidden="true"></i>${STATE[n.state]}</span>
        <p class="b-node__hw">${esc(n.hardware)} · ${esc(n.place)} · <span class="b-nowrap">${esc(n.address)}</span></p>
        ${n.stateNote ? `<p class="b-node__note">${esc(n.stateNote)}</p>` : ''}
      </header>
      <div class="b-node__load">
        ${n.state === 'offline' ? '<p class="b-dim">No reports since it went offline.</p>' : meter('ncpu', 'CPU', n.cpu) + meter('nmem', 'Memory', n.mem) + (n.gpu != null ? meter('ngpu', 'GPU', n.gpu) : '') + (n.state !== 'offline' ? `<div class="b-node__net"><span>${icon('arrow-down', 14)}<b data-lv="ndown|${n.id}">${F.rate(n.down)}</b></span><span>${icon('arrow-up', 14)}<b data-lv="nup|${n.id}">${F.rate(n.up)}</b></span>${n.temp ? `<span>${n.temp} °C</span>` : ''}</div>` : '')}
      </div>
      <div class="b-node__disks">${disks}</div>
      <div class="b-node__roles">${groups.map((x) => `<div class="b-rg"><span class="b-rg__g">${esc(x.g.name)}</span><span class="b-rg__r">${x.rs.map(esc).join(', ')}</span></div>`).join('')}</div>
    </article>`;
  };

  // ---------- Manual search (inline, never a modal) ----------
  VB.releases = (w) => {
    const base = w.title.replace(/[^A-Za-z0-9 ]/g, '').replace(/ +/g, '.');
    const rows = [
      ['2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC.TrueHD.Atmos.7.1-FRAMESTOR', 'Halide', 61.8e9, 212, '2 d', 'uhd'],
      ['2160p.WEB-DL.DV.HDR.DDP5.1.Atmos.H.265-BYNDR', 'Kestrel', 19.4e9, 640, '3 d', 'uhd'],
      ['1080p.BluRay.x264.DTS-HD.MA.5.1-HONE', 'Aperture', 14.2e9, 388, '2 d', 'hd'],
      ['1080p.WEB-DL.DDP5.1.H.264-NTb', 'Kestrel', 6.8e9, 1120, '4 d', 'hd'],
      ['720p.WEBRip.x264.AAC-SPiRiT', 'Lumen Torznab', 1.9e9, 96, '5 d', null],
    ];
    return `<div class="b-rel" role="table" aria-label="Releases for ${esc(w.title)}">
      <div class="b-rel__h" role="row"><span>Release</span><span>Indexer</span><span>Size</span><span>Seeders</span><span>Meets</span><span></span></div>
      ${rows.map(([n, ix, sz, se, age, t], i) => `<div class="b-rel__r" role="row"><span class="b-rel__n">${esc(base)}.${w.year}.${n}<small>${age} old</small></span><span>${ix}</span><span class="b-num">${F.bytes(sz)}</span><span class="b-num">${se}</span><span>${t ? esc(D.targets.find((x) => x.id === t).name) : '<span class="b-dim">No Target</span>'}</span><span><button class="b-btn b-btn--sm${i === 0 ? '' : ' b-btn--quiet'}" type="button" data-cmd="grab" ${t ? '' : 'disabled'}>Grab</button></span></div>`).join('')}
    </div>`;
  };
})();
