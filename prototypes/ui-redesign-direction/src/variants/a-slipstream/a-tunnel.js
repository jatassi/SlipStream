(function () {
  const SS = window.SS;
  const d = SS.data;
  const fmt = SS.fmt;
  const VA = window.VA;
  const esc = fmt.esc;

  const INK = [214, 226, 238];
  const SHEET = [158, 216, 255];
  const RHOD = [255, 111, 125];
  const FLUO = [198, 243, 107];
  const lerp = (a, b, k) => a + (b - a) * k;
  const mixc = (a, b, k) => [lerp(a[0], b[0], k), lerp(a[1], b[1], k), lerp(a[2], b[2], k)];
  const rgba = (c, a) => `rgba(${c[0] | 0},${c[1] | 0},${c[2] | 0},${Math.max(0, Math.min(1, a)).toFixed(3)})`;
  const hash = (s) => { let h = 2166136261; for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619); return (h >>> 0) / 4294967296; };
  const smooth = (e0, e1, x) => { const k = Math.min(1, Math.max(0, (x - e0) / (e1 - e0))); return k * k * (3 - 2 * k); };
  const ease = (k, dt) => 1 - Math.exp(-k * dt);

  const speedOf = (j) => {
    if (!j || j.state !== 'running') return j?.state === 'stalled' ? 3.5 : 0;
    if (j.kind === 'transcode' || j.kind === 'subtitles') return 38 + (j.speed || 3) * 9;
    if (j.kind === 'search') return 64;
    return 14 * Math.pow(Math.max(1, j.rate) / 1e6, 0.45);
  };
  const thickOf = (size) => 0.6 + 0.9 * Math.log10(1 + (size || 2e9) / 1e9);

  // One painter draws every filament in this direction: the Tunnel at full or compact scale, and a single job at row scale.
  function Painter({ S, step, coreLW, wisp, grain, lead = null }) {
    const P = { S, t: 0 };
    const trailLen = (F) => {
      if (F.wisp) return wisp[0] + F.seed * wisp[1];
      if (F.state === 'stalled' || F.state === 'paused') return 70 * S;
      if (F.state === 'failed') return 150 * S;
      return Math.min(400 * S, (90 + F.speed * 2.1) * S + (F.life === 'exit' ? F.exitV * 0.35 : 0));
    };
    const stream = (F, x) => Math.sin(x * 0.0062 + F.seed * 6.28) * 3.2 * S + Math.sin(x * 0.017 + F.seed * 12 + P.t * 0.32) * 1.4 * S;
    const liftAt = (F, x) => (F.sep > 0.01 ? -F.sep * 15 * S * smooth(F.x - 70 * S, F.x, x) : 0);
    const vortexAt = (F) => ({ x: F.x + 16 * S, y: F.y - 19 * S * F.sep, R: 25 * S * (0.4 + 0.6 * F.sep) });
    const span = (F) => {
      const L = trailLen(F);
      return F.wisp ? [F.x, F.x + L] : [Math.max(-20, F.x - L), F.x + 8 * S];
    };

    function strand(c, F, x0, x1, off, ph, amp, v, diff) {
      c.beginPath();
      const width = Math.max(1, x1 - x0);
      let i = 0;
      for (let x = x0; x <= x1 + 0.01; x += step, i++) {
        const u = (x - x0) / width;
        const a = x - v * P.t;
        const y = F.y + stream(F, x) + liftAt(F, x) + off * (0.55 + diff * u * u) + (Math.sin(a * 0.045 + ph) * 0.9 + Math.sin(a * 0.107 + ph * 1.7) * 0.4 + Math.sin(a * 0.021 + ph * 0.6) * 0.7) * amp * (0.3 + 0.7 * u);
        if (i) c.lineTo(x, y); else c.moveTo(x, y);
      }
    }
    const ramp = (c, x0, x1, col, a, tail = 0.25) => {
      const g = c.createLinearGradient(x0, 0, x1, 0);
      g.addColorStop(0, rgba(col, 0));
      g.addColorStop(0.45, rgba(col, a * tail));
      g.addColorStop(0.86, rgba(col, a));
      g.addColorStop(1, rgba(col, 0));
      return g;
    };
    const puff = (c, x0, x1, col, a) => {
      const g = c.createLinearGradient(x0, 0, x1, 0);
      g.addColorStop(0, rgba(col, 0));
      g.addColorStop(0.3, rgba(col, a));
      g.addColorStop(0.65, rgba(col, a * 0.7));
      g.addColorStop(1, rgba(col, 0));
      return g;
    };

    function spiral(c, F, col, a, lw, grow) {
      const V = vortexAt(F);
      const R = V.R * grow;
      const rot = P.t * 0.45;
      c.lineWidth = lw;
      c.strokeStyle = rgba(col, a);
      for (let k = 0; k < 3; k++) {
        c.beginPath();
        for (let i = 0; i <= 60; i++) {
          const th = (i / 60) * Math.PI * 4.2;
          const r = R * Math.exp(-0.17 * th) * (1 + 0.06 * Math.sin(th * 3 + k + P.t * 0.6));
          const ang = th + rot + k * 2.094;
          const px = V.x + r * Math.cos(ang), py = V.y + r * Math.sin(ang) * 0.84;
          if (i) c.lineTo(px, py); else c.moveTo(px, py);
        }
        c.stroke();
      }
    }

    // The body of the smoke: drawn soft into a low-resolution layer that is blurred as a whole.
    function glow(gx, F, col, a) {
      const [x0, x1] = span(F);
      const stalled = F.state === 'stalled' || F.state === 'paused';
      const th = F.th * S;
      const v = F.wisp ? 5 : F.speed;
      if (F.wisp) {
        gx.strokeStyle = puff(gx, x0, x1, col, a * (F.state === 'queued' ? 0.42 : 0.34));
        gx.lineWidth = 3 * th;
        strand(gx, F, x0, x1, 0, F.seed * 9, 0.4 * S, v, 0.4);
        gx.stroke();
        return;
      }
      gx.strokeStyle = ramp(gx, x0, x1, col, a * (stalled ? 0.18 : 0.6), 0.34);
      gx.lineWidth = (stalled ? 2.2 : 4) * th;
      strand(gx, F, x0, x1, 0, F.seed * 9, 0.9 * S, v, stalled ? 1.6 : 0.7);
      gx.stroke();
      if (!stalled && F.sep < 0.5) {
        const hy = F.y + stream(F, F.x);
        gx.fillStyle = rgba(col, a * 0.26);
        gx.beginPath();
        gx.ellipse(F.x - 7 * th, hy, 13 * th, 5.5 * th, 0, 0, Math.PI * 2);
        gx.fill();
      }
      for (const p of F.parts) {
        if (p.b < 0.9) continue;
        const x = x0 + p.u * (x1 - x0);
        const y = F.y + stream(F, x) + liftAt(F, x) + p.o * th * 2.2 * (0.4 + p.u);
        gx.fillStyle = rgba(col, a * 0.5 * Math.pow(p.u, 1.4));
        gx.fillRect(x - 1.5, y - 1.5, 3, 3);
      }
      if (F.sep > 0.01) spiral(gx, F, col, a * F.sep * 0.5, 5 * S, 1);
    }

    function core(cx, F, col, a) {
      const [x0, x1] = span(F);
      const stalled = F.state === 'stalled' || F.state === 'paused';
      const th = F.th * S;
      const v = F.wisp ? 5 : F.speed;
      if (!F.wisp && lead) {
        const rx = lead(F);
        if (rx < x0 - 4) { cx.strokeStyle = rgba(col, a * 0.06); cx.lineWidth = 1; cx.beginPath(); cx.moveTo(rx, F.y + stream(F, rx)); for (let x = rx; x <= x0; x += 8) cx.lineTo(x, F.y + stream(F, x)); cx.stroke(); }
      }
      const K = F.wisp || stalled ? 2 : Math.max(2, Math.min(4, Math.round(1 + F.th * 1.4)));
      const amp = (0.6 + F.th * 0.4) * S * (stalled ? 2.2 : 1);
      cx.lineWidth = coreLW;
      cx.strokeStyle = F.wisp ? puff(cx, x0, x1, col, a * 0.4) : ramp(cx, x0, x1, col, a * (stalled ? 0.38 : 0.62), 0.3);
      if (stalled) { cx.setLineDash([5 * S, 7 * S]); cx.lineDashOffset = -P.t * 3; }
      for (let i = 0; i < K; i++) {
        strand(cx, F, x0, x1, (i - (K - 1) / 2) * 0.95 * th, F.seed * 17 + i * 1.9, amp, v, stalled ? 1.4 : 0.35);
        cx.stroke();
      }
      if (stalled) cx.setLineDash([]);
      if (F.wisp) return;
      for (const p of F.parts) {
        const x = x0 + p.u * (x1 - x0);
        const y = F.y + stream(F, x) + liftAt(F, x) + p.o * th * 1.5 * (0.35 + 0.9 * p.u) + Math.sin((x - v * P.t) * 0.045 + p.ph) * S;
        cx.fillStyle = rgba(col, a * (stalled ? 0.22 : 0.36) * p.b * Math.pow(p.u, 1.2));
        cx.fillRect(x - 0.6, y - 0.6, 1.2, 1.2);
      }
      if (F.sep > 0.01) {
        spiral(cx, F, col, a * F.sep * 0.34, coreLW, 1);
        const V = vortexAt(F);
        const rot = P.t * 0.45;
        for (const p of F.vparts) {
          const r = V.R * (0.15 + p.r * 0.95);
          cx.fillStyle = rgba(col, a * F.sep * 0.55 * p.b * (0.3 + p.r));
          cx.fillRect(V.x + r * Math.cos(p.th + rot) - 0.6, V.y + r * Math.sin(p.th + rot) * 0.84 - 0.6, 1.2, 1.2);
        }
      }
    }

    const make = (key, x, size) => {
      const parts = [];
      const n = Math.round(grain[0] + thickOf(size) * grain[1]);
      for (let i = 0; i < n * 1.6; i++) parts.push({ u: Math.random(), o: (Math.random() - 0.5) * 2, b: 0.5 + Math.random() * 0.8, ph: Math.random() * 6.28 });
      const vparts = [];
      for (let i = 0; i < 26; i++) vparts.push({ th: Math.random() * 6.28, r: Math.random(), b: 0.4 + Math.random() * 0.8 });
      return { key, seed: hash(key), parts, vparts, x, vx: 0, tx: x, y: null, ty: 0, lane: null, alpha: 0, sep: 0, hov: 0, dye: 0, exitV: 0, life: 'live' };
    };
    const advance = (F, dt) => {
      const L = trailLen(F);
      if (!F.wisp) for (const p of F.parts) { p.u += (F.speed * dt * (0.85 + p.b * 0.3)) / Math.max(20, L); if (p.u > 1) p.u -= 1; }
      if (F.sep > 0.01) for (const p of F.vparts) { p.th += dt * (0.35 + 1.1 / (p.r + 0.25)) * 0.5; p.r -= dt * 0.05; if (p.r < 0.02) { p.r = 1; p.th = Math.random() * 6.28; } }
    };
    const color = (F) => {
      let col = F.state === 'failed' ? mixc(INK, RHOD, Math.max(0.7, F.sep)) : mixc(INK, SHEET, F.hov * 0.9);
      if (F.arrival) col = mixc(col, FLUO, F.dye);
      return col;
    };
    return Object.assign(P, { trailLen, stream, liftAt, vortexAt, span, spiral, glow, core, make, advance, color });
  }

  VA.Tunnel = function (host, opt = {}) {
    const compact = !!opt.compact;
    const S = compact ? 0.56 : 1;
    host.classList.add('a-tunnel');
    if (compact) host.classList.add('a-tunnel--compact');
    host.innerHTML = `<canvas class="a-tunnel__cv"></canvas><div class="a-tunnel__labels" aria-hidden="true">${VA.STAGES.map((s, i) => `<span class="a-tunnel__label" data-i="${i}">${s}<b data-n></b></span>`).join('')}</div><div class="a-tunnel__card" role="dialog" aria-live="polite"></div>`;
    host.setAttribute('role', 'group');
    const cv = host.querySelector('canvas');
    const cx = cv.getContext('2d');
    const card = host.querySelector('.a-tunnel__card');
    const labels = [...host.querySelectorAll('.a-tunnel__label')];
    const bg = document.createElement('canvas');
    const bx = bg.getContext('2d');
    const glow = document.createElement('canvas');
    const gx = glow.getContext('2d');
    const glowB = document.createElement('canvas');
    const gb = glowB.getContext('2d');
    const GS = compact ? 0.5 : 0.34;
    const blurOK = 'filter' in gx;

    let W = 0, H = 0, dpr = 1, dim = 0, dirty = true, alive = true, edge = null;
    let hoverKey = null, pinKey = null, focusKey = null, focusSep = false;
    const pad = compact ? 16 : 32;
    const top = compact ? 12 : 30;
    const bottom = compact ? 30 : 46;
    const LANES = 9;
    const laneOwner = new Array(LANES).fill(null);
    const fil = new Map();
    const ghosts = [];
    let wanted = [], wantedClock = 0;

    const colW = () => (W - pad * 2) / 6;
    const rakeX = (i) => pad + i * colW();
    const xOf = (stage, p) => rakeX(stage) + colW() * (0.07 + 0.85 * p);
    const laneY = (i) => top + (i + 0.5) * ((H - top - bottom) / LANES);
    const P = Painter({ S, step: compact ? 3 : 4, coreLW: compact ? 0.6 : 0.8, wisp: compact ? [14, 8] : [26, 18], grain: compact ? [10, 6] : [18, 12], lead: compact ? null : (F) => rakeX(Math.min(5, F.stage)) });

    const takeLane = (key) => {
      const pref = Math.floor(hash(key) * LANES);
      for (let k = 0; k < LANES; k++) for (const i of [pref + k, pref - k]) {
        const j = ((i % LANES) + LANES) % LANES;
        if (!laneOwner[j]) { laneOwner[j] = key; return j; }
      }
      return pref;
    };
    const freeLane = (F) => { if (F.lane != null && laneOwner[F.lane] === F.key) laneOwner[F.lane] = null; };

    const create = (f, x) => Object.assign(P.make(f.key, x, f.job?.size), { work: f.work, job: f.job, episode: f.episode, target: f.target, state: f.state, stage: f.stage });

    function sync(dt) {
      wantedClock -= dt;
      if (wantedClock <= 0) { wanted = VA.wantedFlows(); wantedClock = 1; }
      const flows = VA.jobFlows();
      const seen = new Set();
      const span = H - top - bottom;
      wanted.forEach((f, i) => { f.wy = top + (i + 0.5) * (span / Math.max(wanted.length, 6)); });
      for (const f of flows.concat(wanted)) {
        seen.add(f.key);
        const p = f.state === 'failed' ? Math.min(0.62, Math.max(0.3, f.progress)) : f.progress;
        const tx = f.state === 'wanted' ? rakeX(0) + 5 : f.state === 'queued' ? rakeX(f.stage) + 5 : xOf(f.stage, p);
        let F = fil.get(f.key);
        if (!F) { F = create(f, tx); fil.set(f.key, F); }
        if (F.life !== 'live') { F.life = 'live'; F.exitV = 0; F.dye = 0; F.arrival = false; }
        if (f.stage < F.stage || (F.state === 'failed' && f.state !== 'failed')) {
          ghosts.push({ ...F, life: 'ghost', g: 1 });
          F.x = tx; F.vx = 0; F.alpha = 0; F.sep = 0;
        }
        F.job = f.job; F.state = f.state; F.stage = f.stage; F.tx = tx; F.work = f.work; F.episode = f.episode || f.job?.episode; F.target = f.target || f.job?.target;
        F.wisp = f.state === 'wanted' || f.state === 'queued';
        if (f.state === 'wanted') { freeLane(F); F.lane = null; F.ty = f.wy; }
        else { if (F.lane == null) F.lane = takeLane(f.key); F.ty = laneY(F.lane); }
        if (F.y == null) F.y = F.ty;
        F.speed = speedOf(f.job);
        F.th = thickOf(f.job?.size || (f.work?.type === 'series' ? 3e9 : 20e9));
      }
      for (const F of fil.values()) {
        if (seen.has(F.key) || F.life !== 'live') continue;
        const j = F.job;
        F.life = j && j.state === 'done' && !j.blocklisted && !j.replaced && (j.kind === 'import' || j.kind === 'transcode' || j.kind === 'subtitles') ? 'exit' : 'fade';
        F.arrival = F.life === 'exit' && j.kind === 'import';
        F.exitV = Math.max(F.speed, 60);
        freeLane(F);
        if (F.key === hoverKey) hoverKey = null;
        if (F.key === pinKey) unpin();
      }
    }

    function step(dt) {
      for (const F of fil.values()) {
        const w = 6.5;
        if (F.life === 'exit') {
          F.exitV += 640 * dt;
          F.x += F.exitV * dt;
          F.dye = Math.min(1, F.dye + dt * 1.4);
          if (F.x - P.trailLen(F) > W - pad + 10) fil.delete(F.key);
        } else {
          F.vx += (w * w * (F.tx - F.x) - 2 * w * F.vx) * dt;
          F.x += F.vx * dt;
        }
        F.y += (F.ty - F.y) * ease(5, dt);
        if (F.life === 'fade') { F.alpha -= dt * 1.5; if (F.alpha <= 0) fil.delete(F.key); }
        else F.alpha = Math.min(1, F.alpha + dt * 1.6);
        F.sep += ((F.state === 'failed' ? 1 : 0) - F.sep) * ease(1.6, dt);
        const on = F.key === hoverKey || F.key === pinKey || F.key === focusKey || (focusSep && F.state === 'failed');
        F.hov += ((on ? 1 : 0) - F.hov) * ease(12, dt);
        P.advance(F, dt);
      }
      for (let i = ghosts.length - 1; i >= 0; i--) { ghosts[i].g -= dt * 0.8; if (ghosts[i].g <= 0) ghosts.splice(i, 1); }
      const anyFocus = hoverKey || pinKey || focusKey || focusSep;
      dim += ((anyFocus ? 1 : 0) - dim) * ease(10, dt);
    }

    function moving() {
      if (ghosts.length) return true;
      if (Math.abs(dim - (hoverKey || pinKey || focusKey || focusSep ? 1 : 0)) > 0.01) return true;
      for (const F of fil.values()) {
        if (F.life !== 'live' || F.alpha < 1 || F.sep > 0.01 || Math.abs(F.tx - F.x) > 0.3 || Math.abs(F.ty - F.y) > 0.3) return true;
        if (F.speed > 0 && !F.wisp) return true;
        if (Math.abs(F.hov - Math.round(F.hov)) > 0.01) return true;
      }
      return false;
    }

    function draw() {
      gx.setTransform(GS, 0, 0, GS, 0, 0);
      gx.globalCompositeOperation = 'source-over';
      gx.clearRect(0, 0, W, H);
      gx.globalCompositeOperation = 'lighter';
      for (const G of ghosts) { G.sep = G.g; P.spiral(gx, G, RHOD, G.g * 0.4, 3.2 * S, 1 + (1 - G.g) * 0.7); }
      const look = [];
      for (const F of fil.values()) {
        const k = 1 - dim * (1 - F.hov) * 0.7;
        const a = F.alpha * k * (1 + F.hov * 0.4);
        const col = P.color(F);
        look.push([F, col, a]);
        P.glow(gx, F, col, a);
      }
      gb.globalCompositeOperation = 'copy';
      if (blurOK) gb.filter = `blur(${compact ? 1.4 : 2}px)`;
      gb.drawImage(glow, 0, 0);
      gb.filter = 'none';
      cx.setTransform(dpr, 0, 0, dpr, 0, 0);
      cx.globalCompositeOperation = 'source-over';
      cx.clearRect(0, 0, W, H);
      cx.globalCompositeOperation = 'lighter';
      cx.imageSmoothingQuality = 'high';
      cx.drawImage(glowB, 0, 0, W, H);
      for (const G of ghosts) P.spiral(cx, G, RHOD, G.g * 0.3, 0.8, 1 + (1 - G.g) * 0.7);
      for (const [F, col, a] of look) P.core(cx, F, col, a);
      // Filaments live between the outer rakes: they fade in past the first and leave at the last.
      cx.globalCompositeOperation = 'destination-in';
      cx.fillStyle = edge;
      cx.fillRect(0, 0, W, H);
      cx.globalCompositeOperation = 'destination-over';
      cx.drawImage(bg, 0, 0, W, H);
      cx.globalCompositeOperation = 'source-over';
    }

    function paintBg() {
      bg.width = Math.round(W * dpr);
      bg.height = Math.round(H * dpr);
      bx.setTransform(dpr, 0, 0, dpr, 0, 0);
      const n = compact ? 8 : 15;
      for (let i = 0; i < n; i++) {
        const y0 = top - 6 + (i + 0.5) * ((H - top - bottom + 12) / n);
        bx.strokeStyle = `rgba(214,226,238,${(0.014 + 0.012 * ((i * 7) % 3) / 2).toFixed(3)})`;
        bx.lineWidth = 1;
        bx.beginPath();
        for (let x = 0; x <= W; x += 8) { const y = y0 + Math.sin(x * 0.004 + i * 0.9) * 2.2 * S; x ? bx.lineTo(x, y) : bx.moveTo(x, y); }
        bx.stroke();
      }
      for (let i = 0; i < 6; i++) {
        const x = Math.round(rakeX(i)) + 0.5;
        const g = bx.createLinearGradient(0, top - 14, 0, H - bottom + 8);
        g.addColorStop(0, 'rgba(214,226,238,0)');
        g.addColorStop(0.5, 'rgba(214,226,238,0.11)');
        g.addColorStop(1, 'rgba(214,226,238,0.03)');
        bx.strokeStyle = g;
        bx.beginPath(); bx.moveTo(x, top - 14); bx.lineTo(x, H - bottom + 8); bx.stroke();
        bx.fillStyle = 'rgba(214,226,238,0.2)';
        for (let l = 0; l < LANES; l++) bx.fillRect(x - 1, laneY(l) - 0.5, 2, 1);
      }
    }

    function layout() {
      const r = host.getBoundingClientRect();
      const nw = host.clientWidth, nh = host.clientHeight;
      if (!nw || !nh) return;
      const scale = r.width / nw || 1;
      dpr = Math.min(2, (window.devicePixelRatio || 1) * scale);
      W = nw; H = nh;
      cv.width = Math.round(W * dpr); cv.height = Math.round(H * dpr);
      glow.width = glowB.width = Math.ceil(W * GS);
      glow.height = glowB.height = Math.ceil(H * GS);
      cv.style.width = W + 'px'; cv.style.height = H + 'px';
      labels.forEach((el, i) => { el.style.left = rakeX(i) + 'px'; });
      const at = (x) => Math.min(1, Math.max(0, x / W));
      edge = cx.createLinearGradient(0, 0, W, 0);
      edge.addColorStop(at(pad - 18), 'rgba(0,0,0,0)');
      edge.addColorStop(at(pad + 6), '#000');
      edge.addColorStop(at(W - pad - 22), '#000');
      edge.addColorStop(at(W - pad + 4), 'rgba(0,0,0,0)');
      paintBg();
      wantedClock = 0;
      sync(0);
      for (const F of fil.values()) { F.x = F.tx; F.y = F.ty; }
      dirty = true;
      draw();
    }

    // Interaction: hover reads a job, a click opens it; the eddy pins its card so it can be resolved here.
    const localPoint = (e) => {
      const r = cv.getBoundingClientRect();
      const s = r.width / W || 1;
      return [(e.clientX - r.left) / s, (e.clientY - r.top) / s];
    };
    function pick(px, py) {
      let best = null, bd = compact ? 16 : 12;
      for (const F of fil.values()) {
        if (F.life !== 'live' || F.alpha < 0.3) continue;
        const L = P.trailLen(F);
        const x0 = F.wisp ? F.x - 4 : F.x - L, x1 = F.wisp ? F.x + L : F.x;
        let dd = Infinity;
        if (F.sep > 0.4) { const V = P.vortexAt(F); dd = Math.max(0, Math.hypot(px - V.x, py - V.y) - V.R); }
        if (px >= x0 && px <= x1 + 10) dd = Math.min(dd, Math.abs(py - (F.y + P.stream(F, px) + P.liftAt(F, px))) + (px > x1 ? (px - x1) * 0.5 : 0));
        else dd = Math.min(dd, Math.hypot(px - x1, py - F.y));
        if (!F.wisp) dd -= 2;
        if (dd < bd) { bd = dd; best = F; }
      }
      return best;
    }

    function cardHTML(F) {
      const w = F.work;
      const title = F.job ? VA.jobTitle(F.job) : F.episode ? `${w.title} · ${fmt.ep(F.episode)}` : w.title;
      if (F.state === 'wanted') {
        const lim = d.indexers.find((i) => i.state === 'limited');
        return `<div class="a-card__title">${esc(title)}</div><div class="a-card__line">Missing ${esc(VA.targetName(F.target))} · waiting for a search</div>${lim ? `<div class="a-card__meta">${esc(lim.name)} is ${esc(lim.note.toLowerCase())}</div>` : ''}${F.key === pinKey ? `<div class="a-card__actions"><button class="a-btn a-btn--primary a-btn--sm" data-card="search">Search now</button><button class="a-btn a-btn--sm" data-card="work">Open</button></div>` : ''}`;
      }
      const j = F.job;
      const note = VA.jobNote(j);
      const stats = [];
      if (j.state === 'running') {
        const rate = VA.jobRate(j);
        if (rate) stats.push(['Rate', rate, 'rate']);
        if (j.eta) stats.push(['Left', VA.jobEta(j), 'eta']);
      }
      stats.push(['Node', j.node, '']);
      let actions = '';
      if (F.key === pinKey || j.state === 'failed') {
        if (j.state === 'failed') actions = `<button class="a-btn a-btn--primary a-btn--sm" data-card="blocklist">Blocklist and search</button><button class="a-btn a-btn--sm" data-card="open">Details</button>`;
        else if (j.state === 'stalled') actions = `<button class="a-btn a-btn--primary a-btn--sm" data-card="another">Find another release</button><button class="a-btn a-btn--sm" data-card="open">Details</button>`;
        else actions = `<button class="a-btn a-btn--sm" data-card="open">Open in Activity</button>`;
      }
      const pct = j.state === 'failed' ? '' : `<span class="a-num" data-c="pct">${fmt.pct(j.progress)}</span>`;
      return `<div class="a-card__title">${esc(title)}</div>
        <div class="a-card__line ${j.state === 'failed' ? 'is-sep' : ''}"><span data-c="line">${esc(VA.jobLine(j))}</span>${pct}</div>
        ${j.release ? `<div class="a-card__mono">${esc(j.release)}</div>` : note ? `<div class="a-card__note">${esc(note)}</div>` : ''}
        ${j.message ? `<div class="a-card__msg">${esc(j.message)}</div>` : ''}
        <dl class="a-card__stats">${stats.map(([k, v, c]) => `<div><dt>${k}</dt><dd class="a-num" ${c ? `data-c="${c}"` : ''}>${esc(v)}</dd></div>`).join('')}</dl>
        ${actions && (F.key === pinKey) ? `<div class="a-card__actions">${actions}</div>` : ''}`;
    }
    let cardFor = null;
    function showCard(F) {
      if (!F) { card.classList.remove('is-on'); cardFor = null; return; }
      const sig = `${F.key}|${F.state}|${F.job?.id || ''}|${F.key === pinKey}`;
      if (card.dataset.sig !== sig) { card.innerHTML = cardHTML(F); card.dataset.sig = sig; }
      cardFor = F.key;
      card.classList.toggle('is-pinned', F.key === pinKey);
      placeCard(F);
      card.classList.add('is-on');
    }
    function placeCard(F) {
      const cw = card.offsetWidth || 280, ch = card.offsetHeight || 120;
      const L = P.trailLen(F);
      let ax = F.wisp ? F.x + L : F.x, ay = F.y;
      if (F.sep > 0.4) { const V = P.vortexAt(F); ax = V.x + V.R; ay = V.y; }
      let left = ax + 16, topY = ay - 26;
      if (compact) { left = Math.min(Math.max(8, ax - cw / 2), W - cw - 8); topY = ay > H / 2 ? ay - ch - 14 : ay + 14; }
      else if (left + cw > W - 12) left = ax - cw - 16 - (F.wisp ? L : 0);
      topY = Math.min(Math.max(6, topY), H - ch - 6);
      card.style.transform = `translate(${Math.round(left)}px, ${Math.round(topY)}px)`;
    }
    function refreshCard() {
      if (!cardFor) return;
      const F = fil.get(cardFor);
      if (!F || F.life !== 'live') { showCard(null); return; }
      const j = F.job;
      if (!j) return;
      if (card.dataset.sig !== `${F.key}|${F.state}|${j.id}|${F.key === pinKey}`) { showCard(F); return; }
      const set = (c, v) => { const el = card.querySelector(`[data-c="${c}"]`); if (el && el.textContent !== v) el.textContent = v; };
      set('pct', fmt.pct(j.progress));
      set('rate', VA.jobRate(j));
      set('eta', VA.jobEta(j));
      set('line', VA.jobLine(j));
    }
    function unpin() { pinKey = null; hoverKey = null; card.classList.remove('is-pinned'); showCard(null); dirty = true; }
    function pin(F) {
      pinKey = F.key; hoverKey = F.key; showCard(F); dirty = true;
      card.querySelector('.a-btn--primary')?.focus({ preventScroll: true });
    }

    const onMove = (e) => {
      if (pinKey) return;
      const [px, py] = localPoint(e);
      const F = pick(px, py);
      const k = F ? F.key : null;
      cv.style.cursor = F ? 'pointer' : '';
      if (k !== hoverKey) { hoverKey = k; showCard(F); dirty = true; }
      else if (F) placeCard(F);
    };
    const onLeave = () => { if (pinKey) return; hoverKey = null; showCard(null); cv.style.cursor = ''; dirty = true; };
    const onClick = (e) => {
      const [px, py] = localPoint(e);
      const F = pick(px, py);
      if (!F) { if (pinKey) unpin(); return; }
      const touch = e.pointerType === 'touch' || compact;
      if (F.state === 'failed' || F.state === 'stalled' || F.state === 'wanted' || (touch && pinKey !== F.key)) { pinKey = F.key; hoverKey = F.key; showCard(F); dirty = true; return; }
      opt.onOpen?.(F);
    };
    const onCard = (e) => {
      const b = e.target.closest('[data-card]');
      if (!b) return;
      const F = fil.get(cardFor);
      if (!F) return;
      const a = b.dataset.card;
      if (a === 'blocklist') { b.classList.add('is-loading'); VA.act.blocklist(F.job); opt.onResolve?.('Blocklisted the sample. Searching Kestrel for another release.'); unpin(); }
      else if (a === 'another') { VA.act.findAnother(F.job); opt.onResolve?.('Looking for a release with seeders.'); unpin(); }
      else if (a === 'search') { VA.act.search(F.work, { episode: F.episode, target: F.target }); opt.onResolve?.(`Searching for ${F.work.title}.`); unpin(); }
      else if (a === 'work') opt.onOpenWork?.(F.work);
      else opt.onOpen?.(F);
    };
    const onKey = (e) => { if (e.key === 'Escape' && pinKey) unpin(); };
    cv.addEventListener('pointermove', onMove);
    cv.addEventListener('pointerleave', onLeave);
    cv.addEventListener('click', onClick);
    card.addEventListener('click', onCard);
    window.addEventListener('keydown', onKey);
    const ro = new ResizeObserver(layout);
    ro.observe(host);

    let acc = 0;
    const tick = (dt) => {
      if (!alive) return false;
      if (document.hidden || !W) return;
      const reduced = VA.reduced();
      sync(dt);
      acc += dt;
      if (reduced) {
        for (const F of fil.values()) { F.x = F.tx; F.y = F.ty; F.alpha = F.life === 'live' ? 1 : 0; F.sep = F.state === 'failed' ? 1 : 0; const on = F.key === hoverKey || F.key === pinKey || F.key === focusKey || (focusSep && F.state === 'failed'); F.hov = on ? 1 : 0; if (F.life !== 'live') fil.delete(F.key); }
        ghosts.length = 0;
        dim = hoverKey || pinKey || focusKey || focusSep ? 1 : 0;
        if (dirty || acc > 1) { draw(); dirty = false; acc = 0; refreshCard(); }
        return;
      }
      const go = moving();
      if (!go && !dirty) { if (acc > 1) { acc = 0; refreshCard(); } return; }
      P.t += dt;
      step(dt);
      draw();
      dirty = false;
      if (acc > 1) { acc = 0; refreshCard(); }
    };
    const offFrames = VA.frames(tick);
    const offSecond = SS.sim.on('second', () => {
      const c = VA.stageCounts();
      labels.forEach((el, i) => { const b = el.querySelector('[data-n]'); const v = c[i] ? String(c[i]) : ''; if (b.textContent !== v) b.textContent = v; });
      host.setAttribute('aria-label', `Flow: ${VA.STAGES.map((s, i) => `${s} ${c[i]}`).join(', ')}`);
    });
    requestAnimationFrame(() => { layout(); const c = VA.stageCounts(); labels.forEach((el, i) => { el.querySelector('[data-n]').textContent = c[i] ? String(c[i]) : ''; }); });

    return {
      focus(key) { focusKey = key; dirty = true; },
      focusSeparations(on) { focusSep = on; dirty = true; },
      pin(key) {
        const F = fil.get(key);
        if (!F || F.life !== 'live') return false;
        pin(F);
        return true;
      },
      pinSeparation() {
        const F = [...fil.values()].find((x) => x.state === 'failed' && x.life === 'live');
        if (!F) return false;
        pin(F);
        return true;
      },
      destroy() { alive = false; offFrames(); offSecond(); ro.disconnect(); window.removeEventListener('keydown', onKey); },
    };
  };

  // Row scale: one filament per job row, poster or request, drawn by the Tunnel's painter on one frame loop that rests when nothing moves.
  // A canvas opts in with class a-fil and data-fil="job:<id>", "work:<id>" or "req:<id>".
  VA.Filaments = function (root) {
    const painters = new Map();
    const painter = (S) => {
      if (!painters.has(S)) painters.set(S, Painter({ S, step: 2, coreLW: 0.6, wisp: [48 * S, 28 * S], grain: [8, 5] }));
      return painters.get(S);
    };
    const glow = document.createElement('canvas');
    const gx = glow.getContext('2d');
    const glowB = document.createElement('canvas');
    const gb = glowB.getContext('2d');
    const GS = 0.5;
    const blurOK = 'filter' in gx;
    const fil = new Map();
    const bound = new Map();
    let hovKey = null, sub = null, alive = true, queued = false;

    const source = (key) => {
      const [kind, id] = key.split(':');
      if (kind === 'job') { const j = d.jobs.find((x) => x.id === id); return j && VA.isLive(j) ? { job: j, p: j.progress } : null; }
      if (kind === 'work') { const j = VA.statusOf(d.W[id]).job; return j ? { job: j, p: j.progress } : null; }
      if (kind === 'req') {
        const r = d.requests.find((x) => x.id === id);
        if (!r) return null;
        if (r.state === 'pending') return { wanted: true };
        const j = VA.requestFlow(r)[0];
        return j ? { job: j, p: VA.requestProgress(r) } : null;
      }
      return null;
    };

    function sync(B) {
      const src = source(B.key);
      let F = fil.get(B.key);
      if (!src) { if (F) F.gone = true; return F; }
      const state = src.wanted ? 'wanted' : src.job.state;
      const idle = state === 'wanted' || state === 'queued';
      const p = state === 'failed' ? Math.min(0.66, Math.max(0.34, src.p)) : src.p;
      const tx = idle ? B.pad : B.pad + (B.w - B.pad * 2) * p;
      if (!F) { F = B.P.make(B.key, tx, src.job?.size); F.alpha = 1; F.clock = 0; F.w = B.w; fil.set(B.key, F); }
      if (F.w !== B.w) { F.x = F.x * (B.w / (F.w || B.w)); F.w = B.w; }
      Object.assign(F, { gone: false, state, wisp: idle, tx, job: src.job, speed: speedOf(src.job), th: thickOf(src.job?.size || 20e9), sep: state === 'failed' ? 1 : 0 });
      F.y = F.ty = B.h * B.yk;
      return F;
    }

    function paint(B, F) {
      const { w: W, h: H, dpr, cx, P } = B;
      const gw = Math.ceil(W * GS), gh = Math.ceil(H * GS);
      if (glow.width < gw || glow.height < gh) { glow.width = glowB.width = Math.max(glow.width, gw); glow.height = glowB.height = Math.max(glow.height, gh); }
      P.t = F.clock;
      const col = P.color(F);
      const a = F.alpha * (1.05 + F.hov * 0.3);
      gx.setTransform(1, 0, 0, 1, 0, 0);
      gx.globalCompositeOperation = 'source-over';
      gx.clearRect(0, 0, glow.width, glow.height);
      gx.setTransform(GS, 0, 0, GS, 0, 0);
      gx.globalCompositeOperation = 'lighter';
      P.glow(gx, F, col, a);
      gb.globalCompositeOperation = 'copy';
      if (blurOK) gb.filter = 'blur(1.1px)';
      gb.drawImage(glow, 0, 0);
      gb.filter = 'none';
      cx.setTransform(dpr, 0, 0, dpr, 0, 0);
      cx.globalCompositeOperation = 'source-over';
      cx.clearRect(0, 0, W, H);
      if (B.track && F.state !== 'failed') {
        const from = F.wisp ? F.x + P.trailLen(F) + 4 : F.x + 6;
        cx.save();
        cx.strokeStyle = 'rgba(214,226,238,0.2)';
        cx.lineWidth = 1;
        cx.lineCap = 'round';
        cx.setLineDash([0.5, 4.5]);
        cx.beginPath(); cx.moveTo(from, F.ty); cx.lineTo(W - B.pad, F.ty); cx.stroke();
        cx.restore();
        cx.fillStyle = 'rgba(214,226,238,0.22)';
        cx.fillRect(Math.round(W - B.pad) - 0.5, F.ty - 4, 1, 8);
      }
      cx.globalCompositeOperation = 'lighter';
      cx.drawImage(glowB, 0, 0, gw, gh, 0, 0, gw / GS, gh / GS);
      P.core(cx, F, col, a);
      cx.globalCompositeOperation = 'destination-in';
      cx.fillStyle = B.mask;
      cx.fillRect(0, 0, W, H);
      cx.globalCompositeOperation = 'source-over';
      B.dirty = false;
    }

    function tick(dt) {
      if (!alive) { sub = null; return false; }
      if (document.hidden) return;
      const reduced = VA.reduced();
      let busy = false;
      for (const [c, B] of bound) {
        if (!c.isConnected) { unbind(c); continue; }
        if (!B.w) continue;
        const F = sync(B);
        if (!F) continue;
        const hv = F.key === hovKey ? 1 : 0;
        if (reduced) {
          const moved = Math.abs(F.tx - F.x) > 0.25 || F.hov !== hv || (F.gone && F.alpha > 0);
          F.x = F.tx; F.vx = 0; F.hov = hv;
          if (F.gone) F.alpha = 0;
          if (B.vis && (moved || B.dirty)) paint(B, F);
          continue;
        }
        const run = F.state === 'running' && !F.gone;
        const settle = Math.abs(F.tx - F.x) > 0.25 || Math.abs(F.vx) > 0.5 || Math.abs(F.hov - hv) > 0.01 || (F.gone && F.alpha > 0);
        if (!B.vis || (!run && !settle && !B.dirty)) continue;
        busy = busy || run || settle;
        if (run) { F.clock += dt; B.P.t = F.clock; B.P.advance(F, dt); }
        F.vx += (42 * (F.tx - F.x) - 13 * F.vx) * dt;
        F.x += F.vx * dt;
        F.hov += (hv - F.hov) * ease(12, dt);
        if (F.gone) F.alpha = Math.max(0, F.alpha - dt * 2.5);
        paint(B, F);
      }
      if (!busy) { sub = null; return false; }
    }
    const wake = () => { if (!sub && alive) sub = VA.frames(tick); };

    const maskFor = (B) => {
      const m = B.cx.createLinearGradient(0, 0, B.w, 0);
      m.addColorStop(0, 'rgba(0,0,0,0)');
      m.addColorStop(Math.min(0.3, (B.pad * 1.2) / B.w), '#000');
      m.addColorStop(1 - Math.min(0.1, 3 / B.w), '#000');
      m.addColorStop(1, 'rgba(0,0,0,0)');
      return m;
    };
    function size(c) {
      const B = bound.get(c);
      if (!B) return;
      const w = c.clientWidth, h = c.clientHeight;
      if (!w || !h) return;
      const dpr = Math.min(2, (window.devicePixelRatio || 1) * (c.getBoundingClientRect().width / w || 1));
      if (w === B.w && h === B.h && dpr === B.dpr) return;
      Object.assign(B, { w, h, dpr });
      c.width = Math.round(w * dpr);
      c.height = Math.round(h * dpr);
      B.mask = maskFor(B);
      const F = sync(B);
      if (F) paint(B, F);
      wake();
    }
    const ro = new ResizeObserver((es) => es.forEach((e) => size(e.target)));
    const io = new IntersectionObserver((es) => es.forEach((e) => { const B = bound.get(e.target); if (B) { B.vis = e.isIntersecting; if (B.vis) wake(); } }));
    function bind(c) {
      bound.set(c, { key: c.dataset.fil, P: painter(+(c.dataset.filS || 0.42)), w: 0, h: 0, dpr: 1, cx: c.getContext('2d'), vis: false, dirty: true, yk: +(c.dataset.filY || 0.6), pad: +(c.dataset.filPad || 8), track: c.dataset.filTrack !== 'off', mask: null });
      ro.observe(c);
      io.observe(c);
    }
    function unbind(c) { bound.delete(c); ro.unobserve(c); io.unobserve(c); }
    const scan = () => {
      queued = false;
      if (!alive) return;
      for (const c of root.querySelectorAll('canvas.a-fil')) if (!bound.has(c)) bind(c);
      for (const c of [...bound.keys()]) if (!c.isConnected) unbind(c);
      const keys = new Set([...bound.values()].map((B) => B.key));
      for (const k of fil.keys()) if (!keys.has(k)) fil.delete(k);
      wake();
    };
    const mo = new MutationObserver(() => { if (!queued) { queued = true; queueMicrotask(scan); } });
    mo.observe(root, { childList: true, subtree: true });

    const over = (e) => {
      if (e.pointerType !== 'mouse') return;
      const host = e.target.closest?.('.a-job, .a-pjob, .a-tile, .a-req, .a-work__status');
      const k = host?.querySelector('canvas.a-fil')?.dataset.fil || null;
      if (k !== hovKey) { hovKey = k; wake(); }
    };
    const out = () => { if (hovKey) { hovKey = null; wake(); } };
    root.addEventListener('pointerover', over);
    root.addEventListener('pointerleave', out);
    const offSecond = SS.sim.on('second', wake);
    const offUpdate = SS.sim.on('job:update', wake);
    scan();

    return {
      destroy() {
        alive = false;
        mo.disconnect(); ro.disconnect(); io.disconnect();
        offSecond(); offUpdate();
        root.removeEventListener('pointerover', over);
        root.removeEventListener('pointerleave', out);
      },
    };
  };
})();
