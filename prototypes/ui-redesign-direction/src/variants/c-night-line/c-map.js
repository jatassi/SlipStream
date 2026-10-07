(function () {
  'use strict';
  const SS = window.SS;
  const d = SS.data;
  const F = SS.fmt;
  const esc = F.esc;
  const VC = window.VC;

  // Stations sit on waypoints, so a route is a slice of its line and trains follow the drawn path exactly.
  const STATIONS = {
    atlas: { x: 3, y: 5, node: 'atlas', side: 'left' },
    'ams-1': { x: 6.75, y: 2, node: 'ams-1', side: 'top' },
    vault: { x: 10, y: 5, node: 'vault', side: 'topright' },
    plex: { x: 13.5, y: 5, media: 'plex', side: 'bottom' },
    relay: { x: 17.5, y: 3, node: 'relay', side: 'top', terminus: true },
    forge: { x: 11.5, y: 8, node: 'forge', side: 'bottom' },
    den: { x: 8.5, y: 8, node: 'den', side: 'bottom' },
    accounts: { x: 3, y: 7.5, side: 'right', terminus: true },
  };
  const LINE_PTS = {
    library: { off: 4, pts: ['atlas', 'vault'] },
    deliver: { off: -4, pts: ['atlas', 'vault', 'plex', [14.75, 5], [16.75, 3], 'relay'] },
    acquire: { off: 0, pts: ['atlas', [3, 4], [5, 2], 'ams-1', [8.5, 2], [10, 3.5], 'vault'] },
    process: { off: 0, pts: ['vault', [13, 8], 'forge', 'den', [7, 8], 'vault'] },
    notify: { off: 0, pts: ['atlas', 'accounts'] },
  };
  const BOUNDS = { x0: 1.15, x1: 18.45, y0: 1.1, y1: 9.1 };
  const RADIUS = 14;
  const NS = 'http://www.w3.org/2000/svg';

  const unit = (a, b) => {
    const dx = b[0] - a[0], dy = b[1] - a[1], l = Math.hypot(dx, dy) || 1;
    return [dx / l, dy / l];
  };
  function offsetLine(pts, off) {
    if (!off) return pts.map((p) => [p[0], p[1]]);
    return pts.map((p, i) => {
      const a = i > 0 ? unit(pts[i - 1], p) : null;
      const b = i < pts.length - 1 ? unit(p, pts[i + 1]) : null;
      const n1 = a && [a[1], -a[0]], n2 = b && [b[1], -b[0]];
      if (!n1 || !n2) { const n = n1 || n2; return [p[0] + n[0] * off, p[1] + n[1] * off]; }
      const m = [n1[0] + n2[0], n1[1] + n2[1]], ml = Math.hypot(m[0], m[1]);
      if (ml < 1e-6) return [p[0] + n1[0] * off, p[1] + n1[1] * off];
      const mu = [m[0] / ml, m[1] / ml], k = off / (mu[0] * n1[0] + mu[1] * n1[1]);
      return [p[0] + mu[0] * k, p[1] + mu[1] * k];
    });
  }
  const pt = (p) => `${p[0].toFixed(2)} ${p[1].toFixed(2)}`;
  function pathD(pts) {
    let s = `M${pt(pts[0])}`;
    for (let i = 1; i < pts.length - 1; i++) {
      const p = pts[i], a = unit(pts[i - 1], p), b = unit(p, pts[i + 1]);
      const cross = a[0] * b[1] - a[1] * b[0], dot = a[0] * b[0] + a[1] * b[1];
      if (Math.abs(cross) < 1e-6) continue;
      const t = RADIUS * Math.tan(Math.acos(Math.max(-1, Math.min(1, dot))) / 2);
      s += ` L${pt([p[0] - a[0] * t, p[1] - a[1] * t])} A${RADIUS} ${RADIUS} 0 0 ${cross > 0 ? 1 : 0} ${pt([p[0] + b[0] * t, p[1] + b[1] * t])}`;
    }
    return s + ` L${pt(pts[pts.length - 1])}`;
  }

  const stationLines = (name) => Object.keys(LINE_PTS).filter((id) => LINE_PTS[id].pts.includes(name));
  const stationState = (name) => {
    const st = STATIONS[name];
    return st.node ? VC.nodeState(d.N[st.node]) : 'good';
  };
  const SUB = {
    atlas: () => 'Controller',
    plex: () => 'Media Server',
  };
  const stateWord = { delays: 'Delays', suspended: 'Suspended', paused: 'Paused' };
  function stationSub(name) {
    const s = stationState(name);
    if (s !== 'good') return { text: stateWord[s], tone: s };
    return SUB[name] ? { text: SUB[name](), tone: '' } : null;
  }
  VC.stationSub = stationSub;

  function routeOf(j) {
    if (j.kind === 'download') return { line: 'acquire', from: 'atlas', to: 'ams-1' };
    if (j.kind === 'transfer') return { line: 'acquire', from: STATIONS[j.from] ? j.from : 'ams-1', to: 'vault' };
    if (j.kind === 'import') return { line: 'library', from: 'vault', to: 'atlas' };
    if (j.kind === 'transcode' || j.kind === 'subtitles') return j.node === 'den' ? { line: 'process', from: 'forge', to: 'den' } : { line: 'process', from: 'vault', to: 'forge' };
    return null;
  }
  VC.routeOf = routeOf;
  const onMap = (j) => ['running', 'stalled', 'paused', 'failed'].includes(j.state) && routeOf(j);
  const streams = () => d.playing.map((p, i) => ({ id: 'play-' + i, kind: 'stream', work: p.work, play: p, line: 'deliver', state: 'running', progress: p.progress, route: p.remote ? { line: 'deliver', from: 'plex', to: 'relay' } : { line: 'deliver', from: 'vault', to: 'plex' } }));

  VC.nodeCard = (name) => {
    const st = STATIONS[name] || { node: name };
    if (st.media) {
      const ms = d.mediaServers[0];
      return `<div class="c-card__h"><b>Plex</b><span>Media Server</span></div>
        <div class="c-card__roles">${VC.bullet('deliver')}<span>Linked from vault</span></div>
        <div class="c-card__kv"><span>Streams</span><b class="c-num">${ms.streams}</b><span>State</span><b>Online</b></div>
        <div class="c-card__note">${d.playing.map((p) => `${esc(d.A[p.who].name.split(' ')[0])} on ${esc(p.device)}`).join(' · ')}</div>`;
    }
    if (!st.node) {
      return `<div class="c-card__h"><b>Accounts</b><span>Notify terminus</span></div>
        <div class="c-card__roles">${VC.bullet('notify')}<span>Notifying on atlas</span></div>
        <div class="c-card__note">Arrivals and failures reach ${F.plural(d.accounts.length, 'Account')} by push and email.</div>`;
    }
    const n = d.N[st.node];
    const s = VC.nodeState(n);
    const roles = n.roles.map((r) => `<span class="c-card__role">${VC.bullet(VC.roleLine[r])}${esc(r)}</span>`).join('');
    const load = s === 'suspended' ? '' : `<div class="c-card__kv">
        <span>CPU</span><b class="c-num">${F.pct(n.cpu)}</b>${n.gpu != null ? `<span>GPU</span><b class="c-num">${F.pct(n.gpu)}</b>` : `<span>Memory</span><b class="c-num">${F.pct(n.mem)}</b>`}
        ${n.temp ? `<span>Temp</span><b class="c-num">${n.temp} °C</b>` : ''}${n.down ? `<span>In</span><b class="c-num">${F.rate(n.down)}</b>` : ''}${n.up ? `<span>Out</span><b class="c-num">${F.rate(n.up)}</b>` : ''}
      </div>`;
    const disks = n.disks.map((k) => `<div class="c-card__disk"><span>${esc(k.path)}</span><span class="c-num">${F.bytes(k.used)} of ${F.bytes(k.total)}</span>${VC.bar(k.used / k.total, 'library', 'c-bar--thin')}</div>`).join('');
    return `<div class="c-card__h"><b>${esc(n.name)}</b><span class="c-tone-${s}">${n.controller ? 'Controller · ' : ''}${{ good: 'Online', delays: 'Delays', suspended: 'Suspended', paused: 'Paused' }[s]}</span></div>
      <div class="c-card__hw">${esc(n.hardware)} · ${esc(n.place)}</div>
      <div class="c-card__roles">${roles}</div>${load}${disks}
      ${n.stateNote ? `<div class="c-card__note c-tone-${s}">${esc(n.stateNote)}</div>` : ''}`;
  };
  VC.trainCard = (j) => {
    if (j.kind === 'stream') {
      const p = j.play, w = d.W[p.work];
      return `<div class="c-card__h"><b>${esc(w.title)}${p.episode ? ' · ' + F.ep(p.episode) : ''}</b>${VC.bullet('deliver')}</div>
        <div class="c-card__hw">${esc(d.A[p.who].name)} on ${esc(p.device)} · ${esc(p.mode)}${p.remote ? ' through relay' : ''}</div>
        ${VC.bar(p.progress, 'deliver')}<div class="c-card__kv"><span>Played</span><b class="c-num">${F.pct(p.progress)}</b><span>Target</span><b>${esc(VC.targetName(p.target))}</b></div>`;
    }
    const path = VC.jobPath(j).join(' to ');
    return `<div class="c-card__h"><b>${esc(VC.jobTitle(j))}${j.episode ? ' · ' + F.ep(j.episode) : ''}</b>${VC.bullet(VC.jobLine(j))}</div>
      <div class="c-card__hw">${esc(VC.jobStage(j))} · ${esc(VC.targetName(j.target))} · ${esc(path)}</div>
      ${j.state === 'failed' ? '' : VC.bar(j.progress, VC.jobLine(j))}
      <div class="c-card__kv"><span>Progress</span><b class="c-num">${F.pct(j.progress)}</b>${VC.jobRate(j) ? `<span>Rate</span><b class="c-num">${VC.jobRate(j)}</b>` : ''}${VC.jobEta(j) ? `<span>Arrives in</span><b class="c-num">${VC.jobEta(j)}</b>` : ''}</div>
      ${j.message ? `<div class="c-card__note c-tone-${j.state === 'failed' ? 'failed' : 'delays'}">${esc(j.message)}</div>` : ''}`;
  };

  VC.netMap = (host, ctx, { onStation } = {}) => {
    host.classList.add('c-map');
    host.innerHTML = `<svg class="c-map__svg" role="img" aria-label="Network map of Nodes and lines"></svg><div class="c-card c-map__card" role="tooltip" hidden></div>`;
    const svg = host.querySelector('svg');
    const card = host.querySelector('.c-map__card');
    let W = 0, H = 0, geo = null, focus = null, sticky = null, cardFor = null;
    const trains = new Map();

    function layout() {
      W = host.clientWidth; H = host.clientHeight;
      if (!W || !H) return;
      const u = Math.min((W - 28) / (BOUNDS.x1 - BOUNDS.x0), (H - 12) / (BOUNDS.y1 - BOUNDS.y0), 46);
      const ox = (W - (BOUNDS.x1 - BOUNDS.x0) * u) / 2 - BOUNDS.x0 * u;
      const oy = (H - (BOUNDS.y1 - BOUNDS.y0) * u) / 2 - BOUNDS.y0 * u;
      const P = (x, y) => [ox + x * u, oy + y * u];
      const lines = {};
      for (const [id, def] of Object.entries(LINE_PTS)) {
        const raw = def.pts.map((p) => (typeof p === 'string' ? P(STATIONS[p].x, STATIONS[p].y) : P(p[0], p[1])));
        const pts = offsetLine(raw, def.off);
        const idx = {};
        def.pts.forEach((p, i) => { if (typeof p === 'string' && !(p in idx)) idx[p] = i; });
        const named = def.pts.map((p, i) => (typeof p === 'string' ? i : -1)).filter((i) => i >= 0);
        const runs = [];
        for (let k = 0; k < named.length - 1; k++) {
          const a = named[k], b = named[k + 1];
          const off = stationState(def.pts[a]) === 'suspended' || stationState(def.pts[b]) === 'suspended';
          const last = runs[runs.length - 1];
          if (last && last.off === off) last.to = b; else runs.push({ from: a, to: b, off });
        }
        lines[id] = { id, pts, idx, runs };
      }
      geo = { u, P, lines };
      draw();
    }

    function stationPoint(name) {
      const pts = stationLines(name).map((id) => geo.lines[id].pts[geo.lines[id].idx[name]]);
      const xs = pts.map((p) => p[0]), ys = pts.map((p) => p[1]);
      return { pts, x0: Math.min(...xs), x1: Math.max(...xs), y0: Math.min(...ys), y1: Math.max(...ys), cx: (Math.min(...xs) + Math.max(...xs)) / 2, cy: (Math.min(...ys) + Math.max(...ys)) / 2 };
    }

    function draw() {
      let s = '<g class="c-map__lines">';
      for (const l of VC.LINES) {
        const g = geo.lines[l.id];
        s += `<g class="c-ln" data-line="${l.id}" style="--ink:${l.ink}">`;
        for (const r of g.runs) s += `<path class="c-ln__run${r.off ? ' is-suspended' : ''}" d="${pathD(g.pts.slice(r.from, r.to + 1))}"/>`;
        s += `<path class="c-ln__hit" d="${pathD(g.pts)}"/></g>`;
      }
      s += '</g><g class="c-map__stations">';
      for (const name of Object.keys(STATIONS)) s += stationSVG(name);
      s += '</g><g class="c-map__routes" visibility="hidden"></g><g class="c-map__trains"></g>';
      svg.setAttribute('viewBox', `0 0 ${W} ${H}`);
      svg.setAttribute('width', W);
      svg.setAttribute('height', H);
      svg.innerHTML = s;
      trains.clear();
      syncTrains();
      applyFocus();
    }

    function stationSVG(name) {
      const st = STATIONS[name];
      const ids = stationLines(name);
      const sp = stationPoint(name);
      const state = stationState(name);
      const first = geo.lines[ids[0]];
      const i = first.idx[name];
      const a = first.pts[Math.max(0, i - 1)], b = first.pts[Math.min(first.pts.length - 1, i + 1)];
      const dir = i === 0 ? unit(first.pts[0], first.pts[1]) : unit(a, b);
      let mark = '';
      if (ids.length > 1) {
        mark = `<rect class="c-st__ring" x="${(sp.x0 - 7).toFixed(1)}" y="${(sp.y0 - 7).toFixed(1)}" width="${(sp.x1 - sp.x0 + 14).toFixed(1)}" height="${(sp.y1 - sp.y0 + 14).toFixed(1)}" rx="7"/>`;
      } else if (state === 'suspended') {
        mark = `<circle class="c-st__hollow" cx="${sp.cx}" cy="${sp.cy}" r="5.5"/>`;
      } else if (st.terminus) {
        const n = [-dir[1], dir[0]];
        mark = `<line class="c-st__tick" style="--ink:${VC.L[ids[0]].ink}" x1="${sp.cx + n[0] * 8}" y1="${sp.cy + n[1] * 8}" x2="${sp.cx - n[0] * 8}" y2="${sp.cy - n[1] * 8}"/>`;
      } else {
        const out = { top: [0, -1], bottom: [0, 1], left: [-1, 0], right: [1, 0] }[st.side] || [0, -1];
        mark = `<line class="c-st__tick" style="--ink:${VC.L[ids[0]].ink}" x1="${sp.cx}" y1="${sp.cy}" x2="${sp.cx + out[0] * 9}" y2="${sp.cy + out[1] * 9}"/>`;
      }
      if (state === 'delays') mark += `<circle class="c-st__delay" cx="${sp.cx}" cy="${sp.cy}" r="11"/>`;
      if (state === 'paused') mark += `<circle class="c-st__paused" cx="${sp.cx}" cy="${sp.cy}" r="11"/>`;
      const label = st.node ? d.N[st.node].name : st.media ? 'Plex' : 'Accounts';
      const sub = stationSub(name);
      const lines = [label, sub && sub.text].filter(Boolean);
      let tx, ty, anchor;
      const halfW = (sp.x1 - sp.x0) / 2 + 7, halfH = (sp.y1 - sp.y0) / 2 + 7;
      if (st.side === 'left') { anchor = 'end'; tx = sp.cx - halfW - 9; ty = sp.cy + (lines.length > 1 ? -2 : 4.5); }
      else if (st.side === 'right') { anchor = 'start'; tx = sp.cx + halfW + 9; ty = sp.cy + (lines.length > 1 ? -2 : 4.5); }
      else if (st.side === 'top') { anchor = 'middle'; tx = sp.cx; ty = sp.cy - 17 - (lines.length - 1) * 14; }
      else if (st.side === 'topright') { anchor = 'start'; tx = sp.cx + halfW + 6; ty = sp.cy - halfH - 2; }
      else { anchor = 'middle'; tx = sp.cx; ty = sp.cy + 28; }
      const text = `<text class="c-st__label" x="${tx.toFixed(1)}" y="${ty.toFixed(1)}" text-anchor="${anchor}">${esc(label)}${sub ? `<tspan class="c-st__sub c-tone-${sub.tone}" x="${tx.toFixed(1)}" dy="14">${esc(sub.text)}</tspan>` : ''}</text>`;
      const hit = `<rect class="c-st__hit" x="${(sp.x0 - 16).toFixed(1)}" y="${(sp.y0 - 16).toFixed(1)}" width="${(sp.x1 - sp.x0 + 32).toFixed(1)}" height="${(sp.y1 - sp.y0 + 32).toFixed(1)}"/>`;
      return `<g class="c-st" data-st="${name}" data-lines="${ids.join(' ')}" data-state="${state}" tabindex="${st.node ? 0 : -1}" role="button" aria-label="${esc(label)}">${mark}${text}${hit}</g>`;
    }

    function routePath(spec) {
      const g = geo.lines[spec.line];
      const i = g.idx[spec.from], j = g.idx[spec.to];
      const pts = i < j ? g.pts.slice(i, j + 1) : g.pts.slice(j, i + 1).reverse();
      const p = document.createElementNS(NS, 'path');
      p.setAttribute('d', pathD(pts));
      svg.querySelector('.c-map__routes').appendChild(p);
      return { el: p, len: p.getTotalLength() };
    }

    function syncTrains() {
      if (!geo) return;
      const want = new Map();
      for (const j of d.jobs) if (onMap(j)) want.set(j.id, j);
      for (const s of streams()) want.set(s.id, s);
      for (const [id, t] of trains) {
        if (want.has(id) || t.leaving) continue;
        t.leaving = true;
        t.g.classList.add('is-arriving');
        setTimeout(() => { t.g.remove(); t.route.el.remove(); trains.delete(id); }, 700);
      }
      const layer = svg.querySelector('.c-map__trains');
      for (const [id, j] of want) {
        let t = trains.get(id);
        if (t && t.leaving) continue;
        const spec = j.route || routeOf(j);
        const key = spec.line + spec.from + spec.to;
        if (t && t.key !== key) { t.g.remove(); t.route.el.remove(); trains.delete(id); t = null; }
        if (!t) {
          const g = document.createElementNS(NS, 'g');
          g.setAttribute('class', 'c-train');
          g.dataset.id = id;
          g.dataset.line = spec.line;
          g.innerHTML = '<circle class="c-train__ring" r="9.5"/><rect class="c-train__car" x="-7.5" y="-3.25" width="15" height="6.5" rx="3.25"/><rect class="c-train__hit" x="-13" y="-11" width="26" height="22"/>';
          layer.appendChild(g);
          t = { g, j, key, route: routePath(spec), last: '' };
          trains.set(id, t);
        }
        t.j = j;
        t.g.dataset.state = j.state;
      }
      tick();
      applyFocus();
    }

    function tick() {
      for (const t of trains.values()) {
        if (t.leaving) { place(t, t.route.len); continue; }
        const j = t.j;
        const edge = Math.min(16, t.route.len * 0.12);
        const at = j.state === 'failed' ? Math.min(26, t.route.len * 0.2) : edge + Math.max(0, Math.min(1, j.progress || 0)) * (t.route.len - 2 * edge);
        place(t, at);
      }
    }
    function place(t, at) {
      const p = t.route.el.getPointAtLength(at);
      const q = t.route.el.getPointAtLength(Math.min(t.route.len, at + 1));
      const r = at + 1 > t.route.len ? t.route.el.getPointAtLength(Math.max(0, at - 1)) : null;
      const ang = r ? Math.atan2(p.y - r.y, p.x - r.x) : Math.atan2(q.y - p.y, q.x - p.x);
      const tf = `translate(${p.x.toFixed(2)} ${p.y.toFixed(2)}) rotate(${((ang * 180) / Math.PI).toFixed(1)})`;
      if (tf !== t.last) { t.g.setAttribute('transform', tf); t.last = tf; }
    }

    function applyFocus() {
      const f = sticky || focus;
      if (f) host.dataset.focus = f; else delete host.dataset.focus;
      svg.querySelectorAll('.c-ln').forEach((el) => el.classList.toggle('is-on', el.dataset.line === f));
      svg.querySelectorAll('.c-st').forEach((el) => el.classList.toggle('is-on', !!f && el.dataset.lines.split(' ').includes(f)));
      svg.querySelectorAll('.c-train').forEach((el) => el.classList.toggle('is-on', el.dataset.line === f));
    }

    function showCard(html, x, y) {
      card.innerHTML = html;
      card.hidden = false;
      const cw = card.offsetWidth, ch = card.offsetHeight;
      let left = x + 18, top = y - ch / 2;
      if (left + cw > W - 8) left = x - cw - 18;
      top = Math.max(8, Math.min(H - ch - 8, top));
      card.style.transform = `translate(${Math.round(left)}px, ${Math.round(top)}px)`;
    }
    function hideCard() { card.hidden = true; cardFor = null; }
    function openStation(name) {
      const sp = stationPoint(name);
      cardFor = { kind: 'st', name };
      showCard(VC.nodeCard(name), sp.x1 + 4, sp.cy);
    }
    function openTrain(id) {
      const t = trains.get(id);
      if (!t) return;
      const m = t.g.transform.baseVal.consolidate()?.matrix;
      cardFor = { kind: 'train', id };
      showCard(VC.trainCard(t.j), m ? m.e : 0, m ? m.f : 0);
    }

    ctx.listen(svg, 'pointerover', (e) => {
      const ln = e.target.closest('.c-ln__hit');
      const st = e.target.closest('.c-st');
      const tr = e.target.closest('.c-train');
      if (tr) { if (cardFor?.id !== tr.dataset.id) openTrain(tr.dataset.id); return; }
      if (st) { if (cardFor?.name !== st.dataset.st) openStation(st.dataset.st); return; }
      if (ln) { focus = ln.parentNode.dataset.line; applyFocus(); }
    });
    ctx.listen(svg, 'pointerout', (e) => {
      const to = e.relatedTarget;
      if (e.target.closest('.c-ln__hit') && !to?.closest?.('.c-ln__hit')) { focus = null; applyFocus(); }
      if ((e.target.closest('.c-st') && !to?.closest?.('.c-st')) || (e.target.closest('.c-train') && !to?.closest?.('.c-train'))) hideCard();
    });
    ctx.listen(svg, 'click', (e) => {
      const st = e.target.closest('.c-st');
      if (st && STATIONS[st.dataset.st].node) onStation?.(STATIONS[st.dataset.st].node);
    });
    ctx.listen(svg, 'keydown', (e) => {
      const st = e.target.closest('.c-st');
      if (st && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); onStation?.(STATIONS[st.dataset.st].node); }
    });
    ctx.listen(svg, 'focusin', (e) => { const st = e.target.closest('.c-st'); if (st) openStation(st.dataset.st); });
    ctx.listen(svg, 'focusout', hideCard);

    const ro = new ResizeObserver(() => {
      if (Math.abs(host.clientWidth - W) > 1 || Math.abs(host.clientHeight - H) > 1) layout();
    });
    ro.observe(host);
    layout();

    return {
      tick,
      sync: syncTrains,
      relayout: layout,
      second() {
        if (cardFor?.kind === 'st') card.innerHTML = VC.nodeCard(cardFor.name);
        if (cardFor?.kind === 'train') { const t = trains.get(cardFor.id); if (t) card.innerHTML = VC.trainCard(t.j); }
      },
      focus(line, keep) {
        if (keep) sticky = sticky === line ? null : line;
        else focus = line;
        applyFocus();
        return sticky;
      },
      destroy() { ro.disconnect(); },
    };
  };

  const ROWS = ['atlas', 'ams-1', 'vault', 'forge', 'den', 'plex', 'relay'];
  const COLS = ['acquire', 'library', 'process', 'deliver', 'notify'];
  const SPANS = { acquire: [0, 2], library: [0, 2], process: [2, 4], deliver: [0, 6] };
  const STRIP_ROUTE = { download: ['acquire', 0, 1], transfer: ['acquire', 1, 2], import: ['library', 2, 0], transcode: ['process', 2, 3], subtitles: ['process', 2, 3] };

  VC.stripMap = (host, ctx, { onStation, rowH = 58 } = {}) => {
    const top = 26, gap = 10, x0 = 22;
    const cx = (line) => x0 + COLS.indexOf(line) * gap;
    const ry = (i) => top + i * rowH;
    const H = top * 2 + (ROWS.length - 1) * rowH;
    const stops = (name) => COLS.filter((l) => stationLines(name).includes(l)).filter((l) => {
      const span = SPANS[l];
      const i = ROWS.indexOf(name);
      return l === 'notify' || (span && i >= span[0] && i <= span[1]);
    });
    host.classList.add('c-strip');
    host.style.height = H + 'px';

    function svgLines() {
      let s = '';
      for (const l of VC.LINES) {
        const span = SPANS[l.id];
        if (!span) continue;
        for (let i = span[0]; i < span[1]; i++) {
          const off = stationState(ROWS[i]) === 'suspended' || stationState(ROWS[i + 1]) === 'suspended';
          s += `<line class="c-sl${off ? ' is-suspended' : ''}" data-line="${l.id}" style="--ink:${l.ink}" x1="${cx(l.id)}" y1="${ry(i)}" x2="${cx(l.id)}" y2="${ry(i + 1)}"/>`;
        }
      }
      const nx = cx('notify');
      s += `<line class="c-sl" data-line="notify" style="--ink:${VC.L.notify.ink}" x1="${nx}" y1="${ry(0)}" x2="${nx}" y2="${ry(0) + 26}"/><line class="c-st__tick" style="--ink:${VC.L.notify.ink}" x1="${nx - 7}" y1="${ry(0) + 26}" x2="${nx + 7}" y2="${ry(0) + 26}"/>`;
      ROWS.forEach((name, i) => {
        const on = stops(name);
        const state = stationState(name);
        const y = ry(i);
        if (on.length > 1) {
          const xs = on.map(cx);
          s += `<rect class="c-st__ring" x="${Math.min(...xs) - 7}" y="${y - 7}" width="${Math.max(...xs) - Math.min(...xs) + 14}" height="14" rx="7"/>`;
        } else if (state === 'suspended') {
          s += `<circle class="c-st__hollow" cx="${cx(on[0])}" cy="${y}" r="5.5"/>`;
        } else if (i === ROWS.length - 1) {
          s += `<line class="c-st__tick" style="--ink:${VC.L[on[0]].ink}" x1="${cx(on[0]) - 8}" y1="${y}" x2="${cx(on[0]) + 8}" y2="${y}"/>`;
        } else {
          s += `<line class="c-st__tick" style="--ink:${VC.L[on[0]].ink}" x1="${cx(on[0])}" y1="${y}" x2="${cx(on[0]) - 9}" y2="${y}"/>`;
        }
        if (state === 'delays') s += `<circle class="c-st__delay" cx="${cx(on[0])}" cy="${y}" r="11"/>`;
        if (state === 'paused') s += `<circle class="c-st__paused" cx="${cx(on[0])}" cy="${y}" r="11"/>`;
      });
      return s;
    }

    function rowInfo(name) {
      const st = STATIONS[name];
      if (st.media) return { title: 'Plex', sub: `Media Server · ${F.plural(d.playing.length, 'stream')}`, tone: '' };
      const n = d.N[st.node];
      const s = VC.nodeState(n);
      if (s === 'suspended') return { title: n.name, sub: `Suspended · last seen ${F.time(n.lastSeen)}`, tone: 'suspended' };
      if (s === 'delays') return { title: n.name, sub: 'Delays · 2 h behind', tone: 'delays' };
      if (s === 'paused') return { title: n.name, sub: 'Paused', tone: 'paused' };
      const mine = d.jobs.filter((j) => j.node === n.id && j.state === 'running');
      const bits = {
        atlas: `Controller · CPU ${F.pct(n.cpu)}`,
        'ams-1': `${F.plural(mine.filter((j) => j.kind === 'download').length, 'train')} · ${F.rate(n.down)} in`,
        vault: `${F.bytes(n.disks[0].used)} of ${F.bytes(n.disks[0].total)}`,
        relay: `${esc(n.place)} · ${F.rate(n.up)} out`,
      };
      return { title: n.name, sub: bits[name] || n.place, tone: '' };
    }

    function render() {
      host.innerHTML = `<svg class="c-strip__svg" width="${x0 + gap * 4 + 16}" height="${H}" aria-hidden="true"><g>${svgLines()}</g><g class="c-strip__trains"></g></svg>
        <ol class="c-strip__rows">${ROWS.map((name, i) => {
          const info = rowInfo(name);
          return `<li><button class="c-strip__row" data-st="${name}" style="top:${ry(i) - rowH / 2}px;height:${rowH}px">
            <span class="c-strip__name">${esc(info.title)}</span>
            <span class="c-strip__sub c-tone-${info.tone}" data-sub>${info.sub}</span>
            ${VC.bullets(stops(name))}
          </button></li>`;
        }).join('')}</ol>`;
      sync();
    }
    const trainEls = new Map();
    function sync() {
      const layer = host.querySelector('.c-strip__trains');
      if (!layer) return;
      const want = new Map();
      for (const j of d.jobs) if (['running', 'stalled', 'paused', 'failed'].includes(j.state) && STRIP_ROUTE[j.kind] && j.node !== 'den') want.set(j.id, j);
      d.playing.forEach((p, i) => want.set('play-' + i, { id: 'play-' + i, kind: p.remote ? 'remote' : 'local', progress: p.progress, state: 'running' }));
      for (const [id, el] of trainEls) if (!want.has(id)) { el.classList.add('is-arriving'); setTimeout(() => el.remove(), 600); trainEls.delete(id); }
      for (const [id, j] of want) {
        let el = trainEls.get(id);
        if (!el) {
          el = document.createElementNS(NS, 'g');
          el.setAttribute('class', 'c-train');
          el.innerHTML = '<circle class="c-train__ring" r="9"/><rect class="c-train__car" x="-3.25" y="-7.5" width="6.5" height="15" rx="3.25"/>';
          layer.appendChild(el);
          trainEls.set(id, el);
        }
        el.dataset.state = j.state;
        el._j = j;
      }
      tick();
    }
    function tick() {
      for (const el of trainEls.values()) {
        const j = el._j;
        const [line, a, b] = j.kind === 'remote' ? ['deliver', 5, 6] : j.kind === 'local' ? ['deliver', 2, 5] : STRIP_ROUTE[j.kind];
        const p = Math.max(0, Math.min(1, j.progress || 0));
        const y = j.state === 'failed' ? ry(a) + Math.sign(ry(b) - ry(a)) * 15 : ry(a) + (ry(b) - ry(a)) * (0.2 + 0.6 * p);
        const tf = `translate(${cx(line)} ${y.toFixed(2)})`;
        if (el._tf !== tf) { el.setAttribute('transform', tf); el._tf = tf; }
      }
    }
    ctx.listen(host, 'click', (e) => {
      const b = e.target.closest('.c-strip__row');
      if (b) onStation?.(b.dataset.st, b);
    });
    render();
    return {
      tick,
      sync,
      render,
      second() {
        host.querySelectorAll('.c-strip__row').forEach((b) => {
          const info = rowInfo(b.dataset.st);
          const el = b.querySelector('[data-sub]');
          if (el.innerHTML !== info.sub) el.innerHTML = info.sub;
        });
      },
    };
  };
})();
