// PROTOTYPE. Live simulation over SS.data: jobs advance, rates breathe, Nodes report load.
// Variants subscribe with SS.sim.on(event, fn) and read SS.data directly.
(function () {
  const SS = window.SS;
  const d = SS.data;
  const listeners = {};
  const on = (ev, fn) => ((listeners[ev] ||= new Set()).add(fn), () => listeners[ev].delete(fn));
  const emit = (ev, payload) => listeners[ev]?.forEach((fn) => fn(payload));

  // One simulated second is SPEED real seconds of work, so progress is visible in a screencast.
  const SPEED = 9;
  let clock = d.now.getTime();
  let running = false, last = 0, raf = 0, acc = 0;

  const baseRate = Object.fromEntries(d.jobs.map((j) => [j.id, j.rate]));
  let t = 0;
  const wobble = (id, k) => 1 + 0.08 * Math.sin(t * 0.9 + k) + 0.05 * Math.sin(t * 2.3 + k * 1.7);

  function nodeLoad() {
    for (const n of d.nodes) {
      if (n.state === 'offline') continue;
      const mine = d.jobs.filter((j) => j.node === n.id && j.state === 'running');
      n.down = mine.filter((j) => j.kind === 'download').reduce((s, j) => s + j.rate, 0);
      n.up = n.id === 'ams-1' ? 31.4e6 * wobble(n.id, 3) : n.id === 'relay' ? 6.2e6 * wobble(n.id, 5) : n.id === 'vault' ? 12.8e6 * wobble(n.id, 7) : 0.2e6;
      if (n.id === 'vault') n.down = mine.filter((j) => j.kind === 'transfer').reduce((s, j) => s + j.rate, 0);
      const k = n.id.length;
      n.cpu = Math.min(0.98, Math.max(0.02, (n.cpuBase ??= n.cpu) * wobble(n.id, k)));
      if (n.gpu != null) n.gpu = Math.min(0.99, (n.gpuBase ??= n.gpu) * wobble(n.id, k + 1));
    }
  }

  function complete(j) {
    j.progress = 1;
    j.rate = 0;
    j.state = 'done';
    const w = d.W[j.work];
    emit('job:done', j);
    if (j.kind === 'download') {
      const next = { id: j.id + 't', kind: 'transfer', work: j.work, episode: j.episode, label: j.label, target: j.target, from: j.node, to: 'vault', node: 'vault', size: j.size, progress: 0, rate: 96e6, streams: 16, state: 'running' };
      baseRate[next.id] = next.rate;
      d.jobs.push(next);
      emit('job:new', next);
    } else if (j.kind === 'transfer') {
      const next = { id: j.id + 'i', kind: 'import', work: j.work, episode: j.episode, label: j.label, target: j.target, node: 'vault', size: j.size, progress: 0, rate: 700e6, state: 'running', step: 'Verifying chunks' };
      baseRate[next.id] = next.rate;
      d.jobs.push(next);
      emit('job:new', next);
    } else if (j.kind === 'import') {
      if (w.type === 'movie') {
        w.coverage[j.target] = 'met';
        if (Object.values(w.coverage).every((c) => c === 'met' || c === 'unwanted' || c === 'waiting')) w.status = 'available';
      } else if (j.episode) {
        const e = w.seasons[j.episode[0] - 1].episodes[j.episode[1] - 1];
        e.status = 'available';
        w.counts.downloading = Math.max(0, w.counts.downloading - 1);
        w.counts.available++;
      }
      d.history.unshift({ at: new Date(clock), kind: 'imported', work: j.work, episode: j.episode, target: j.target, text: `Imported in ${d.targets.find((t) => t.id === j.target).name}`, node: 'vault' });
      const req = d.requests.find((r) => r.work === j.work && r.state !== 'available' && r.state !== 'denied');
      if (req) { req.state = 'available'; req.doneAt = new Date(clock); emit('request:update', req); }
      emit('work:update', w);
      emit('history:new', d.history[0]);
    }
    // Keep the screen alive: a quiet queue starts the next waiting job.
    const queued = d.jobs.find((x) => x.state === 'queued' && x.node === j.node);
    if (queued) { queued.state = 'running'; queued.rate = queued.kind === 'transcode' ? 0 : 20e6; emit('job:update', queued); }
  }

  function step(dt) {
    t += dt;
    clock += dt * 1000;
    for (const j of d.jobs) {
      if (j.state !== 'running') continue;
      if (j.kind === 'transcode' || j.kind === 'subtitles') {
        j.speed = (j.speed || 3.2) * (0.995 + Math.random() * 0.01);
        j.progress = Math.min(1, j.progress + dt * SPEED * 0.0009 * (j.speed || 1));
        j.eta = ((1 - j.progress) * (d.W[j.work].runtime || 50) * 60) / (j.speed || 1);
      } else {
        j.rate = (baseRate[j.id] || 20e6) * wobble(j.id, j.id.charCodeAt(1));
        j.progress = Math.min(1, j.progress + (j.rate * dt * SPEED) / j.size);
        j.eta = j.rate ? ((1 - j.progress) * j.size) / j.rate : null;
      }
      if (j.progress >= 1) complete(j);
    }
    nodeLoad();
    emit('tick', { dt, clock: new Date(clock) });
  }

  function loop(ts) {
    if (!running) return;
    const dt = Math.min(0.1, (ts - (last || ts)) / 1000);
    last = ts;
    acc += dt;
    step(dt);
    // Coarse listeners (text that changes once a second) fire on 'second'.
    if (acc >= 1) { acc = 0; emit('second', { clock: new Date(clock) }); }
    raf = requestAnimationFrame(loop);
  }

  for (const j of d.jobs) if (j.size && j.rate) j.eta = ((1 - j.progress) * j.size) / j.rate;
  nodeLoad();
  SS.sim = {
    on, emit,
    start() { if (running) return; running = true; last = 0; raf = requestAnimationFrame(loop); },
    stop() { running = false; cancelAnimationFrame(raf); },
    get clock() { return new Date(clock); },
    speed: SPEED,
    active: () => d.jobs.filter((j) => j.state === 'running'),
    attention: () => d.alerts.filter((a) => a.level !== 'info'),
    totals() {
      const run = d.jobs.filter((j) => j.state === 'running');
      return {
        down: run.filter((j) => j.kind === 'download').reduce((s, j) => s + j.rate, 0),
        up: d.nodes.reduce((s, n) => s + (n.up || 0), 0),
        active: run.length,
        queued: d.jobs.filter((j) => j.state === 'queued').length,
        failed: d.jobs.filter((j) => j.state === 'failed' || j.state === 'stalled').length,
      };
    },
  };
})();
