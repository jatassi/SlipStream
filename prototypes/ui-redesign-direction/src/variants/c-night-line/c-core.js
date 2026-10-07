(function () {
  'use strict';
  const SS = window.SS;
  const d = SS.data;
  const F = SS.fmt;
  const esc = F.esc;
  const VC = (window.VC = {});

  VC.LINES = [
    { id: 'acquire', key: 'A', name: 'Acquire', ink: '#F0524F' },
    { id: 'library', key: 'L', name: 'Library', ink: '#3D7BFF' },
    { id: 'process', key: 'P', name: 'Process', ink: '#F4A62A' },
    { id: 'deliver', key: 'D', name: 'Deliver', ink: '#22C08A' },
    { id: 'notify', key: 'N', name: 'Notify', ink: '#A99BFF' },
  ];
  VC.L = Object.fromEntries(VC.LINES.map((l) => [l.id, l]));
  VC.roleLine = {};
  for (const g of d.roleGroups) for (const r of g.roles) VC.roleLine[r] = g.id;
  const JOB_LINE = { download: 'acquire', transfer: 'acquire', search: 'acquire', import: 'library', transcode: 'process', subtitles: 'process' };
  VC.jobLine = (j) => JOB_LINE[j.kind] || 'acquire';
  VC.nodeLines = (n) => VC.LINES.filter((l) => n.roles.some((r) => VC.roleLine[r] === l.id)).map((l) => l.id);
  VC.bullet = (id, cls = '') => {
    const l = VC.L[id];
    return `<span class="c-bullet ${cls}" data-line="${id}" style="--ink:${l.ink}" role="img" aria-label="${l.name} line">${l.key}</span>`;
  };
  VC.bullets = (ids, cls) => `<span class="c-bullets">${ids.map((id) => VC.bullet(id, cls)).join('')}</span>`;

  VC.target = (id) => d.targets.find((t) => t.id === id);
  VC.targetName = (id) => VC.target(id)?.name || '';
  VC.profile = (w) => d.profiles.find((p) => p.id === w.profile);
  VC.clock = () => (SS.sim ? SS.sim.clock : d.now);
  VC.reduced = () => matchMedia('(prefers-reduced-motion: reduce)').matches;
  VC.emit = (ev, payload) => SS.sim?.emit(ev, payload);

  const VERB = { download: 'Downloading', transfer: 'Transferring', import: 'Importing', transcode: 'Deriving', subtitles: 'Subtitles', search: 'Searching' };
  const STATE_WORD = { stalled: 'Stalled', failed: 'Failed', queued: 'Queued', paused: 'Paused', done: 'Arrived' };
  VC.jobStage = (j) => STATE_WORD[j.state] || VERB[j.kind];
  VC.jobVerb = (j) => VERB[j.kind];
  VC.jobItem = (j) => (j.episode ? F.ep(j.episode) : '');
  VC.jobTitle = (j) => d.W[j.work]?.title || j.label;
  VC.jobPath = (j) => (j.kind === 'transfer' ? [j.from, j.to] : [j.node]);
  VC.jobRate = (j) => {
    if (j.state !== 'running') return '';
    if (j.kind === 'transcode' || j.kind === 'subtitles') return j.speed ? `${j.speed.toFixed(1)}×` : '';
    return F.rate(j.rate);
  };
  VC.jobEta = (j) => (j.state === 'running' && j.eta ? F.dur(j.eta) : '');
  VC.isLive = (j) => j.state !== 'done';
  const KIND_RANK = { download: 1, transfer: 2, import: 3, transcode: 4, subtitles: 4 };
  VC.workJobs = (w) => d.jobs.filter((j) => j.work === w.id && j.state !== 'done');
  VC.workJob = (w) => VC.workJobs(w).sort((a, b) => KIND_RANK[b.kind] - KIND_RANK[a.kind])[0];

  const LEG = { download: [0, 0.82], transfer: [0.82, 0.97], import: [0.97, 1] };
  VC.journey = (w) => {
    const j = VC.workJobs(w).filter((x) => LEG[x.kind]).sort((a, b) => KIND_RANK[b.kind] - KIND_RANK[a.kind])[0];
    if (!j) return null;
    const [a, b] = LEG[j.kind];
    const p = a + (b - a) * (j.progress || 0);
    const rest = { download: ['transfer', 'import'], transfer: ['import'], import: [] }[j.kind];
    const eta = (j.eta || 0) + rest.reduce((s, k) => s + legSeconds(k, j.size), 0);
    return { job: j, p, eta, state: j.state };
  };
  function legSeconds(kind, size = 10e9, w) {
    return { search: 60, grab: 40, download: size / 40e6, transfer: size / 96e6, import: size / 700e6 + 20, derive: ((w?.runtime || 50) * 60) / 4.1, available: 40 }[kind] || 60;
  }

  VC.nodeState = (n) => (n.paused ? 'paused' : n.state === 'offline' ? 'suspended' : n.state === 'degraded' ? 'delays' : 'good');

  // A line's service, worst first. Nodes and jobs both disrupt a line, so both feed it.
  const SERVICE = [
    { state: 'failed', tone: 'failed', has: (l) => l.failed.length, short: (l) => `${l.failed.length} failed` },
    { state: 'suspended', tone: 'suspended', has: (l) => l.nodes.length && l.suspended.length === l.nodes.length, short: () => 'Suspended' },
    { state: 'delays', tone: 'delays', has: (l) => l.delays.length, short: () => 'Delays' },
    { state: 'stalled', tone: 'delays', has: (l) => l.stalled.length, short: (l) => `${l.stalled.length} stalled` },
    { state: 'part', tone: 'suspended', has: (l) => l.suspended.length, short: () => 'Part suspended' },
    { state: 'paused', tone: 'paused', has: (l) => l.paused.length, short: () => 'Paused' },
    { state: 'good', tone: '', has: () => true, short: () => 'Good service' },
  ];
  VC.net = () => {
    const lines = {};
    for (const l of VC.LINES) lines[l.id] = { ...l, running: 0, nodes: [], delays: [], suspended: [], paused: [], failed: [], stalled: [] };
    for (const j of d.jobs) {
      const l = lines[VC.jobLine(j)];
      if (j.state === 'running') l.running++;
      if (j.state === 'failed' || j.state === 'stalled') l[j.state].push(j);
    }
    for (const n of d.nodes) {
      const st = VC.nodeState(n);
      for (const id of VC.nodeLines(n)) {
        lines[id].nodes.push(n.id);
        if (st !== 'good') lines[id][st].push(n.id);
      }
    }
    for (const l of Object.values(lines)) {
      const s = SERVICE.find((x) => x.has(l));
      l.state = s.state;
      l.tone = s.tone;
      l.short = s.short(l);
      l.label = SERVICE.filter((x) => x.state !== 'good' && x.has(l)).map((x) => x.short(l)).join(', ') || 'Good service';
    }
    const sentence = [];
    const fully = Object.values(lines).filter((l) => l.state === 'suspended');
    if (fully.length) sentence.push({ tone: 'suspended', text: `${F.plural(fully.length, 'line')} suspended` });
    for (const n of d.nodes) {
      const st = VC.nodeState(n);
      const names = VC.nodeLines(n).map((id) => VC.L[id].name);
      if (st === 'delays') sentence.push({ tone: 'delay', text: `Delays on ${F.list(names)} at ${n.name}` });
      if (st === 'paused') sentence.push({ tone: 'paused', text: `${n.name} paused` });
      if (st === 'suspended') sentence.push({ tone: 'suspended', text: `${n.name} suspended` });
    }
    for (const l of Object.values(lines)) {
      if (l.failed.length) sentence.push({ tone: 'failed', text: `${F.plural(l.failed.length, 'failure')} on ${l.name}` });
      if (l.stalled.length) sentence.push({ tone: 'delay', text: `${F.plural(l.stalled.length, 'stall')} on ${l.name}` });
    }
    if (!sentence.length) sentence.push('All lines running');
    return { lines, sentence };
  };

  // One row per Work or episode: a finished leg yields to the leg that took over from it.
  // What Activity's All tab lists: unfinished work plus arrivals still lit, one row per Work-episode.
  VC.activityRows = () => VC.collapse(SS.data.jobs.filter((j) => j.state !== 'done' || VC.lit.has(j.id)));
  VC.collapse = (list) => {
    const items = new Map();
    for (const j of list) {
      const k = j.work + (j.episode ? ' ' + j.episode.join('x') : '');
      if (!items.has(k)) items.set(k, []);
      items.get(k).push(j);
    }
    return [...items.values()].flatMap((g) => {
      const live = g.filter((j) => j.state !== 'done');
      return live.length ? live : [g.sort((a, b) => KIND_RANK[b.kind] - KIND_RANK[a.kind])[0]];
    });
  };

  VC.disruptions = () => d.alerts.filter((a) => a.level !== 'info').map((a) => {
    const node = a.node && d.N[a.node];
    const job = a.work && d.jobs.find((j) => j.work === a.work && (j.state === 'failed' || j.state === 'stalled'));
    const line = node ? VC.nodeLines(node)[0] : job ? VC.jobLine(job) : 'acquire';
    const kind = node && node.state === 'offline' ? 'suspended' : a.level === 'error' ? 'failed' : 'delay';
    return { ...a, line, kind, job, nodeRef: node };
  }).sort((x, y) => (x.level === y.level ? y.at - x.at : x.level === 'error' ? -1 : 1));

  // A departure keeps the time it was first given; the sim runs work faster than its clock, so a live estimate would slide by the second.
  const SCHEDULE = new Map();
  const scheduled = (id, at) => {
    if (!SCHEDULE.has(id)) SCHEDULE.set(id, at);
    return SCHEDULE.get(id);
  };
  VC.departures = (limit = 6) => {
    const now = VC.clock();
    const out = [];
    for (const n of d.nodes) {
      const queued = d.jobs.filter((j) => j.node === n.id && j.state === 'queued');
      if (!queued.length) continue;
      const running = d.jobs.filter((j) => j.node === n.id && j.state === 'running');
      let t = now.getTime() + (running.length ? Math.min(...running.map((j) => j.eta || 0)) : 0) * 1000;
      for (const j of queued) {
        const w = d.W[j.work];
        out.push({ id: j.id, at: scheduled(j.id, new Date(t)), line: VC.jobLine(j), title: w.title, sub: j.kind === 'subtitles' ? 'Subtitles · ' + (j.step || 'English') : `${VC.targetName(j.target)} from ${VC.targetName(j.from)}`, platform: n.name, status: VC.nodeState(n) === 'delays' ? 'Delayed' : VC.nodeState(n) === 'suspended' ? 'Cancelled' : 'On time', tone: VC.nodeState(n) === 'delays' ? 'delay' : '', work: w.id });
        t += (j.kind === 'subtitles' ? 240 : legSeconds('derive', 0, w)) * 1000;
      }
    }
    const limited = d.indexers.find((x) => x.state === 'limited');
    for (const c of d.calendar) {
      if (c.at < now) continue;
      const w = d.W[c.work];
      const late = limited && c.at.getTime() - now.getTime() < 40 * 60e3;
      out.push({ id: 'cal-' + c.work + (c.episode ? c.episode.join('-') : ''), at: c.at, line: 'acquire', title: w.title, sub: c.episode ? `Search for ${F.ep(c.episode)}` : 'Search on release', platform: 'atlas', status: late ? 'Delayed' : 'On time', tone: late ? 'delay' : '', work: w.id });
    }
    return out.sort((a, b) => a.at - b.at).slice(0, limit);
  };
  VC.boardTime = (date) => {
    const now = VC.clock();
    const same = date.toDateString() === now.toDateString();
    return same ? F.time(date) : `${date.toLocaleDateString('en-GB', { weekday: 'short' })} ${F.time(date)}`;
  };
  // An arrival estimate jitters with download rates; the time only turns over when it moves by a minute or more.
  VC.boardFlap = (el, at) => {
    if (!el) return;
    const prev = +el.dataset.at;
    if (prev && Math.abs(at - prev) < 60e3) return;
    el.dataset.at = +at;
    VC.flap(el, VC.boardTime(at));
  };

  const STOP_FOR = { download: 'downloaded', transfer: 'transferred', import: 'imported', transcode: 'derived', subtitles: 'derived' };
  VC.route = (w) => {
    const now = VC.clock().getTime();
    const prof = VC.profile(w);
    const wantsMobile = prof.targets.includes('mobile');
    const job = VC.workJob(w);
    const mainTarget = job?.target && job.kind !== 'transcode' ? job.target : prof.targets[0];
    const file = w.files?.find((f) => f.target === mainTarget);
    const size = job?.size && job.kind !== 'transcode' ? job.size : file?.size || (w.type === 'series' ? 3.2e9 : 14e9);
    const forge = d.N.forge;
    const forgeLate = VC.nodeState(forge) === 'delays';
    const group = (job?.release || file?.name || '').split('-').pop();
    const stops = [
      { id: 'searched', line: 'acquire', label: 'Searched', detail: job?.indexer || 'Kestrel', dur: 0 },
      { id: 'grabbed', line: 'acquire', label: 'Grabbed', detail: group ? `${group} · ${F.bytes(size)}` : F.bytes(size), dur: legSeconds('grab') },
      { id: 'downloaded', line: 'acquire', label: 'Downloaded on ams-1', detail: `${F.bytes(size)} at ams-1`, dur: legSeconds('download', size) },
      { id: 'transferred', line: 'acquire', label: 'Transferred to vault', detail: 'Transfer from ams-1', dur: legSeconds('transfer', size) },
      { id: 'imported', line: 'library', label: 'Imported', detail: `${VC.targetName(mainTarget)} into the library`, dur: legSeconds('import', size) },
      wantsMobile && { id: 'derived', line: 'process', label: 'Derived 720p Mobile on forge', detail: 'From 1080p · NVENC HEVC', dur: legSeconds('derive', 0, w) + (forgeLate ? 7200 : 0) },
      { id: 'available', line: 'deliver', label: 'Available in Plex', detail: `${F.plural(prof.targets.length, 'Target')} on vault`, dur: legSeconds('available') },
    ].filter(Boolean);
    let context = w.type === 'movie' ? VC.targetName(mainTarget) : '';
    const idx = (id) => stops.findIndex((s) => s.id === id);

    if (job) {
      const i = idx(STOP_FOR[job.kind]);
      const cur = stops[i];
      cur.state = { running: 'current', stalled: 'delayed', failed: 'failed', queued: 'waiting', paused: 'paused' }[job.state] || 'current';
      cur.progress = job.progress;
      cur.job = job;
      if (job.kind === 'transcode' || job.kind === 'subtitles') cur.detail = `${VC.targetName(job.target)} from ${VC.targetName(job.from || 'hd')}${forgeLate ? ' · forge is 2 h behind' : ''}`;
      if (job.state === 'running') cur.detail = `${F.pct(job.progress)} · ${VC.jobRate(job)}${job.eta ? ` · ${F.dur(job.eta)} left` : ''}`;
      if (job.message) cur.detail = job.message;
      if (job.state === 'queued') cur.detail = `Waiting for forge${forgeLate ? ' · 2 h behind' : ''}`;
      const elapsed = cur.dur * (job.progress || 0);
      let t = now - elapsed * 1000;
      cur.at = new Date(job.state === 'running' ? now + (job.eta || cur.dur - elapsed) * 1000 : t);
      cur.est = job.state === 'running';
      for (let k = i - 1; k >= 0; k--) { stops[k].state = 'done'; stops[k].at = new Date(t); t -= stops[k].dur * 1000; }
      let f = cur.at.getTime();
      for (let k = i + 1; k < stops.length; k++) { stops[k].state = 'next'; f += stops[k].dur * 1000; stops[k].at = new Date(f); stops[k].est = true; }
      if (job.episode) context = `${F.ep(job.episode)} · ${w.seasons[job.episode[0] - 1].episodes[job.episode[1] - 1].title}`;
      return { stops, context, job, live: job.state === 'running' };
    }

    if (w.status === 'unreleased' || (w.status === 'missing' && w.type === 'movie')) {
      stops.forEach((s) => (s.state = 'next'));
      if (w.status === 'unreleased') {
        stops[0] = { ...stops[0], label: 'Released', detail: w.release ? `Digital release ${F.until(w.release)}` : `Due ${w.year}`, state: 'waiting', at: w.release || null };
      } else {
        stops[0] = { ...stops[0], detail: 'No release passes the Profile yet · next search in 3 h', state: 'waiting', at: new Date(now - 30 * 60e3) };
      }
      return { stops, context, job: null, live: false };
    }

    const hist = d.history.filter((h) => h.work === w.id && !h.episode);
    const histFor = { imported: hist.find((h) => h.kind === 'imported'), transferred: hist.find((h) => h.kind === 'transferred'), derived: hist.find((h) => h.kind === 'derived') };
    let anchor = histFor.imported?.at?.getTime() ?? Math.min(now - 3600e3, w.added.getTime() + 2 * 3600e3);
    let series = null;
    if (w.type === 'series') {
      series = d.history.find((h) => h.work === w.id && h.episode);
      if (series) { anchor = series.at.getTime(); context = `${F.ep(series.episode)} · ${w.seasons[series.episode[0] - 1].episodes[series.episode[1] - 1].title}`; }
      else {
        const last = w.seasons.flatMap((s) => s.episodes.map((e) => ({ s: s.n, e }))).filter((x) => x.e.status === 'available').pop();
        if (last) { anchor = Math.min(now - 86400e3, last.e.air.getTime() + 3 * 3600e3); context = `S${last.s} E${last.e.n} · ${last.e.title}`; }
      }
    }
    const ia = idx('imported');
    stops[ia].at = new Date(anchor);
    for (let k = ia - 1; k >= 0; k--) {
      const exact = histFor[stops[k].id]?.at?.getTime();
      stops[k].at = new Date(Math.min(exact ?? Infinity, stops[k + 1].at.getTime() - stops[k + 1].dur * 1000));
    }
    for (let k = ia + 1; k < stops.length; k++) {
      const exact = histFor[stops[k].id]?.at?.getTime();
      stops[k].at = new Date(Math.min(now - 60e3 * (stops.length - k), exact ?? stops[k - 1].at.getTime() + Math.min(stops[k].dur, 5400) * 1000));
      if (stops[k].at < stops[k - 1].at) stops[k].at = new Date(stops[k - 1].at.getTime() + 60e3);
    }
    stops.forEach((s) => (s.state = 'done'));
    if (w.coverage.mobile === 'missing' && wantsMobile) { const s = stops[idx('derived')]; s.state = 'next'; s.at = null; s.detail = 'Waiting for 1080p'; }
    return { stops, context, job: null, live: false };
  };

  VC.coverage = (w) => {
    const prof = VC.profile(w);
    const j = VC.journey(w);
    return d.targets.map((t) => {
      const raw = w.coverage[t.id];
      const cell = { target: t, raw, inProfile: prof.targets.includes(t.id) };
      if (raw === 'downloading' && j && j.job.target === t.id) { cell.state = 'moving'; cell.p = j.p; cell.job = j.job; }
      else if (raw === 'downloading') { cell.state = 'queued'; }
      else cell.state = raw;
      return cell;
    });
  };

  let seq = 1;
  const dropAlert = (pred) => {
    const i = d.alerts.findIndex(pred);
    if (i >= 0) d.alerts.splice(i, 1);
    VC.emit('alerts:update');
  };
  VC.act = {
    approve(r) {
      r.state = 'approved';
      r.decidedAt = VC.clock();
      VC.emit('request:update', r);
      const w = d.W[r.work];
      const ep = w.type === 'series' ? w.seasons.flatMap((s) => s.episodes.map((e) => ({ s: s.n, e }))).find((x) => (!r.seasons || r.seasons.includes(x.s)) && (x.e.status === 'missing' || x.e.status === 'failed')) : null;
      if (w.status === 'unreleased' || (w.type === 'series' && !ep)) return;
      setTimeout(() => {
        if (r.state !== 'approved') return;
        r.state = 'downloading';
        VC.act.grab(w.id, ep ? [ep.s, ep.e.n] : null);
        VC.emit('request:update', r);
      }, 2600);
    },
    grab(workId, episode) {
      const w = d.W[workId];
      const target = VC.profile(w).targets[0];
      const big = target === 'uhd';
      const nj = { id: 'cx' + seq++, kind: 'download', work: workId, episode: episode || undefined, label: w.title + (episode ? ' · ' + F.ep(episode) : ''), target: episode ? 'hd' : target, release: `${w.title.replace(/[^A-Za-z0-9]+/g, '.')}${episode ? '.' + F.sxe(episode) : '.' + w.year}.${big && !episode ? '2160p.WEB-DL.DV.HDR' : '1080p.WEB-DL'}.DDP5.1.H.265-LUMiNA`, node: 'ams-1', size: episode ? 2.8e9 : big ? 24e9 : 9.6e9, progress: 0, rate: 20e6, seeds: 140, peers: 22, state: 'running', indexer: 'Kestrel' };
      if (episode) { const e = w.seasons[episode[0] - 1].episodes[episode[1] - 1]; e.status = 'downloading'; w.counts.downloading++; w.counts.missing = Math.max(0, w.counts.missing - 1); }
      else if (w.coverage[nj.target] === 'missing') w.coverage[nj.target] = 'downloading';
      d.jobs.push(nj);
      VC.emit('job:new', nj);
      VC.emit('work:update', w);
      return nj;
    },
    deny(r, reason) { r.state = 'denied'; r.reason = reason || 'Not this time.'; r.decidedAt = VC.clock(); VC.emit('request:update', r); },
    request(workId, seasons) {
      const r = { id: 'rq' + seq++, work: workId, by: d.member, at: VC.clock(), state: 'pending', seasons, note: '' };
      d.requests.unshift(r);
      VC.emit('request:update', r);
      return r;
    },
    blocklistAndSearch(workId) {
      const failed = d.jobs.find((j) => j.work === workId && j.state === 'failed');
      if (failed) d.jobs.splice(d.jobs.indexOf(failed), 1);
      dropAlert((a) => a.work === workId);
      const w = d.W[workId];
      const nj = { id: 'cx' + seq++, kind: 'download', work: workId, label: w.title, target: failed?.target || 'hd', release: `${w.title.replace(/[^A-Za-z0-9]+/g, '.')}.${w.year}.1080p.BluRay.x264-HONE`, node: 'ams-1', size: 9.4e9, progress: 0, rate: 20e6, seeds: 88, peers: 9, state: 'running', indexer: 'Kestrel' };
      d.jobs.push(nj);
      VC.emit('job:new', nj);
      return nj;
    },
    findRelease(workId) {
      const j = d.jobs.find((x) => x.work === workId && x.state === 'stalled');
      if (!j) return;
      const w = d.W[workId];
      j.release = `${w.title.replace(/[^A-Za-z0-9]+/g, '.')}.${w.year}.2160p.WEB-DL.DV.HDR.DDP5.1.Atmos.H.265-LUMiNA`;
      j.seeds = 64; j.peers = 11; j.message = null; j.state = 'running'; j.indexer = 'Kestrel';
      dropAlert((a) => a.work === workId);
      VC.emit('job:update', j);
    },
    pauseNode(id) {
      const n = d.N[id];
      n.paused = true;
      for (const j of d.jobs) if (j.node === id && j.state === 'running') { j.state = 'paused'; VC.emit('job:update', j); }
      VC.emit('alerts:update');
    },
    resumeNode(id) {
      const n = d.N[id];
      n.paused = false;
      for (const j of d.jobs) if (j.node === id && j.state === 'paused') { j.state = 'running'; VC.emit('job:update', j); }
      VC.emit('alerts:update');
    },
    resumeAll() { for (const n of d.nodes) if (n.paused) VC.act.resumeNode(n.id); },
    missingEpisodes() {
      const out = [];
      for (const w of d.works) if (w.type === 'series') w.seasons.forEach((s) => s.episodes.forEach((e) => { if (e.status === 'missing' || e.status === 'failed') out.push({ w, s: s.n, e }); }));
      return out;
    },
  };

  VC.lit = new Map();
  VC.light = (id) => VC.lit.set(id, Date.now());
  VC.notice = (el) => {
    const id = el?.dataset.lit;
    if (!id || !VC.lit.has(id) || Date.now() - VC.lit.get(id) < 1600) return;
    VC.lit.delete(id);
    el.classList.add('is-noticed');
    el.removeAttribute('data-lit');
  };

  VC.flap = (el, text) => {
    if (!el) return;
    text = String(text ?? '');
    const old = el.dataset.v;
    if (old === text) return;
    el.dataset.v = text;
    if (old == null || VC.reduced()) { el.textContent = text; return; }
    let html = '';
    for (let i = 0; i < text.length; i++) {
      const c = esc(text[i]);
      html += text[i] !== old[i] && text[i] !== ' ' ? `<span class="c-flap" style="animation-delay:${Math.min(i, 8) * 22}ms">${c}</span>` : c;
    }
    el.innerHTML = html;
  };
  VC.set = (el, text) => { if (el && el.textContent !== text) el.textContent = text; };

  // A spring as a CSS linear() easing, so gesture releases settle on the compositor with the finger's velocity.
  VC.spring = ({ velocity = 0, response = 0.42, damping = 1 } = {}) => {
    const w0 = (2 * Math.PI) / response;
    const zeta = damping;
    const x = (t) => {
      if (zeta >= 1) return (1 + (w0 - velocity) * t) * Math.exp(-w0 * t);
      const wd = w0 * Math.sqrt(1 - zeta * zeta);
      return Math.exp(-zeta * w0 * t) * (Math.cos(wd * t) + ((zeta * w0 - velocity) / wd) * Math.sin(wd * t));
    };
    let T = 0.1;
    while (T < 2 && (Math.abs(x(T)) > 0.002 || Math.abs(x(T + 0.016)) > 0.002)) T += 0.016;
    const n = 48;
    const pts = [];
    for (let i = 0; i <= n; i++) pts.push((1 - x((T * i) / n)).toFixed(4));
    pts[n] = '1';
    return { easing: `linear(${pts.join(', ')})`, duration: Math.round(T * 1000) };
  };

  // A toast with a key replaces the live toast that shares it, so "resumed" takes the place of "paused".
  VC.toast = (root, text, { tone = '', sub = '', key = '' } = {}) => {
    let host = root.querySelector(':scope > .c-toasts');
    if (!host) { host = document.createElement('div'); host.className = 'c-toasts'; host.setAttribute('role', 'status'); root.appendChild(host); }
    let t = key ? [...host.children].find((x) => x.dataset.key === key && !x.classList.contains('is-out')) : null;
    if (!t) {
      t = document.createElement('div');
      if (key) t.dataset.key = key;
      host.appendChild(t);
    }
    clearTimeout(t._out);
    t.className = `c-toast ${tone ? 'c-toast--' + tone : ''}`;
    t.innerHTML = `<span class="c-toast__mark"></span><span><span class="c-toast__t">${esc(text)}</span>${sub ? `<span class="c-toast__s">${esc(sub)}</span>` : ''}</span>`;
    t._out = setTimeout(() => {
      t.classList.add('is-out');
      setTimeout(() => t.remove(), 260);
    }, 3600);
  };

  VC.poster = (w, cls = '', vt = '') => `<img class="c-poster ${cls}" src="${SS.art.poster(w)}" alt="" decoding="async" loading="lazy"${vt ? ` style="view-transition-name:${vt}"` : ''}>`;
  VC.avatar = (id, cls = '') => {
    const a = d.A[id];
    return `<span class="c-avatar ${cls}" style="--h:${a.hue}" aria-hidden="true">${esc(F.initials(a.name))}</span>`;
  };
  VC.bar = (p, line, cls = '') => `<span class="c-bar ${cls}" style="--ink:${VC.L[line]?.ink || 'var(--c-ink)'}"><span class="c-bar__fill" style="transform:scaleX(${Math.max(0, Math.min(1, p)).toFixed(4)})"></span></span>`;
  VC.workMeta = (w) => (w.type === 'movie'
    ? [w.year, F.runtime(w.runtime), w.cert].filter(Boolean)
    : [w.endYear ? `${w.year} to ${w.endYear}` : `Since ${w.year}`, w.network, F.plural(w.seasons.length, 'season')]).join(' · ');
  VC.statusOf = (w) => {
    const j = VC.workJob(w);
    if (j?.state === 'failed') return { key: 'failed', label: 'Import failed', line: VC.jobLine(j) };
    if (j?.state === 'stalled') return { key: 'stalled', label: 'Stalled', line: 'acquire' };
    if (j?.state === 'paused') return { key: 'paused', label: 'Paused', line: 'acquire' };
    const jr = VC.journey(w);
    if (jr) return { key: 'moving', label: `${VC.jobVerb(jr.job)} ${F.pct(jr.job.progress)}`, line: VC.jobLine(jr.job), p: jr.p };
    if (w.type === 'series' && w.counts.missing) return { key: 'missing', label: `${F.plural(w.counts.missing, 'episode')} missing` };
    return { key: w.status, label: F.statusLabel[w.status] || 'Available' };
  };

  VC.timeOrDay = (date) => {
    const now = VC.clock();
    if (date.toDateString() === now.toDateString()) return F.time(date);
    if (Math.abs(now - date) < 36 * 3600e3) return `${date.toLocaleDateString('en-GB', { weekday: 'short' })} ${F.time(date)}`;
    return date.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' });
  };

  VC.searchWorks = (q, filter) => {
    q = q.trim().toLowerCase();
    const list = filter ? d.works.filter(filter) : d.works;
    if (!q) return [];
    return list
      .map((w) => {
        const t = w.title.toLowerCase();
        const score = t.startsWith(q) ? 0 : t.split(/\s+/).some((p) => p.startsWith(q)) ? 1 : t.includes(q) ? 2 : w.genres.some((g) => g.toLowerCase().startsWith(q)) ? 3 : 9;
        return { w, score };
      })
      .filter((x) => x.score < 9)
      .sort((a, b) => a.score - b.score || a.w.title.localeCompare(b.w.title))
      .map((x) => x.w);
  };

  VC.logo = (size = 20) => `<svg class="c-logo" width="${size}" height="${size}" viewBox="0 0 24 24" aria-hidden="true"><path d="M6.5 17.5L17.5 6.5" stroke="var(--c-ink)" stroke-width="3.8"/><circle cx="6.5" cy="17.5" r="6" fill="var(--c-a)"/><circle cx="17.5" cy="6.5" r="6" fill="var(--c-l)"/></svg>`;
})();
