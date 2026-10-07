(function () {
  const SS = window.SS;
  const d = SS.data;
  const fmt = SS.fmt;
  const VA = (window.VA = window.VA || {});

  const icon = (name, o = {}) => SS.icon(name, { size: 18, stroke: 1.5, ...o });
  VA.icon = icon;

  VA.STAGES = ['Wanted', 'Searching', 'Downloading', 'Moving', 'Importing', 'Library'];
  const STAGE_OF = { search: 1, download: 2, transfer: 3, import: 4, transcode: 5, subtitles: 5 };
  VA.stageOf = (j) => STAGE_OF[j.kind] ?? 2;
  const LIVE = new Set(['running', 'queued', 'stalled', 'failed', 'paused']);
  VA.isLive = (j) => LIVE.has(j.state);

  const WORDS = ['No', 'One', 'Two', 'Three', 'Four', 'Five', 'Six', 'Seven', 'Eight', 'Nine', 'Ten', 'Eleven', 'Twelve'];
  VA.word = (n) => WORDS[n] || String(n);

  VA.target = (id) => d.targets.find((t) => t.id === id);
  VA.targetName = (id) => VA.target(id)?.name || '';
  VA.wantedTarget = (w) => {
    const p = d.profiles.find((x) => x.id === w.profile);
    return p.targets.find((t) => VA.target(t).kind === 'downloaded');
  };
  VA.flowKey = (j) => `${j.work}|${j.episode ? j.episode.join('.') : ''}|${j.target || ''}`;

  VA.jobTitle = (j) => {
    const w = d.W[j.work];
    return j.episode ? `${w.title} · ${fmt.ep(j.episode)}` : w.title;
  };
  VA.jobVerb = (j) => {
    if (j.state === 'failed') return j.kind === 'import' ? 'Import failed' : 'Failed';
    if (j.state === 'stalled') return 'Stalled';
    if (j.state === 'paused') return 'Paused';
    if (j.state === 'queued') return 'Queued';
    return fmt.jobVerb[j.kind] || 'Working';
  };
  VA.jobLine = (j) => {
    const t = VA.targetName(j.target);
    if (j.kind === 'transfer') return `${VA.jobVerb(j)} ${t} to ${j.to}`;
    if (j.kind === 'transcode') return `${j.state === 'running' ? 'Deriving' : VA.jobVerb(j)} ${t} from ${VA.targetName(j.from)}`;
    if (j.kind === 'subtitles') return `${j.state === 'queued' ? 'Queued' : 'Subtitles'} · ${j.step || 'English'}`;
    if (j.kind === 'search') return `Searching for ${t}`;
    return `${VA.jobVerb(j)} ${t}`;
  };
  VA.jobRate = (j) => {
    if (j.state !== 'running') return '';
    if (j.kind === 'transcode' || j.kind === 'subtitles') return `${(j.speed || 1).toFixed(1)}× realtime`;
    if (j.kind === 'search') return '';
    return fmt.rate(j.rate);
  };
  VA.jobEta = (j) => (j.state === 'running' && j.eta ? fmt.dur(j.eta) : '');

  // What the Tunnel draws: one flow per live job, and a wisp per wanted Item at the first rake.
  VA.jobFlows = () => {
    const out = [];
    for (const j of d.jobs) {
      if (!VA.isLive(j)) continue;
      out.push({ key: VA.flowKey(j), job: j, work: d.W[j.work], stage: VA.stageOf(j), progress: j.state === 'queued' ? 0 : j.progress, state: j.state });
    }
    return out;
  };
  VA.wantedFlows = () => {
    const out = [];
    const live = new Set(d.jobs.filter(VA.isLive).map(VA.flowKey));
    for (const w of d.works) {
      const t = VA.wantedTarget(w);
      if (w.type === 'movie') {
        if (w.status !== 'missing') continue;
        const key = `${w.id}||${t}`;
        if (!live.has(key)) out.push({ key, work: w, target: t, stage: 0, progress: 0, state: 'wanted' });
      } else {
        for (const s of w.seasons) for (const e of s.episodes) {
          if (e.status !== 'missing' && e.status !== 'failed') continue;
          const key = `${w.id}|${s.n}.${e.n}|${t}`;
          if (!live.has(key)) out.push({ key, work: w, episode: [s.n, e.n], target: t, stage: 0, progress: 0, state: 'wanted' });
        }
      }
    }
    return out;
  };
  VA.stageCounts = () => {
    const c = [0, 0, 0, 0, 0, 0];
    c[0] = VA.wantedFlows().length;
    for (const f of VA.jobFlows()) c[f.stage]++;
    return c;
  };

  // Separations need the Admin; everything else recedes.
  VA.needs = () => {
    const out = [];
    for (const j of d.jobs) {
      if (j.state === 'failed') out.push({ id: 'job:' + j.id, level: 'error', title: `${VA.jobTitle(j)} ${j.kind === 'import' ? 'failed to import' : 'failed'}`, detail: j.message, job: j, flow: VA.flowKey(j), action: 'Blocklist and search', at: alertAt(j.work) });
    }
    for (const n of d.nodes) if (n.state === 'offline') out.push({ id: 'node:' + n.id, level: 'error', title: `${n.name} is offline`, detail: d.alerts.find((a) => a.node === n.id)?.detail || n.stateNote, node: n, action: `Open ${n.name}`, at: n.lastSeen });
    for (const j of d.jobs) {
      if (j.state === 'stalled') out.push({ id: 'job:' + j.id, level: 'warn', title: `${VA.jobTitle(j)} has stalled`, detail: `${j.message} on ${j.node}.`, job: j, flow: VA.flowKey(j), action: 'Find another release', at: alertAt(j.work) });
    }
    for (const n of d.nodes) if (n.state === 'degraded') out.push({ id: 'node:' + n.id, level: 'warn', title: `${n.name} is behind`, detail: `${n.stateNote}.`, node: n, action: `Open ${n.name}`, at: d.alerts.find((a) => a.node === n.id)?.at });
    const pend = d.requests.filter((r) => r.state === 'pending');
    if (pend.length) out.push({ id: 'requests', level: 'ask', title: `${VA.word(pend.length)} ${pend.length === 1 ? 'request is' : 'requests are'} waiting`, detail: fmt.list(pend.map((r) => d.W[r.work].title)), action: 'Review', at: pend.reduce((a, r) => (r.at > a ? r.at : a), pend[0].at) });
    return out;
  };
  const alertAt = (work) => d.alerts.find((a) => a.work === work)?.at || SS.sim.clock;
  VA.separations = () => VA.needs().filter((n) => n.level === 'error');

  VA.sentence = () => {
    const run = d.jobs.filter((j) => j.state === 'running' && j.kind !== 'search');
    const rate = SS.sim.totals().down;
    const sep = VA.separations().length;
    return {
      count: run.length,
      lead: run.length ? `${VA.word(run.length)} ${run.length === 1 ? 'job is' : 'jobs are'} moving` : 'Nothing is moving',
      rate: run.length && rate > 0.5e6 ? `${Math.round(rate / 1e6)}\u00a0MB/s` : '',
      needs: sep ? `${VA.word(sep)} ${sep === 1 ? 'needs' : 'need'} you.` : 'Nothing needs you.',
      sep,
    };
  };

  // Arrivals stay marked until the Admin has seen them.
  VA.seen = VA.seen || new Set();
  const arrivalKey = (h) => `${h.work}|${h.episode ? h.episode.join('.') : ''}|${+h.at}`;
  VA.arrivalKey = arrivalKey;
  VA.arrivals = (hours = 12) => {
    const since = SS.sim.clock - hours * 3600e3;
    return d.history.filter((h) => (h.kind === 'imported' || h.kind === 'upgraded' || h.kind === 'derived') && h.at >= since);
  };
  for (const h of d.history) if (SS.data.now - h.at > 15 * 60e3) VA.seen.add(arrivalKey(h));

  // Requests read as a journey for Members, never as machinery.
  VA.JOURNEY = ['Requested', 'Approved', 'On its way', 'Ready to watch'];
  VA.requestFlow = (r) => d.jobs.filter((j) => VA.isLive(j) && j.work === r.work);
  VA.journey = (r) => {
    if (r.state === 'denied') return { step: 0, denied: true, label: 'Not added', detail: r.reason || '' };
    if (r.state === 'available') return { step: 3, label: 'Ready to watch', detail: r.doneAt ? `Since ${fmt.ago(r.doneAt)}` : '' };
    const jobs = VA.requestFlow(r);
    if (jobs.length || r.state === 'downloading') {
      const eta = VA.requestEta(r);
      return { step: 2, label: 'On its way', detail: eta ? `Ready in about ${fmt.dur(eta)}` : 'Getting ready', progress: VA.requestProgress(r) };
    }
    if (r.state === 'approved') return { step: 1, label: 'Approved', detail: 'Looking for the best copy' };
    return { step: 0, label: 'Waiting for approval', detail: `Requested ${fmt.ago(r.at)}` };
  };
  VA.requestProgress = (r) => {
    const j = VA.requestFlow(r)[0];
    if (!j) return 0;
    const base = { search: 0, download: 0.05, transfer: 0.86, import: 0.97 }[j.kind] ?? 0;
    const span = { search: 0.05, download: 0.81, transfer: 0.11, import: 0.03 }[j.kind] ?? 0.1;
    return Math.min(0.995, base + span * j.progress);
  };
  VA.requestEta = (r) => {
    const j = VA.requestFlow(r)[0];
    if (!j) return null;
    const size = j.size || 0;
    const after = { search: size / 30e6 + size / 96e6 + size / 700e6, download: size / 96e6 + size / 700e6, transfer: size / 700e6, import: 0 }[j.kind] ?? 0;
    return (j.eta || 0) + after;
  };

  // One clock: every animation in this direction runs on the sim's frame tick,
  // with a self-stopping rAF only when the sim is not running (still mode).
  const subs = new Set();
  let lastTick = 0, fallback = 0;
  const run = (dt) => { for (const fn of [...subs]) if (fn(dt) === false) subs.delete(fn); };
  SS.sim.on('tick', ({ dt }) => { lastTick = performance.now(); if (subs.size) run(dt); });
  const kick = () => {
    if (fallback || performance.now() - lastTick < 100) return;
    let prev = 0;
    const loop = (ts) => {
      fallback = 0;
      if (!subs.size || performance.now() - lastTick < 100) return;
      run(prev ? Math.min(0.05, (ts - prev) / 1000) : 1 / 60);
      prev = ts;
      fallback = requestAnimationFrame(loop);
    };
    fallback = requestAnimationFrame(loop);
  };
  VA.frames = (fn) => { subs.add(fn); kick(); return () => subs.delete(fn); };
  VA.reduced = () => matchMedia('(prefers-reduced-motion: reduce)').matches;

  // Critically damped spring as a CSS linear() easing, with the gesture's velocity handed off.
  VA.spring = ({ response = 0.4, v0 = 0 } = {}) => {
    const w = (2 * Math.PI) / response;
    const u = (t) => (1 + (w - v0) * t) * Math.exp(-w * t);
    let T = 0.08;
    while (T < 2.5 && (1 + Math.abs(w - v0) * T) * Math.exp(-w * T) > 0.002) T += 0.02;
    const N = 40, pts = [];
    for (let i = 0; i <= N; i++) pts.push(+(1 - u((i / N) * T)).toFixed(4));
    pts[N] = 1;
    return { easing: `linear(${pts.join(', ')})`, duration: Math.round(T * 1000) };
  };
  VA.project = (v, rate = 0.998) => ((v / 1000) * rate) / (1 - rate);

  let seq = 0;
  const groups = ['HONE', 'FRAMESTOR', 'NTb', 'KiNGDOM', 'LUMiNA'];
  const releaseFor = (w, target, episode) => {
    const base = w.title.replace(/[^A-Za-z0-9 ]/g, '').replace(/ +/g, '.');
    const g = groups[(w.title.length + (episode ? episode[1] : 0)) % groups.length];
    if (episode) return `${base}.S${String(episode[0]).padStart(2, '0')}E${String(episode[1]).padStart(2, '0')}.1080p.WEB-DL.DDP5.1.H.264-${g}`;
    return target === 'uhd' ? `${base}.${w.year}.2160p.UHD.BluRay.REMUX.HDR.HEVC.TrueHD.Atmos.7.1-${g}` : `${base}.${w.year}.1080p.BluRay.x264.DTS-HD.MA.5.1-${g}`;
  };
  const dropAlert = (pred) => { const i = d.alerts.findIndex(pred); if (i >= 0) d.alerts.splice(i, 1); };
  const remember = (h) => { d.history.unshift({ at: SS.sim.clock, node: 'atlas', ...h }); SS.sim.emit('history:new', d.history[0]); };

  VA.act = {
    search(w, { episode, target = VA.wantedTarget(w), release, size, requestedBy } = {}) {
      const label = episode ? `${w.title} · ${fmt.ep(episode)}` : w.title;
      size = size || (episode ? 2.6e9 + (episode[1] % 4) * 0.4e9 : target === 'uhd' ? 54e9 : 11.8e9);
      const j = { id: 'va' + ++seq, kind: 'search', work: w.id, episode, label, target, node: 'atlas', size: 7.2e8, progress: 0, rate: 0, state: 'running', indexer: 'Kestrel', then: { release: release || releaseFor(w, target, episode), size, requestedBy } };
      if (w.type === 'movie' && (w.status === 'missing' || w.status === 'failed')) w.status = 'downloading';
      d.jobs.push(j);
      SS.sim.emit('job:new', j);
      return j;
    },
    blocklist(j) {
      j.state = 'done';
      j.blocklisted = true;
      dropAlert((a) => a.work === j.work && a.level === 'error');
      remember({ kind: 'blocklisted', work: j.work, target: j.target, text: 'Blocklisted the sample release' });
      SS.sim.emit('job:update', j);
      return VA.act.search(d.W[j.work], { target: j.target });
    },
    findAnother(j) {
      j.state = 'done';
      j.replaced = true;
      dropAlert((a) => a.work === j.work && a.level === 'warning');
      SS.sim.emit('job:update', j);
      return VA.act.search(d.W[j.work], { target: j.target, episode: j.episode });
    },
    approve(r) {
      r.state = 'approved';
      SS.sim.emit('request:update', r);
      const w = d.W[r.work];
      if (w.type === 'movie' && w.status === 'missing') VA.act.search(w, { requestedBy: r.by });
      if (w.type === 'series') {
        const seasons = r.seasons || w.seasons.map((s) => s.n);
        for (const s of w.seasons) if (seasons.includes(s.n)) for (const e of s.episodes) if (e.status === 'missing') VA.act.search(w, { episode: [s.n, e.n], requestedBy: r.by });
      }
    },
    deny(r, reason) {
      r.state = 'denied';
      r.reason = reason || 'Not right for this library.';
      SS.sim.emit('request:update', r);
    },
    searchMissingEpisodes() {
      let n = 0;
      for (const w of d.works) if (w.type === 'series') for (const s of w.seasons) for (const e of s.episodes) {
        if (e.status !== 'missing' && e.status !== 'failed') continue;
        if (d.jobs.some((j) => VA.isLive(j) && j.work === w.id && j.episode && j.episode[0] === s.n && j.episode[1] === e.n)) continue;
        e.status = 'missing';
        VA.act.search(w, { episode: [s.n, e.n] });
        n++;
      }
      return n;
    },
    pauseNode(id, pause = true) {
      const n = d.N[id];
      n.paused = pause;
      for (const j of d.jobs) if (j.node === id && j.state === (pause ? 'running' : 'paused')) { j.state = pause ? 'paused' : 'running'; if (pause) j.rate = 0; SS.sim.emit('job:update', j); }
    },
    toggleJob(j) {
      if (j.state === 'running') { j.state = 'paused'; j.rate = 0; }
      else if (j.state === 'paused') j.state = 'running';
      SS.sim.emit('job:update', j);
    },
  };
  SS.sim.on('job:done', (j) => {
    if (j.kind !== 'search' || !j.then) return;
    const n = { id: j.id + 'd', kind: 'download', work: j.work, episode: j.episode, label: j.label, target: j.target, release: j.then.release, node: 'ams-1', size: j.then.size, progress: 0, rate: 24e6, seeds: 380 + (j.label.length * 17) % 900, peers: 40 + (j.label.length * 7) % 120, state: d.N['ams-1'].paused ? 'paused' : 'running', indexer: j.indexer, requestedBy: j.then.requestedBy };
    if (j.episode) {
      const e = d.W[j.work].seasons[j.episode[0] - 1].episodes[j.episode[1] - 1];
      e.status = 'downloading';
    }
    d.jobs.push(n);
    SS.sim.emit('job:new', n);
  });

  // Text that changes state flips in place; live numbers just update.
  VA.flip = (el, text) => {
    if (!el || el.textContent === text) return;
    el.textContent = text;
    if (VA.reduced()) { el.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 160, easing: 'ease-out' }); return; }
    el.animate([{ opacity: 0, transform: 'translateY(40%)', filter: 'blur(2px)' }, { opacity: 1, transform: 'none', filter: 'none' }], { duration: 240, easing: 'cubic-bezier(0.23, 1, 0.32, 1)' });
  };

  VA.patch = (box, items, key, render) => {
    const have = new Map([...box.children].filter((c) => c.dataset.key).map((c) => [c.dataset.key, c]));
    let prev = null;
    for (const it of items) {
      const k = key(it);
      const html = render(it);
      let el = have.get(k);
      if (el) {
        have.delete(k);
        if (el.__html !== html) {
          const tmp = document.createElement('div');
          tmp.innerHTML = html;
          const next = tmp.firstElementChild;
          next.querySelectorAll('[data-flip]').forEach((n) => {
            const old = el.querySelector(`[data-flip="${n.dataset.flip}"]`);
            if (old && old.textContent !== n.textContent) n.dataset.flipNow = '1';
          });
          next.__html = html;
          next.dataset.key = k;
          el.replaceWith(next);
          next.querySelectorAll('[data-flip-now]').forEach((n) => { const t = n.textContent; n.textContent = ''; VA.flip(n, t); delete n.dataset.flipNow; });
          el = next;
        }
      } else {
        const tmp = document.createElement('div');
        tmp.innerHTML = html;
        el = tmp.firstElementChild;
        el.dataset.key = k;
        el.__html = html;
        el.classList.add('a-enter');
      }
      const want = prev ? prev.nextElementSibling : box.firstElementChild;
      if (want !== el) box.insertBefore(el, want);
      prev = el;
    }
    for (const el of have.values()) {
      if (el.classList.contains('a-leave')) continue;
      el.classList.add('a-leave');
      el.style.height = el.offsetHeight + 'px';
      requestAnimationFrame(() => { el.style.height = '0px'; });
      setTimeout(() => el.remove(), 260);
    }
  };

  VA.toast = (root, text) => {
    let layer = root.querySelector('.a-toasts');
    if (!layer) { layer = document.createElement('div'); layer.className = 'a-toasts'; layer.setAttribute('role', 'status'); root.appendChild(layer); }
    const t = document.createElement('div');
    t.className = 'a-toast';
    t.textContent = text;
    layer.appendChild(t);
    setTimeout(() => { t.classList.add('is-out'); setTimeout(() => t.remove(), 200); }, 2600);
  };

  VA.mark = (size = 28) => `<svg class="a-mark" width="${size}" height="${size}" viewBox="0 0 28 28" fill="none" stroke-width="1.9" stroke-linecap="round" aria-hidden="true"><path stroke="currentColor" d="M8.5 6.5H10c4 0 4 5 8 5h6"/><path stroke="var(--a-sheet)" d="M3.5 11.5H10c4 0 4 5 8 5h7"/><path stroke="currentColor" d="M6 16.5h4c4 0 4 5 8 5h3"/></svg>`;
  VA.flowlines = (size = 20) => `<svg class="a-flowlines" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" aria-hidden="true"><path d="M3 7.5h12"/><path d="M6 12h15"/><path d="M3 16.5h10"/></svg>`;
  const spiral = (() => {
    const pts = [];
    for (let th = 0; th <= Math.PI * 2.4; th += 0.16) {
      const r = 7.2 * Math.exp(-0.25 * th);
      pts.push(`${(13.5 + r * Math.cos(th + 3.5)).toFixed(2)} ${(12.5 + r * Math.sin(th + 3.5) * 0.9).toFixed(2)}`);
    }
    return `M1.5 11.2C3.4 10.6 4.6 10.4 ${pts[0]}L${pts.slice(1).join('L')}`;
  })();
  VA.vortex = (size = 18) => `<svg class="a-glyph a-glyph--sep" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="${spiral}"/></svg>`;
  VA.thinning = (size = 18) => `<svg class="a-glyph a-glyph--thin" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" aria-hidden="true"><path d="M2 12.5c2.5-1.2 4.5-1.2 7 0"/><path d="M12 12.5h1.5" opacity=".7"/><path d="M17 12.5h.5" opacity=".45"/><path d="M21 12.5h.01" opacity=".3"/></svg>`;
  VA.laminar = (size = 18) => `<svg class="a-glyph" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" aria-hidden="true"><path d="M2 8.5c6-1.4 13-1.4 20 0" opacity=".5"/><path d="M2 12.5h20"/><path d="M2 16.5c6 1.4 13 1.4 20 0" opacity=".5"/></svg>`;

  // A Node is a body in the flow: the lines bend around it and its wake roughens with load.
  const LINES = [-22, -14, -7, 7, 14, 22];
  const bodyPath = (bx, by, bw, bh) => `M${bx - bw / 2} ${by}C${bx - bw / 2} ${by - bh * 0.62} ${bx - bw / 4} ${by - bh / 2} ${bx} ${by - bh / 2}C${bx + bw / 4} ${by - bh / 2} ${bx + bw / 2 - 2} ${by - bh * 0.18} ${bx + bw / 2 + 5} ${by}C${bx + bw / 2 - 2} ${by + bh * 0.18} ${bx + bw / 4} ${by + bh / 2} ${bx} ${by + bh / 2}C${bx - bw / 4} ${by + bh / 2} ${bx - bw / 2} ${by + bh * 0.62} ${bx - bw / 2} ${by}Z`;
  const linePath = (n, off, w, h, bx, by, bw, bh) => {
    const load = n.state === 'offline' ? 0 : Math.max(n.cpu, n.gpu ?? 0);
    const rough = n.state === 'degraded' ? 2.2 : 1;
    const tail = bx + bw / 2 + 4;
    let p = '';
    for (let x = 0; x <= w; x += 4) {
      const dx = (x - bx) / (bw * 0.62);
      const push = Math.sign(off) * bh * 0.62 * Math.exp(-dx * dx) * (11 / (Math.abs(off) + 4));
      const k = Math.min(1, Math.max(0, (x - tail) / 26));
      const wake = k * Math.sin((x - tail) * (0.16 + 0.05 * rough) + off * 0.7) * load * 3.6 * rough * Math.exp(-Math.abs(off) / 15) * (1 - (x - tail) / (w - tail + 40));
      p += `${p ? 'L' : 'M'}${x} ${(by + off + push + wake).toFixed(2)}`;
    }
    return p;
  };
  VA.nodeBody = (n, { w = 200, h = 60 } = {}) => {
    const bx = w * 0.36, by = h / 2, bw = Math.min(56, w * 0.24), bh = 15;
    const tput = (n.down || 0) + (n.up || 0);
    const dur = n.state === 'offline' ? 0 : Math.max(1.1, Math.min(7, 5.5 - Math.log10(1 + tput / 1e6) * 2.1));
    const lines = LINES.map((off, i) => {
      const d0 = linePath(n, off, w, h, bx, by, bw, bh);
      return `<path class="a-body__base" d="${d0}"/><path class="a-body__puff" pathLength="100" d="${d0}" style="animation-delay:${(-i * 0.37 * dur).toFixed(2)}s"/>`;
    }).join('');
    return `<svg class="a-body a-body--${n.state}" viewBox="0 0 ${w} ${h}" width="${w}" height="${h}" style="--a-dur:${dur.toFixed(2)}s" data-w="${w}" data-h="${h}" aria-hidden="true">${lines}<path class="a-body__shape" d="${bodyPath(bx, by, bw, bh)}"/></svg>`;
  };
  VA.nodeBodyUpdate = (svg, n) => {
    const w = +svg.dataset.w, h = +svg.dataset.h;
    const bx = w * 0.36, by = h / 2, bw = Math.min(56, w * 0.24), bh = 15;
    const paths = svg.querySelectorAll('.a-body__base');
    const puffs = svg.querySelectorAll('.a-body__puff');
    LINES.forEach((off, i) => { const p = linePath(n, off, w, h, bx, by, bw, bh); paths[i].setAttribute('d', p); puffs[i].setAttribute('d', p); });
  };
  VA.nodeState = (n) => {
    if (n.state === 'offline') return `Offline since ${fmt.time(n.lastSeen)}`;
    if (n.paused) return 'Paused';
    if (n.state === 'degraded') return `Behind · GPU ${n.temp} °C`;
    return `Online · ${Math.round(n.cpu * 100)}% CPU`;
  };
  VA.nodeRates = (n) => (n.state === 'offline' ? 'No flow' : `↓ ${fmt.rate(n.down || 0)}  ↑ ${fmt.rate(n.up || 0)}`);

  VA.posterImg = (w, cls = '', vt = '') => `<img class="a-poster ${cls}" src="${SS.art.poster(w)}" alt="" loading="lazy" decoding="async" draggable="false"${vt ? ` data-vt="${vt}"` : ''}>`;
  VA.statusOf = (w) => {
    if (w.type === 'series') {
      const c = w.counts;
      if (c.downloading) return { tone: 'live', text: `Downloading ${c.downloading === 1 ? '1 episode' : c.downloading + ' episodes'}` };
      if (c.missing) return { tone: 'muted', text: `${c.missing} missing` };
      return { tone: '', text: `${c.available} of ${c.total} episodes` };
    }
    const j = d.jobs.find((x) => x.work === w.id && VA.isLive(x));
    if (j?.state === 'failed') return { tone: 'sep', text: 'Import failed' };
    if (j?.state === 'stalled') return { tone: 'muted', text: 'Stalled' };
    if (j?.state === 'queued') return { tone: 'muted', text: j.kind === 'subtitles' ? 'Subtitles queued' : j.kind === 'transcode' ? 'Derive queued' : 'Queued' };
    if (j) return { tone: 'live', text: `${VA.jobVerb(j)} · ${fmt.pct(j.progress)}`, job: j };
    if (w.status === 'unreleased') return { tone: 'muted', text: w.release ? `Releases ${w.release.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })}` : 'Unreleased' };
    if (w.status === 'missing') return { tone: 'muted', text: 'Missing' };
    if (w.status === 'upgradable') return { tone: '', text: 'Upgradable' };
    if (w.status === 'failed') return { tone: 'sep', text: 'Failed' };
    return { tone: '', text: `${w.year} · ${fmt.runtime(w.runtime)}` };
  };
})();
