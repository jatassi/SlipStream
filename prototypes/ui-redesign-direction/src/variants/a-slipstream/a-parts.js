(function () {
  const SS = window.SS;
  const d = SS.data;
  const fmt = SS.fmt;
  const VA = window.VA;
  const esc = fmt.esc;
  const icon = VA.icon;

  const btn = (label, { kind = '', ic = '', attrs = '', sm = false } = {}) =>
    `<button class="a-btn ${kind ? 'a-btn--' + kind : ''} ${sm ? 'a-btn--sm' : ''}" ${attrs}>${ic}<span>${esc(label)}</span><i class="a-btn__load" aria-hidden="true"><b></b><b></b><b></b></i></button>`;
  VA.btn = btn;
  VA.iconBtn = (ic, label, attrs = '') => `<button class="a-ibtn" aria-label="${esc(label)}" title="${esc(label)}" ${attrs}>${ic}</button>`;
  VA.seg = (name, opts, value) => `<div class="a-seg" role="tablist" data-seg="${name}">${opts.map(([v, label, n]) => `<button role="tab" aria-selected="${v === value}" data-v="${esc(v)}">${label}${n != null ? `<span class="a-seg__n a-num">${n}</span>` : ''}</button>`).join('')}</div>`;
  VA.prog = (p, state = 'running') => `<span class="a-prog a-prog--${state}" style="--a-p:${p.toFixed(4)}"><span class="a-prog__fill"></span><span class="a-prog__head"></span></span>`;
  // A job's progress is its filament, drawn by VA.Filaments wherever the shell runs one.
  VA.fil = (key, { cls = '', y, pad, s, track = true } = {}) => `<canvas class="a-fil ${cls}" data-fil="${esc(key)}"${y != null ? ` data-fil-y="${y}"` : ''}${pad != null ? ` data-fil-pad="${pad}"` : ''}${s != null ? ` data-fil-s="${s}"` : ''}${track ? '' : ' data-fil-track="off"'} aria-hidden="true"></canvas>`;
  VA.avatar = (id, size = 24) => {
    const a = d.A[id];
    return `<span class="a-avatar" style="--a-hue:${a.hue};width:${size}px;height:${size}px;font-size:${Math.round(size * 0.42)}px" aria-hidden="true">${esc(fmt.initials(a.name))}</span>`;
  };
  VA.firstName = (id) => d.A[id].name.split(' ')[0];

  VA.tile = (w, { quiet = false } = {}) => {
    const st = VA.statusOf(w);
    const meta = w.type === 'series' && !st.tone ? `${w.seasons.length} ${w.seasons.length === 1 ? 'season' : 'seasons'} · ${st.text}` : st.text;
    return `<a class="a-tile" href="#work/${w.id}" data-work="${w.id}" data-key="${w.id}">
      <span class="a-tile__art">${VA.posterImg(w, '', w.id)}${st.job && !quiet ? `<span class="a-tile__prog">${VA.fil('work:' + w.id, { y: 0.7, pad: 0, track: false })}</span>` : ''}</span>
      <span class="a-tile__title">${esc(w.title)}</span>
      <span class="a-tile__meta ${st.tone ? 'is-' + st.tone : ''}" data-flip="st">${esc(meta)}</span>
    </a>`;
  };

  VA.covMarks = (w) => d.targets.map((t) => {
    const c = w.coverage[t.id];
    return `<span class="a-cov a-cov--${c}" title="${esc(t.name)}: ${esc(fmt.coverageLabel[c])}"></span>`;
  }).join('');

  VA.jobRow = (j, open = false) => {
    const w = d.W[j.work];
    const st = j.state;
    const canToggle = st === 'running' || st === 'paused';
    return `<div class="a-job ${open ? 'is-open' : ''}" data-state="${st}" data-key="${j.id}">
      <button class="a-job__main" data-job="${j.id}" aria-expanded="${open}">
        <span class="a-job__art">${VA.posterImg(w, 'a-poster--xs')}</span>
        <span class="a-job__text"><span class="a-job__title">${esc(VA.jobTitle(j))}</span><span class="a-job__line ${st === 'failed' ? 'is-sep' : ''}" data-flip="line">${esc(VA.jobLine(j))}</span></span>
        ${j.release ? `<span class="a-job__rel a-mono">${esc(j.release)}</span>` : `<span class="a-job__rel a-job__note">${esc(VA.jobNote(j))}</span>`}
        <span class="a-job__node">${esc(j.node)}</span>
        <span class="a-job__prog">${VA.fil('job:' + j.id)}</span>
        <span class="a-job__pct a-num" data-live="pct">${st === 'failed' ? '' : fmt.pct(j.progress)}</span>
        <span class="a-job__rate a-num" data-live="rate">${esc(VA.jobRate(j))}</span>
        <span class="a-job__eta a-num" data-live="eta">${esc(VA.jobEta(j) || (st === 'queued' ? 'Waiting' : st === 'stalled' ? 'No seeders' : ''))}</span>
      </button>
      <span class="a-job__actions">${canToggle ? VA.iconBtn(st === 'paused' ? icon('play', { size: 16 }) : icon('pause', { size: 16 }), st === 'paused' ? 'Resume' : 'Pause', `data-toggle="${j.id}"`) : ''}${VA.iconBtn(icon('ellipsis', { size: 16 }), 'More actions', `data-more="${j.id}"`)}</span>
      ${open ? VA.jobDetail(j) : ''}
    </div>`;
  };
  VA.jobDetail = (j) => {
    const w = d.W[j.work];
    const rows = [];
    if (j.release) rows.push(['Release', `<span class="a-mono">${esc(j.release)}</span>`]);
    if (j.indexer) rows.push(['Indexer', esc(j.indexer)]);
    if (j.size) rows.push(['Size', `<span class="a-num">${fmt.bytes(j.size * (j.state === 'failed' ? 1 : j.progress))} of ${fmt.bytes(j.size)}</span>`]);
    if (j.seeds != null) rows.push(['Swarm', `<span class="a-num">${j.seeds} seeds · ${j.peers} peers</span>`]);
    if (j.kind === 'transfer') rows.push(['Transfer', `${esc(j.from)} → ${esc(j.to)} over ${j.streams} streams`]);
    if (j.step) rows.push(['Step', esc(j.step)]);
    if (j.encoder) rows.push(['Encoder', esc(j.encoder)]);
    const dir = j.kind === 'download' ? `/data/downloads/${w.title.replace(/[^A-Za-z0-9 ]/g, '').replace(/ +/g, '.')}` : j.kind === 'transcode' ? `/scratch/derive/${w.id}` : `/mnt/staging/${w.id}`;
    rows.push(['Path', `<span class="a-mono">${esc(dir)}</span>`]);
    if (j.requestedBy) rows.push(['Requested by', esc(d.A[j.requestedBy].name)]);
    const acts = j.state === 'failed' ? btn('Blocklist and search', { kind: 'primary', sm: true, attrs: `data-blocklist="${j.id}"` }) + btn('Open Work', { sm: true, attrs: `data-open-work="${w.id}"` })
      : j.state === 'stalled' ? btn('Find another release', { kind: 'primary', sm: true, attrs: `data-another="${j.id}"` }) + btn('Open Work', { sm: true, attrs: `data-open-work="${w.id}"` })
      : btn('Open Work', { sm: true, attrs: `data-open-work="${w.id}"` });
    return `<div class="a-job__detail">${j.message ? `<p class="a-job__msg">${esc(j.message)}.</p>` : ''}<dl class="a-dl">${rows.map(([k, v]) => `<div><dt>${k}</dt><dd>${v}</dd></div>`).join('')}</dl><div class="a-job__acts">${acts}</div></div>`;
  };
  VA.liveJobs = (box) => {
    box.querySelectorAll('.a-job').forEach((row) => {
      const j = d.jobs.find((x) => x.id === row.dataset.key);
      if (!j) return;
      const set = (k, v) => { const el = row.querySelector(`[data-live="${k}"]`); if (el && el.textContent !== v) el.textContent = v; };
      if (j.state !== 'failed') set('pct', fmt.pct(j.progress));
      set('rate', VA.jobRate(j));
      set('eta', VA.jobEta(j) || (j.state === 'queued' ? 'Waiting' : j.state === 'stalled' ? 'No seeders' : ''));
    });
  };

  VA.needRow = (n) => `<div class="a-need" data-key="${esc(n.id)}" data-flow="${esc(n.flow || '')}">
      <span class="a-need__glyph">${VA.vortex(18)}</span>
      <span class="a-need__text"><span class="a-need__title">${esc(n.title)}</span><span class="a-need__detail">${esc(n.detail || '')}</span></span>
      <span class="a-need__time a-num">${n.at ? esc(fmt.ago(n.at)) : ''}</span>
      ${btn(n.action, { sm: true, kind: n.job ? 'primary' : '', attrs: `data-need="${esc(n.id)}"` })}
    </div>`;
  // Watched items are quieter: no dye, no buttons; each opens where it can be dealt with.
  VA.watchItem = (n) => `<button class="a-witem" type="button" data-key="${esc(n.id)}" data-watch="${esc(n.id)}" data-flow="${esc(n.flow || '')}">${VA.watchGlyph(n.kind, 16)}<span>${esc(n.title)}</span></button>`;
  VA.watchRow = (n) => `<button class="a-wrow" type="button" data-key="${esc(n.id)}" data-watch="${esc(n.id)}">
      <span class="a-wrow__glyph">${VA.watchGlyph(n.kind, 18)}</span>
      <span class="a-wrow__text"><span class="a-wrow__title">${esc(n.title)}</span><span class="a-wrow__detail">${esc(n.detail || '')}</span></span>
      ${icon('chevron-right', { size: 16 })}
    </button>`;

  VA.arrRow = (h) => {
    const w = d.W[h.work];
    const k = VA.arrivalKey(h);
    const fresh = !VA.seen.has(k);
    const title = h.episode ? `${w.title} · ${fmt.ep(h.episode)}` : w.title;
    return `<a class="a-arr ${fresh ? 'is-new' : ''}" href="#work/${w.id}" data-work="${w.id}" data-arr="${esc(k)}">
      <span class="a-arr__art">${VA.posterImg(w, 'a-poster--xs')}</span>
      <span class="a-arr__text"><span class="a-arr__title">${esc(title)}</span><span class="a-arr__detail">${esc(h.text)} · ${esc(h.node)}</span></span>
      <span class="a-arr__time a-num">${esc(fmt.ago(h.at))}</span>
      <span class="a-arr__mark" ${fresh ? 'aria-label="New"' : 'aria-hidden="true"'}></span>
    </a>`;
  };

  VA.nodeCell = (n) => `<a class="a-ncell a-ncell--${n.state}" href="#nodes/${n.id}" data-node="${n.id}" data-key="${n.id}">
      <span class="a-ncell__body">${VA.nodeBody(n, { w: 220, h: 44 })}</span>
      <span class="a-ncell__line"><span class="a-ncell__name">${esc(n.name)}</span><span class="a-ncell__load a-num" data-live="load">${esc(VA.nodeLoad(n))}</span></span>
      <span class="a-ncell__rates a-num" data-live="rates">${esc(VA.nodeRates(n))}</span>
    </a>`;
  VA.liveNodes = (box) => {
    box.querySelectorAll('[data-node]').forEach((el) => {
      const n = d.N[el.dataset.node];
      const set = (k, v) => { const x = el.querySelector(`[data-live="${k}"]`); if (x && x.textContent !== v) x.textContent = v; };
      set('state', VA.nodeState(n));
      set('load', VA.nodeLoad(n));
      set('rates', VA.nodeRates(n));
      set('cpu', fmt.pct(n.cpu));
      if (n.gpu != null) set('gpu', fmt.pct(n.gpu));
      el.querySelectorAll('[data-meter]').forEach((m) => m.style.setProperty('--a-p', String(n[m.dataset.meter] || 0)));
      const svg = el.querySelector('.a-body');
      if (svg) VA.nodeBodyUpdate(svg, n);
    });
  };

  const clockAt = (ms) => new Date(+SS.sim.clock - ms);
  const stamp = (dt) => {
    if (!dt) return '';
    const sameDay = dt.toDateString() === SS.sim.clock.toDateString();
    return sameDay ? fmt.time(dt) : dt.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' });
  };
  VA.workPath = (w) => {
    const j = d.jobs.find((x) => x.work === w.id && VA.isLive(x) && (!x.episode || w.type === 'series'));
    const added = w.added;
    const t = (min) => new Date(+added + min * 60e3);
    if (j) {
      const stage = VA.stageOf(j);
      const startedAgo = j.size && j.rate ? (j.progress * j.size) / Math.max(1e6, j.rate) * 1000 : 30 * 60e3;
      const started = clockAt(startedAgo);
      const stamps = [added, new Date(+started - 3 * 60e3), started, null, null, null];
      if (stage >= 3) { stamps[2] = new Date(+started - 40 * 60e3); stamps[3] = started; }
      if (stage >= 4) { stamps[3] = new Date(+started - 9 * 60e3); stamps[4] = started; }
      const p = j.state === 'failed' ? Math.min(0.62, Math.max(0.3, j.progress)) : j.state === 'queued' ? 0 : j.progress;
      return { at: stage, p, state: j.state === 'failed' ? 'sep' : j.state === 'running' ? 'live' : 'idle', stamps, job: j };
    }
    if (w.status === 'missing' || w.status === 'unreleased' || (w.type === 'series' && w.counts.missing)) return { at: 0, p: 0.2, state: 'idle', stamps: [added, null, null, null, null, null] };
    const imp = d.history.find((h) => h.work === w.id && (h.kind === 'imported' || h.kind === 'upgraded'));
    const lib = imp ? imp.at : t(52);
    const ms = +lib;
    return { at: 5, p: 0, state: 'done', stamps: [added, new Date(ms - 49 * 60e3), new Date(ms - 46 * 60e3), new Date(ms - 9 * 60e3), new Date(ms - 2 * 60e3), lib] };
  };
  VA.strip = ({ stops, at, p, state, stamps = [], notes = [] }) => {
    const n = stops.length;
    const pos = state === 'done' ? (at + 0.06) / n : (at + 0.07 + 0.86 * p) / n;
    const pct = (pos * 100).toFixed(3);
    const yAt = (x) => 20 + Math.sin(x * 0.011 + 0.6) * 4 + Math.sin(x * 0.031) * 1.2;
    const pts = [];
    for (let x = 0; x <= 1000; x += 20) pts.push(`${x} ${yAt(x).toFixed(2)}`);
    const line = `M${pts.join('L')}`;
    return `<div class="a-strip a-strip--${state}" style="--a-n:${n};--a-pos:${pct}%;--a-hy:${yAt(pos * 1000).toFixed(2)}px">
      <svg class="a-strip__svg" viewBox="0 0 1000 40" preserveAspectRatio="none" aria-hidden="true">
        ${stops.map((_, i) => `<line class="a-strip__rake" x1="${(i / n) * 1000}" x2="${(i / n) * 1000}" y1="0" y2="40"/>`).join('')}
        <path class="a-strip__future" d="${line}"/>
        <path class="a-strip__past" d="${line}" style="clip-path: inset(0 ${(100 - pos * 100).toFixed(3)}% 0 0)"/>
      </svg>
      <span class="a-strip__head">${state === 'sep' ? VA.vortex(20) : ''}</span>
      <ol class="a-strip__stops">${stops.map((s, i) => `<li class="${i < at || (i === at && state !== 'idle') || state === 'done' ? 'is-past' : ''} ${i === at ? 'is-at' : ''}"><span class="a-strip__name">${esc(s)}</span><span class="a-strip__time a-num">${esc(notes[i] || stamp(stamps[i]))}</span></li>`).join('')}</ol>
    </div>`;
  };
  VA.pathStrip = (w) => {
    const P = VA.workPath(w);
    const notes = [];
    if (P.job) {
      const j = P.job;
      const pct = fmt.pct(j.progress);
      if (j.state === 'failed') notes[P.at] = j.kind === 'import' ? 'Import failed' : 'Failed';
      else if (j.state === 'stalled') notes[P.at] = `Stalled at ${pct}`;
      else if (j.state === 'queued') notes[P.at] = 'Queued';
      else notes[P.at] = `${pct}${j.eta ? ` · ${fmt.dur(j.eta)} left` : ''}`;
    }
    return VA.strip({ stops: VA.STAGES, at: P.at, p: P.p, state: P.state, stamps: P.stamps, notes });
  };
  VA.journeyStrip = (r) => {
    const J = VA.journey(r);
    const notes = [stamp(r.at), J.step >= 1 ? stamp(VA.approvedAt(r)) : '', J.step === 2 ? J.detail.replace('Ready in about ', 'About ') : '', J.step === 3 ? stamp(r.doneAt) : ''];
    return VA.strip({ stops: VA.JOURNEY, at: J.step, p: J.step === 2 ? J.progress || 0 : 0.15, state: J.step === 3 ? 'done' : J.step === 2 ? 'live' : 'idle', notes });
  };

  VA.coverage = (w) => {
    const prof = d.profiles.find((p) => p.id === w.profile);
    const eds = w.editions;
    const live = d.jobs.filter((j) => j.work === w.id && VA.isLive(j));
    const cell = (t, ei) => {
      let c = w.coverage[t.id];
      if (ei > 0 && c !== 'unwanted') c = 'missing';
      const j = ei === 0 && live.find((x) => x.target === t.id);
      if (j && j.state !== 'failed') return `<span class="a-cell a-cell--live"><span class="a-cell__label" data-flip="cov-${t.id}">${esc(VA.jobVerb(j))} <span class="a-num" data-live="cov-${t.id}">${fmt.pct(j.progress)}</span></span>${VA.fil('job:' + j.id, { y: 0.55, pad: 2 })}</span>`;
      if (j && j.state === 'failed') return `<span class="a-cell a-cell--sep"><span class="a-cell__label">${VA.vortex(14)}Import failed</span></span>`;
      const file = w.files?.find((f) => f.target === t.id);
      return `<span class="a-cell a-cell--${c}"><span class="a-cov a-cov--${c}"></span><span class="a-cell__label">${esc(fmt.coverageLabel[c])}</span>${file && ei === 0 ? `<span class="a-cell__sub a-num">${fmt.bytes(file.size)}</span>` : ''}</span>`;
    };
    return `<div class="a-covgrid" style="--a-eds:${eds.length}">
      <div class="a-covgrid__head"><span>Target</span>${eds.map((e) => `<span>${esc(e)}</span>`).join('')}<span>Source</span></div>
      ${d.targets.map((t) => {
        const inProf = prof.targets.includes(t.id);
        return `<div class="a-covgrid__row ${inProf ? '' : 'is-off'}"><span class="a-covgrid__t"><b>${esc(t.name)}</b><small>${esc(t.note)}</small></span>${eds.map((_, i) => cell(t, i)).join('')}<span class="a-covgrid__src">${inProf ? (t.kind === 'derived' ? `Derived from ${esc(VA.targetName(t.from))}` : 'Downloaded') : 'Not in Profile'}</span></div>`;
      }).join('')}
    </div>`;
  };

  VA.files = (w) => {
    if (!w.files?.length) {
      const j = d.jobs.find((x) => x.work === w.id && VA.isLive(x));
      return `<div class="a-empty"><p>No files yet.</p><p class="a-empty__sub">${j ? `The ${esc(VA.targetName(j.target))} file appears here once it is imported on vault.` : 'Search for a release, or use Manual search to pick one yourself.'}</p></div>`;
    }
    return `<div class="a-table a-files" role="table">
      <div class="a-table__head" role="row"><span role="columnheader">File</span><span role="columnheader">Target</span><span role="columnheader">Video</span><span role="columnheader">Audio</span><span role="columnheader" class="is-num">Size</span><span role="columnheader">Node</span></div>
      ${w.files.map((f) => `<div class="a-table__row" role="row"><span class="a-mono a-files__name" role="cell" title="${esc(f.name)}">${esc(f.name)}</span><span role="cell">${esc(VA.targetName(f.target))}${f.derived ? '<span class="a-tag">Derived</span>' : ''}</span><span role="cell">${esc(f.video)}<small>${esc(f.resolution)}</small></span><span role="cell">${esc(f.audio)}</span><span role="cell" class="is-num a-num">${fmt.bytes(f.size)}</span><span role="cell">${esc(f.node)}</span></div>`).join('')}
    </div>`;
  };

  VA.episodes = (w, sn) => {
    const s = w.seasons[sn - 1];
    return `<div class="a-eps">${s.episodes.map((e) => {
      const j = d.jobs.find((x) => x.work === w.id && x.episode && x.episode[0] === s.n && x.episode[1] === e.n && VA.isLive(x));
      let st;
      if (j) st = `<span class="a-ep__st is-live"><span data-flip="ep">${esc(VA.jobVerb(j))}</span> <span class="a-num" data-live="ep-${e.n}">${fmt.pct(j.progress)}</span>${VA.fil('job:' + j.id, { y: 0.55, pad: 2 })}</span>`;
      else if (e.status === 'failed') st = `<span class="a-ep__st is-sep">${VA.vortex(14)}Failed</span>`;
      else if (e.status === 'missing') st = `<span class="a-ep__st is-muted">Missing</span>`;
      else if (e.status === 'unreleased') st = `<span class="a-ep__st is-muted">Airs ${esc(fmt.until(e.air))}</span>`;
      else st = `<span class="a-ep__st">${esc(VA.targetName('hd'))}</span>`;
      const act = e.status === 'missing' || e.status === 'failed' ? VA.iconBtn(icon('search', { size: 16 }), `Search for ${fmt.ep([s.n, e.n])}`, `data-ep-search="${s.n}.${e.n}"`) : '';
      return `<div class="a-ep" data-key="${s.n}.${e.n}" data-status="${j ? 'live' : e.status}"><span class="a-ep__n a-num">${e.n}</span><span class="a-ep__title">${esc(e.title)}</span><span class="a-ep__air a-num">${e.air.toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })}</span>${st}<span class="a-ep__act">${act}</span></div>`;
    }).join('')}</div>`;
  };

  // A pending ask waits at the Wanted rake as a wisp; once approved it flows like any job.
  VA.reqStatus = (r) => {
    if (r.state === 'denied') return `Denied · ${r.reason || 'No reason given'}`;
    if (r.state === 'available') return `Available · ${fmt.ago(r.doneAt)}`;
    const J = VA.journey(r);
    return J.step === 2 ? `On its way · ${J.detail}` : `Approved · ${J.detail}`;
  };
  VA.reqRow = (r) => {
    const w = d.W[r.work];
    const who = d.A[r.by];
    const pending = r.state === 'pending';
    const flows = pending || VA.requestFlow(r).length;
    const acts = pending
      ? btn('Approve', { kind: 'primary', sm: true, attrs: `data-approve="${r.id}"` }) + btn('Deny', { sm: true, attrs: `data-deny="${r.id}"` })
      : btn('Open', { sm: true, attrs: `data-open-work="${w.id}"` });
    return `<div class="a-req" data-key="${r.id}" data-state="${r.state}">
      <span class="a-req__lead">${flows ? VA.fil('req:' + r.id, { y: 0.5, pad: 2, s: 0.62, track: false }) : ''}</span>
      <span class="a-req__art">${VA.posterImg(w, 'a-poster--sm')}</span>
      <span class="a-req__text">
        <span class="a-req__title">${esc(w.title)} <span class="a-req__year">${w.year}</span></span>
        <span class="a-req__by">${VA.avatar(r.by, 18)}${esc(who.name)} · ${esc(fmt.ago(r.at))}</span>
      </span>
      <span class="a-req__ask">
        <span class="a-req__scope">${esc(VA.requestScope(r))}</span>
        ${pending ? (r.note ? `<span class="a-req__note">“${esc(r.note)}”</span>` : '') : `<span class="a-req__sub is-${r.state}" data-live="sub">${esc(VA.reqStatus(r))}</span>`}
      </span>
      <span class="a-req__acts">${acts}</span>
      <form class="a-req__deny" data-deny-form="${r.id}" hidden><input class="a-input" name="reason" placeholder="Reason, shown to ${esc(who.name.split(' ')[0])}" aria-label="Reason for denying"><button class="a-btn a-btn--sm a-btn--primary" type="submit">Deny request</button><button class="a-btn a-btn--sm a-btn--ghost" type="button" data-deny-cancel>Cancel</button></form>
    </div>`;
  };

  VA.workMeta = (w, short = false) => {
    const bits = [w.type === 'series' && w.endYear ? `${w.year} to ${w.endYear}` : w.year];
    if (w.runtime) bits.push(fmt.runtime(w.runtime));
    if (w.type === 'series') bits.push(`${w.seasons.length} ${w.seasons.length === 1 ? 'season' : 'seasons'}`, w.network);
    bits.push(w.cert);
    if (!short) bits.push(w.genres.join(', '));
    if (w.rating && !short) bits.push(`Rated ${w.rating.toFixed(1)}`);
    return bits.filter(Boolean).map((b) => `<span>${esc(String(b))}</span>`).join('');
  };
})();
