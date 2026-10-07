// PROTOTYPE. The prop department: procedural one-sheets, backdrops and stills for the
// synthetic Works, drawn on canvas so nothing copyrighted ships. Every image is invented.
(function () {
  const SS = window.SS;
  const d = SS.data;

  const FONTS = {
    bebas: { family: 'Bebas Neue', weight: 400, upper: true, lh: 0.88, track: 0.01 },
    gloock: { family: 'Gloock', weight: 400, upper: false, lh: 1.0, track: -0.01 },
    italiana: { family: 'Italiana', weight: 400, upper: true, lh: 1.0, track: 0.08 },
    bodoni: { family: 'Bodoni Moda', weight: 500, upper: false, lh: 1.0, track: -0.01, italic: false },
    unbounded: { family: 'Unbounded', weight: 300, upper: true, lh: 1.02, track: 0.06 },
    'big-shoulders': { family: 'Big Shoulders Display', weight: 800, upper: true, lh: 0.86, track: 0.01 },
    limelight: { family: 'Limelight', weight: 400, upper: true, lh: 1.0, track: 0.04 },
    'archivo-black': { family: 'Archivo Black', weight: 400, upper: true, lh: 0.92, track: -0.01 },
    poiret: { family: 'Poiret One', weight: 400, upper: true, lh: 1.05, track: 0.1 },
  };
  const BILLING = 'Oswald';

  const mulberry = (a) => () => {
    a |= 0; a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
  const hex = (h) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16));
  const mix = (a, b, k) => { const x = hex(a), y = hex(b); return '#' + x.map((v, i) => Math.round(v + (y[i] - v) * k).toString(16).padStart(2, '0')).join(''); };
  const rgba = (h, a) => { const [r, g, b] = hex(h); return `rgba(${r},${g},${b},${a})`; };
  const lum = (h) => { const [r, g, b] = hex(h).map((v) => v / 255); return 0.2126 * r + 0.7152 * g + 0.0722 * b; };

  let grain;
  function grainTile() {
    if (grain) return grain;
    grain = document.createElement('canvas');
    grain.width = grain.height = 256;
    const g = grain.getContext('2d');
    const img = g.createImageData(256, 256);
    const r = mulberry(7);
    for (let i = 0; i < img.data.length; i += 4) {
      const v = 128 + (r() - 0.5) * 255;
      img.data[i] = img.data[i + 1] = img.data[i + 2] = v;
      img.data[i + 3] = 255;
    }
    g.putImageData(img, 0, 0);
    return grain;
  }

  function vgrad(c, h, stops) {
    const g = c.createLinearGradient(0, 0, 0, h);
    stops.forEach(([o, col]) => g.addColorStop(o, col));
    return g;
  }
  function glow(c, x, y, r, col, a = 0.6) {
    const g = c.createRadialGradient(x, y, 0, x, y, r);
    g.addColorStop(0, rgba(col, a));
    g.addColorStop(0.45, rgba(col, a * 0.35));
    g.addColorStop(1, rgba(col, 0));
    c.save(); c.globalCompositeOperation = 'screen'; c.fillStyle = g; c.fillRect(x - r, y - r, r * 2, r * 2); c.restore();
  }
  const noise1 = (r) => { const ph = [r() * 6.28, r() * 6.28, r() * 6.28, r() * 6.28]; return (x) => Math.sin(x * 1.3 + ph[0]) * 0.5 + Math.sin(x * 2.9 + ph[1]) * 0.25 + Math.sin(x * 6.1 + ph[2]) * 0.14 + Math.sin(x * 13.7 + ph[3]) * 0.07; };

  // Each motif paints the full frame from the palette. Coordinates scale with w and h, so the
  // same motif serves the 2:3 one-sheet and the 16:9 backdrop.
  const MOTIFS = {
    sun(c, w, h, [g0, g1, ac, ink], r) {
      const hz = h * (0.6 + r() * 0.06);
      c.fillStyle = vgrad(c, h, [[0, g0], [hz / h * 0.92, mix(g1, ac, 0.25)], [hz / h, mix(g1, ac, 0.45)], [Math.min(1, hz / h + 0.01), mix(g0, g1, 0.5)], [1, g0]]);
      c.fillRect(0, 0, w, h);
      const sx = w * (0.5 + (r() - 0.5) * 0.18), sr = Math.min(w, h) * 0.2, sy = hz - sr * 0.35;
      glow(c, sx, sy, sr * 4, ac, 0.55);
      c.save(); c.beginPath(); c.rect(0, 0, w, hz); c.clip();
      const sg = c.createLinearGradient(0, sy - sr, 0, sy + sr);
      sg.addColorStop(0, mix(ac, '#ffffff', 0.55)); sg.addColorStop(1, ac);
      c.fillStyle = sg; c.beginPath(); c.arc(sx, sy, sr, 0, Math.PI * 2); c.fill(); c.restore();
      for (let i = 0; i < 26; i++) {
        const y = hz + 4 + i * i * 0.55 * (h / 900) + i * 3;
        if (y > h) break;
        const lw = sr * (1.6 - i * 0.045) * (0.6 + r() * 0.5);
        c.fillStyle = rgba(mix(ac, '#ffffff', 0.3), Math.max(0, 0.55 - i * 0.022));
        c.fillRect(sx - lw / 2 + (r() - 0.5) * sr * 0.3, y, lw, 1.5 + r() * 2);
      }
    },
    orbit(c, w, h, [g0, g1, ac, ink], r) {
      const bg = c.createRadialGradient(w * 0.5, h * 0.55, 0, w * 0.5, h * 0.55, Math.max(w, h) * 0.8);
      bg.addColorStop(0, g1); bg.addColorStop(1, g0);
      c.fillStyle = bg; c.fillRect(0, 0, w, h);
      for (let i = 0; i < 260; i++) { c.fillStyle = rgba(ink, r() * r() * 0.9); const s = r() < 0.04 ? 2.2 : 1.1; c.fillRect(r() * w, r() * h, s, s); }
      const cx = w * 0.5, cy = h * 0.54, rot = -0.22 - r() * 0.1;
      c.save(); c.translate(cx, cy); c.rotate(rot);
      for (let i = 0; i < 7; i++) {
        const rx = Math.min(w, h) * (0.2 + i * 0.11);
        c.strokeStyle = rgba(i === 3 ? ac : ink, i === 3 ? 0.85 : 0.14 + 0.05 * (i % 3));
        c.lineWidth = i === 3 ? 2.2 : 1.1;
        c.beginPath(); c.ellipse(0, 0, rx, rx * 0.3, 0, 0, Math.PI * 2); c.stroke();
      }
      const mr = Math.min(w, h) * (0.2 + 3 * 0.11);
      const ang = 0.7 + r();
      c.fillStyle = mix(ac, '#ffffff', 0.4); c.beginPath(); c.arc(Math.cos(ang) * mr, Math.sin(ang) * mr * 0.3, 4.5, 0, Math.PI * 2); c.fill();
      c.restore();
      glow(c, cx + Math.cos(ang) * mr, cy + Math.sin(ang) * mr * 0.3, 40, ac, 0.8);
      const pr = Math.min(w, h) * 0.12;
      const pg = c.createLinearGradient(cx - pr, cy - pr, cx + pr, cy + pr);
      pg.addColorStop(0, mix(ac, '#ffffff', 0.2)); pg.addColorStop(0.55, mix(g1, ac, 0.3)); pg.addColorStop(1, g0);
      c.fillStyle = pg; c.beginPath(); c.arc(cx, cy, pr, 0, Math.PI * 2); c.fill();
      glow(c, cx - pr * 0.4, cy - pr * 0.4, pr * 2.2, ac, 0.35);
    },
    figure(c, w, h, [g0, g1, ac, ink], r) {
      const hz = h * 0.76;
      c.fillStyle = vgrad(c, h, [[0, g0], [0.55, mix(g0, g1, 0.7)], [hz / h, mix(g1, ac, 0.2)], [hz / h + 0.002, mix(g0, g1, 0.35)], [1, g0]]);
      c.fillRect(0, 0, w, h);
      glow(c, w * 0.5, hz, w * 0.7, ac, 0.22);
      const fx = w * (0.58 + r() * 0.1), fh = h * 0.045, col = mix(g0, '#000000', 0.4);
      c.fillStyle = rgba(g0, 0.55);
      c.beginPath(); c.moveTo(fx - 2, hz); c.lineTo(fx - w * 0.42, hz + h * 0.05); c.lineTo(fx - w * 0.42, hz + h * 0.056); c.lineTo(fx + 2, hz); c.fill();
      c.fillStyle = col;
      c.beginPath(); c.arc(fx, hz - fh * 0.88, fh * 0.12, 0, Math.PI * 2); c.fill();
      c.beginPath(); c.moveTo(fx - fh * 0.14, hz - fh * 0.74); c.lineTo(fx + fh * 0.14, hz - fh * 0.74); c.lineTo(fx + fh * 0.1, hz); c.lineTo(fx - fh * 0.1, hz); c.fill();
    },
    ridges(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = vgrad(c, h, [[0, g0], [0.6, mix(g1, ac, 0.35)], [1, g1]]);
      c.fillRect(0, 0, w, h);
      const mx = w * (0.3 + r() * 0.4), my = h * (h > w ? 0.4 : 0.22 + r() * 0.08), mr = Math.min(w, h) * 0.05;
      glow(c, mx, my, mr * 7, ac, 0.4);
      c.fillStyle = mix(ac, '#ffffff', 0.5); c.beginPath(); c.arc(mx, my, mr, 0, Math.PI * 2); c.fill();
      const layers = 6;
      for (let L = 0; L < layers; L++) {
        const k = L / (layers - 1);
        const n = noise1(r);
        const base = h * (0.46 + k * 0.4), amp = h * (0.07 + k * 0.05);
        c.fillStyle = mix(mix(g1, ac, 0.3 * (1 - k)), g0, 0.25 + k * 0.75);
        c.beginPath(); c.moveTo(0, h);
        for (let x = 0; x <= w; x += 4) c.lineTo(x, base - Math.abs(n(x / w * (2.2 + k))) * amp - n(x / w * 5) * amp * 0.25);
        c.lineTo(w, h); c.fill();
      }
    },
    waves(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = vgrad(c, h, [[0, g0], [0.5, mix(g0, g1, 0.6)], [1, g0]]);
      c.fillRect(0, 0, w, h);
      glow(c, w * 0.5, h * 0.36, w * 0.8, ac, 0.25);
      const rows = 18, hi = 6 + Math.floor(r() * 5);
      for (let i = 0; i < rows; i++) {
        const y0 = h * (0.4 + i * 0.034), amp = h * (0.008 + i * 0.0016), f = 2 + r() * 2, ph = r() * 6;
        c.beginPath();
        for (let x = 0; x <= w; x += 3) c.lineTo(x, y0 + Math.sin(x / w * f * Math.PI * 2 + ph) * amp + Math.sin(x / w * 17 + ph) * amp * 0.2);
        c.lineTo(w, h); c.lineTo(0, h); c.closePath();
        c.fillStyle = rgba(mix(g0, g1, 0.3 + (i % 3) * 0.15), 0.55);
        c.fill();
        c.strokeStyle = rgba(i === hi ? mix(ac, '#ffffff', 0.3) : ink, i === hi ? 0.95 : 0.08 + (rows - i) * 0.006);
        c.lineWidth = i === hi ? 2 : 1;
        c.stroke();
      }
    },
    blinds(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = g0; c.fillRect(0, 0, w, h);
      c.save();
      c.translate(w * 0.5, h * 0.45); c.rotate(-0.42); c.translate(-w, -h);
      const n = 11, sh = h * 0.035;
      for (let i = 0; i < n; i++) {
        const y = h * 0.62 + i * sh * 2.1;
        const gg = c.createLinearGradient(w * 0.4, 0, w * 1.7, 0);
        gg.addColorStop(0, rgba(ac, 0)); gg.addColorStop(0.35, rgba(ac, 0.75 - i * 0.04)); gg.addColorStop(1, rgba(g1, 0));
        c.fillStyle = gg; c.fillRect(w * 0.3, y, w * 1.6, sh);
      }
      c.restore();
      const hx = w * 0.42, hy = h * 0.5, hr = Math.min(w, h) * 0.17;
      c.fillStyle = mix(g0, '#000000', 0.3);
      c.beginPath(); c.ellipse(hx, hy, hr * 0.78, hr, 0, 0, Math.PI * 2); c.fill();
      c.beginPath(); c.moveTo(hx - hr * 0.42, hy + hr * 0.7); c.lineTo(hx + hr * 0.36, hy + hr * 0.7); c.lineTo(hx + hr * 1.3, h); c.lineTo(hx - hr * 1.4, h); c.fill();
      c.strokeStyle = rgba(ac, 0.5); c.lineWidth = 1.5;
      c.beginPath(); c.ellipse(hx, hy, hr * 0.78, hr, 0, -1.25, 0.6); c.stroke();
    },
    road(c, w, h, [g0, g1, ac, ink], r) {
      const hz = h * 0.5;
      c.fillStyle = vgrad(c, h, [[0, g0], [hz / h, mix(g1, ac, 0.5)], [hz / h + 0.001, mix(g0, '#000000', 0.3)], [1, g0]]);
      c.fillRect(0, 0, w, h);
      glow(c, w * 0.5, hz, w * 0.6, ac, 0.5);
      c.fillStyle = mix(g0, '#000000', 0.5);
      c.beginPath(); c.moveTo(w * 0.47, hz); c.lineTo(w * 0.53, hz); c.lineTo(w * 1.1, h); c.lineTo(-w * 0.1, h); c.fill();
      for (let i = 0; i < 14; i++) {
        const t0 = Math.pow(i / 14, 2), t1 = Math.pow((i + 0.45) / 14, 2);
        const y0 = hz + (h - hz) * t0, y1 = hz + (h - hz) * t1;
        const w0 = 1 + t0 * w * 0.012, w1 = 1 + t1 * w * 0.012;
        c.fillStyle = rgba(mix(ac, '#ffffff', 0.4), 0.85);
        c.beginPath(); c.moveTo(w / 2 - w0, y0); c.lineTo(w / 2 + w0, y0); c.lineTo(w / 2 + w1, y1); c.lineTo(w / 2 - w1, y1); c.fill();
      }
      for (const side of [-1, 1]) {
        const g = c.createLinearGradient(0, h, 0, hz);
        g.addColorStop(0, rgba('#ffffff', 0.32)); g.addColorStop(1, rgba('#ffffff', 0));
        c.fillStyle = g; c.beginPath(); c.moveTo(w / 2 + side * w * 0.18, h); c.lineTo(w / 2 + side * w * 0.03, hz + 8); c.lineTo(w / 2 + side * w * 0.01, hz + 8); c.lineTo(w / 2 + side * w * 0.04, h); c.fill();
      }
    },
    lanterns(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = vgrad(c, h, [[0, g0], [0.7, mix(g0, g1, 0.8)], [1, mix(g1, ac, 0.2)]]);
      c.fillRect(0, 0, w, h);
      for (let i = 0; i < 70; i++) {
        const y = h * (1 - Math.pow(r(), 0.7) * 0.92), x = r() * w;
        const s = Math.min(w, h) * (0.006 + r() * r() * 0.04) * (0.4 + y / h);
        glow(c, x, y, s * 5, ac, 0.25 + r() * 0.3);
        c.fillStyle = rgba(mix(ac, '#ffffff', 0.35), 0.75 + r() * 0.25);
        c.beginPath(); c.ellipse(x, y, s * 0.8, s, 0, 0, Math.PI * 2); c.fill();
      }
    },
    monolith(c, w, h, [g0, g1, ac, ink], r) {
      const hz = h * 0.72;
      c.fillStyle = vgrad(c, h, [[0, g0], [hz / h, mix(g1, ac, 0.3)], [hz / h + 0.001, mix(g0, g1, 0.25)], [1, g0]]);
      c.fillRect(0, 0, w, h);
      glow(c, w * 0.5, hz, w * 0.9, ac, 0.28);
      const mw = Math.min(w, h) * 0.13, mh = h * 0.44, mx = w * 0.5 - mw / 2;
      c.fillStyle = rgba(g0, 0.6);
      c.beginPath(); c.moveTo(mx, hz); c.lineTo(mx + mw, hz); c.lineTo(mx + mw + w * 0.5, h); c.lineTo(mx + w * 0.3, h); c.fill();
      const mg = c.createLinearGradient(mx, 0, mx + mw, 0);
      mg.addColorStop(0, mix(g0, '#000000', 0.5)); mg.addColorStop(0.92, mix(g0, '#000000', 0.2)); mg.addColorStop(1, mix(ac, '#ffffff', 0.2));
      c.fillStyle = mg; c.fillRect(mx, hz - mh, mw, mh);
      c.fillStyle = mix(g0, '#000000', 0.5);
      const fx = mx - mw * 0.9, fh = mh * 0.09;
      c.beginPath(); c.arc(fx, hz - fh * 0.88, fh * 0.12, 0, Math.PI * 2); c.fill();
      c.fillRect(fx - fh * 0.1, hz - fh * 0.74, fh * 0.2, fh * 0.74);
    },
    rings(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = g0; c.fillRect(0, 0, w, h);
      const cx = w * (0.62 + r() * 0.1), cy = h * (0.36 + r() * 0.08), R = Math.max(w, h) * 0.55;
      for (let i = 0; i < 9; i++) {
        c.fillStyle = i % 2 ? g0 : mix(g1, ac, i / 9 * 0.9);
        c.beginPath(); c.arc(cx + i * w * 0.012, cy + i * h * 0.006, R * (1 - i * 0.105), 0, Math.PI * 2); c.fill();
      }
      c.fillStyle = mix(ac, '#ffffff', 0.25); c.beginPath(); c.arc(cx + 9 * w * 0.012, cy + 9 * h * 0.006, R * 0.04, 0, Math.PI * 2); c.fill();
    },
    forest(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = vgrad(c, h, [[0, g0], [0.5, mix(g1, ink, 0.12)], [1, g0]]);
      c.fillRect(0, 0, w, h);
      glow(c, w * (0.4 + r() * 0.2), h * 0.56, Math.min(w, h) * 0.5, ac, 0.5);
      for (let L = 0; L < 4; L++) {
        const k = L / 3, count = 10 + L * 6;
        for (let i = 0; i < count; i++) {
          const x = r() * w, tw = (2 + r() * 10) * (0.5 + k * 1.4) * (w / 600);
          c.fillStyle = rgba(mix(mix(g1, ink, 0.15), g0, 0.3 + k * 0.7), 0.35 + k * 0.65);
          c.fillRect(x, 0, tw, h);
        }
      }
      c.fillStyle = vgrad(c, h, [[0, rgba(g0, 0)], [0.7, rgba(g0, 0)], [1, rgba(g0, 0.9)]]);
      c.fillRect(0, 0, w, h);
    },
    streaks(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = vgrad(c, h, [[0, g0], [0.6, mix(g0, g1, 0.5)], [1, g0]]);
      c.fillRect(0, 0, w, h);
      c.save(); c.globalCompositeOperation = 'screen';
      for (let i = 0; i < 34; i++) {
        const warm = r() < 0.35, y0 = h * (0.5 + (r() - 0.5) * 0.3), y1 = h * (0.62 + (r() - 0.5) * 0.4);
        c.strokeStyle = rgba(warm ? '#ff5a3c' : mix(ac, '#ffffff', 0.3), 0.15 + r() * 0.5);
        c.lineWidth = 0.6 + r() * r() * 3.5;
        c.shadowColor = warm ? '#ff5a3c' : ac; c.shadowBlur = 12;
        c.beginPath(); c.moveTo(-w * 0.1, y0); c.bezierCurveTo(w * 0.35, y0 - h * 0.08, w * 0.6, y1 + h * 0.06, w * 1.1, y1); c.stroke();
      }
      c.restore();
    },
    moon(c, w, h, [g0, g1, ac, ink], r) {
      const sea = h * 0.74;
      c.fillStyle = vgrad(c, h, [[0, g0], [sea / h, mix(g0, g1, 0.75)], [sea / h + 0.001, mix(g0, '#000000', 0.2)], [1, g0]]);
      c.fillRect(0, 0, w, h);
      const R = Math.min(w, h) * 0.3, mx = w * 0.5, my = h * 0.36;
      glow(c, mx, my, R * 3, ac, 0.35);
      const mg = c.createRadialGradient(mx - R * 0.3, my - R * 0.3, R * 0.1, mx, my, R);
      mg.addColorStop(0, mix(ink, ac, 0.15)); mg.addColorStop(1, mix(ac, g1, 0.25));
      c.fillStyle = mg; c.beginPath(); c.arc(mx, my, R, 0, Math.PI * 2); c.fill();
      c.save(); c.beginPath(); c.arc(mx, my, R, 0, Math.PI * 2); c.clip();
      for (let i = 0; i < 16; i++) { c.fillStyle = rgba(g1, 0.05 + r() * 0.1); c.beginPath(); c.arc(mx + (r() - 0.5) * R * 1.6, my + (r() - 0.5) * R * 1.6, R * (0.04 + r() * 0.14), 0, Math.PI * 2); c.fill(); }
      c.restore();
      for (let i = 0; i < 30; i++) {
        const y = sea + 3 + i * (h - sea) / 30, lw = R * (0.9 - i * 0.02) * (0.5 + r() * 0.7);
        c.fillStyle = rgba(mix(ac, ink, 0.5), Math.max(0, 0.5 - i * 0.015));
        c.fillRect(mx - lw / 2 + (r() - 0.5) * 20, y, lw, 1.2 + r() * 1.5);
      }
    },
    split(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = g0; c.fillRect(0, 0, w, h);
      c.fillStyle = g1;
      c.beginPath(); c.moveTo(0, h * 0.28); c.lineTo(w, h * 0.62); c.lineTo(w, h); c.lineTo(0, h); c.fill();
      c.strokeStyle = ac; c.lineWidth = 2;
      c.beginPath(); c.moveTo(0, h * 0.28); c.lineTo(w, h * 0.62); c.stroke();
      c.fillStyle = ac; c.beginPath(); c.arc(w * 0.62, h * (0.28 + 0.34 * 0.62), Math.min(w, h) * 0.07, 0, Math.PI * 2); c.fill();
    },
    smoke(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = vgrad(c, h, [[0, g0], [1, mix(g0, g1, 0.6)]]);
      c.fillRect(0, 0, w, h);
      glow(c, w * 0.5, h * 0.8, w * 0.7, ac, 0.3);
      c.save(); c.globalCompositeOperation = 'screen';
      for (let i = 0; i < 46; i++) {
        const x0 = w * (0.3 + r() * 0.4), ph = r() * 6, amp = w * (0.04 + r() * 0.14), f = 1.5 + r() * 2;
        c.strokeStyle = rgba(i % 7 === 0 ? ac : ink, 0.05 + r() * 0.14);
        c.lineWidth = 0.6 + r() * 1.4;
        c.beginPath();
        for (let y = h; y > h * 0.05; y -= 6) {
          const k = 1 - y / h;
          c.lineTo(x0 + Math.sin(k * f * 6 + ph) * amp * k + Math.sin(k * 23 + ph) * 6 * k, y);
        }
        c.stroke();
      }
      c.restore();
    },
    door(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = mix(g0, '#000000', 0.4); c.fillRect(0, 0, w, h);
      const dw = Math.min(w, h) * 0.2, dh = h * 0.38, dx = w * 0.5 - dw / 2, fl = h * 0.66;
      const sp = c.createLinearGradient(0, fl, 0, h);
      sp.addColorStop(0, rgba(ac, 0.45)); sp.addColorStop(1, rgba(ac, 0));
      c.fillStyle = sp; c.beginPath(); c.moveTo(dx, fl); c.lineTo(dx + dw, fl); c.lineTo(dx + dw + w * 0.45, h); c.lineTo(dx - w * 0.45, h); c.fill();
      const dg = c.createLinearGradient(0, fl - dh, 0, fl);
      dg.addColorStop(0, mix(ac, ink, 0.6)); dg.addColorStop(1, ac);
      c.fillStyle = dg; c.fillRect(dx, fl - dh, dw, dh);
      glow(c, w * 0.5, fl - dh * 0.5, dh * 1.4, ac, 0.3);
      c.fillStyle = mix(g0, '#000000', 0.6);
      const fh = dh * 0.36, fx = w * 0.5;
      c.beginPath(); c.arc(fx, fl - fh * 0.88, fh * 0.12, 0, Math.PI * 2); c.fill();
      c.beginPath(); c.moveTo(fx - fh * 0.15, fl - fh * 0.74); c.lineTo(fx + fh * 0.15, fl - fh * 0.74); c.lineTo(fx + fh * 0.11, fl); c.lineTo(fx - fh * 0.11, fl); c.fill();
    },
    field(c, w, h, [g0, g1, ac, ink], r) {
      const light = lum(g0) > 0.5, hz = h * 0.68;
      c.fillStyle = vgrad(c, h, [[0, g0], [hz / h, mix(g1, ac, light ? 0.1 : 0.35)], [hz / h + 0.001, mix(g1, g0, 0.5)], [1, mix(g0, '#000000', light ? 0.15 : 0.4)]]);
      c.fillRect(0, 0, w, h);
      for (let i = 0; i < 6; i++) { c.fillStyle = rgba(light ? '#ffffff' : ink, 0.05 + r() * 0.06); c.beginPath(); c.ellipse(r() * w, h * (0.1 + r() * 0.4), w * (0.2 + r() * 0.3), h * (0.03 + r() * 0.04), 0, 0, Math.PI * 2); c.fill(); }
      for (let i = 0; i < 900; i++) {
        const y = hz + Math.pow(r(), 1.4) * (h - hz), x = r() * w, len = 3 + (y - hz) / (h - hz) * 22;
        c.strokeStyle = rgba(r() < 0.5 ? mix(g1, ac, 0.4) : mix(g0, g1, 0.5), 0.35 + r() * 0.4);
        c.lineWidth = 0.8; c.beginPath(); c.moveTo(x, y); c.lineTo(x + (r() - 0.5) * 4, y - len); c.stroke();
      }
      const hx = w * (0.62 + r() * 0.15), hs = Math.min(w, h) * 0.05;
      c.fillStyle = light ? mix(ink, g1, 0.2) : mix(g0, '#000000', 0.4);
      c.fillRect(hx, hz - hs * 0.7, hs * 1.3, hs * 0.7);
      c.beginPath(); c.moveTo(hx - hs * 0.1, hz - hs * 0.7); c.lineTo(hx + hs * 0.65, hz - hs * 1.25); c.lineTo(hx + hs * 1.4, hz - hs * 0.7); c.fill();
      c.fillStyle = rgba(ac, 0.9); c.fillRect(hx + hs * 0.25, hz - hs * 0.45, hs * 0.2, hs * 0.2);
    },
    tower(c, w, h, [g0, g1, ac, ink], r) {
      const sea = h * 0.78;
      c.fillStyle = vgrad(c, h, [[0, g0], [sea / h, mix(g0, g1, 0.7)], [sea / h + 0.001, mix(g0, '#000000', 0.3)], [1, g0]]);
      c.fillRect(0, 0, w, h);
      for (let i = 0; i < 140; i++) { c.fillStyle = rgba(ink, r() * r() * 0.8); c.fillRect(r() * w, r() * sea * 0.8, 1.2, 1.2); }
      const tx = w * 0.7, top = h * 0.42, tw = Math.min(w, h) * 0.045;
      c.save(); c.globalCompositeOperation = 'screen';
      for (const [a0, a1] of [[-0.2, 0.06], [Math.PI - 0.08, Math.PI + 0.12]]) {
        const L = w * 1.4;
        const g = c.createRadialGradient(tx, top, 0, tx, top, L);
        g.addColorStop(0, rgba(mix(ac, '#ffffff', 0.5), 0.6)); g.addColorStop(1, rgba(ac, 0));
        c.fillStyle = g; c.beginPath(); c.moveTo(tx, top); c.lineTo(tx + Math.cos(a0) * L, top + Math.sin(a0) * L); c.lineTo(tx + Math.cos(a1) * L, top + Math.sin(a1) * L); c.fill();
      }
      c.restore();
      glow(c, tx, top, 60, ac, 0.9);
      c.fillStyle = mix(g0, '#000000', 0.5);
      c.beginPath(); c.moveTo(tx - tw * 0.6, top + 8); c.lineTo(tx + tw * 0.6, top + 8); c.lineTo(tx + tw, sea); c.lineTo(tx - tw, sea); c.fill();
      c.fillRect(tx - tw * 0.75, top - 6, tw * 1.5, 14);
      c.fillStyle = mix(ac, '#ffffff', 0.6); c.fillRect(tx - tw * 0.4, top - 2, tw * 0.8, 8);
      c.fillStyle = mix(g0, '#000000', 0.5);
      c.beginPath(); c.moveTo(tx - w * 0.18, sea); c.quadraticCurveTo(tx, sea - h * 0.035, tx + w * 0.25, sea); c.fill();
    },
    stars(c, w, h, [g0, g1, ac, ink], r) {
      c.fillStyle = vgrad(c, h, [[0, g0], [0.8, mix(g0, g1, 0.8)], [1, g1]]);
      c.fillRect(0, 0, w, h);
      c.save(); c.translate(w / 2, h / 2); c.rotate(-0.6);
      for (let i = 0; i < 1600; i++) { const x = (r() - 0.5) * w * 2, y = (r() + r() + r() - 1.5) * h * 0.08; c.fillStyle = rgba(i % 9 ? ink : ac, r() * 0.35); c.fillRect(x, y, 1, 1); }
      c.restore();
      for (let i = 0; i < 220; i++) { c.fillStyle = rgba(ink, r() * r()); const s = r() < 0.03 ? 2 : 1; c.fillRect(r() * w, r() * h * 0.85, s, s); }
      const n = noise1(r);
      c.fillStyle = mix(g0, '#000000', 0.5);
      c.beginPath(); c.moveTo(0, h);
      for (let x = 0; x <= w; x += 4) c.lineTo(x, h * 0.86 - n(x / w * 2) * h * 0.04);
      c.lineTo(w, h); c.fill();
      const fx = w * 0.38, fy = h * 0.86 - n(0.38 * 2) * h * 0.04, fh = h * 0.04;
      c.beginPath(); c.arc(fx, fy - fh * 0.88, fh * 0.12, 0, Math.PI * 2); c.fill();
      c.fillRect(fx - fh * 0.1, fy - fh * 0.74, fh * 0.2, fh * 0.74);
    },
    type(c, w, h, [g0, g1, ac, ink], r, work, font) {
      c.fillStyle = g0; c.fillRect(0, 0, w, h);
      const f = FONTS[font] || FONTS['archivo-black'];
      c.save(); c.translate(w * 0.5, h * 0.48); c.rotate(-0.08);
      c.fillStyle = mix(g0, ac, 0.85);
      c.font = `${f.weight} ${h * 0.62}px "${f.family}"`; c.textAlign = 'center'; c.textBaseline = 'middle';
      c.fillText((work?.title || 'S').replace(/^The /, '')[0].toUpperCase(), 0, 0);
      c.restore();
      glow(c, w * 0.5, h * 0.45, w * 0.6, ac, 0.15);
    },
  };

  function trackText(c, text, x, y, track, align = 'center') {
    const chars = [...text];
    const widths = chars.map((ch) => c.measureText(ch).width);
    const total = widths.reduce((s, v) => s + v, 0) + track * (chars.length - 1);
    let cx = align === 'center' ? x - total / 2 : align === 'right' ? x - total : x;
    const prev = c.textAlign; c.textAlign = 'left';
    chars.forEach((ch, i) => { c.fillText(ch, cx, y); cx += widths[i] + track; });
    c.textAlign = prev;
    return total;
  }

  function fitTitle(c, text, f, maxW, maxSize, maxLines) {
    const words = text.split(' ');
    for (let size = maxSize; size > 12; size -= 2) {
      c.font = `${f.italic ? 'italic ' : ''}${f.weight} ${size}px "${f.family}"`;
      const tr = f.track * size;
      const lines = [];
      let line = '';
      for (const wd of words) {
        const test = line ? line + ' ' + wd : wd;
        if (measure(c, test, tr) <= maxW || !line) line = test;
        else { lines.push(line); line = wd; }
      }
      lines.push(line);
      if (lines.length <= maxLines && lines.every((l) => measure(c, l, tr) <= maxW)) return { size, lines, tr };
    }
    return { size: 12, lines: [text], tr: 0 };
  }
  const measure = (c, t, tr) => c.measureText(t).width + tr * (t.length - 1);

  const LAYOUT = { sun: 'bottom', orbit: 'bottom', moon: 'bottom', rings: 'bottom', split: 'top', ridges: 'top', forest: 'top', figure: 'top', field: 'top', stars: 'top', waves: 'top', lanterns: 'top', monolith: 'top', tower: 'top', blinds: 'bottom', road: 'top', smoke: 'top', door: 'top', streaks: 'top', type: 'bottom' };

  function onesheet(work) {
    const cv = document.createElement('canvas');
    cv.width = 600; cv.height = 900;
    const c = cv.getContext('2d');
    const w = cv.width, h = cv.height;
    const a = work.art, [g0, g1, ac, ink] = a.palette;
    const r = mulberry(a.seed);
    const motif = MOTIFS[a.motif] ? a.motif : 'smoke';
    MOTIFS[motif](c, w, h, a.palette, r, work, a.font);
    finish(c, w, h, a.palette);

    const f = FONTS[a.font] || FONTS.gloock;
    const light = lum(g0) > 0.5;
    const title = f.upper ? work.title.toUpperCase() : work.title;
    const pos = LAYOUT[motif] || 'top';
    const margin = w * 0.08;
    const fit = fitTitle(c, title, f, w - margin * 2, f.upper ? w * 0.2 : w * 0.15, 3);
    const lh = fit.size * f.lh;
    const block = lh * fit.lines.length;
    const billingTop = h - h * 0.085;
    const titleTop = pos === 'top' ? h * 0.1 : billingTop - h * 0.06 - block;

    if (pos === 'bottom' || light) {
      const sc = c.createLinearGradient(0, titleTop - h * 0.12, 0, h);
      sc.addColorStop(0, rgba(g0, 0)); sc.addColorStop(0.4, rgba(g0, light ? 0.3 : 0.55)); sc.addColorStop(1, rgba(g0, light ? 0.6 : 0.9));
      if (pos === 'bottom') { c.fillStyle = sc; c.fillRect(0, titleTop - h * 0.12, w, h); }
    } else {
      const sc = c.createLinearGradient(0, 0, 0, titleTop + block + h * 0.12);
      sc.addColorStop(0, rgba(g0, 0.65)); sc.addColorStop(1, rgba(g0, 0));
      c.fillStyle = sc; c.fillRect(0, 0, w, titleTop + block + h * 0.12);
    }

    c.fillStyle = ink; c.textBaseline = 'alphabetic';
    c.font = `${f.italic ? 'italic ' : ''}${f.weight} ${fit.size}px "${f.family}"`;
    fit.lines.forEach((ln, i) => trackText(c, ln, w / 2, titleTop + lh * (i + 0.82), fit.tr));

    c.font = `400 ${w * 0.024}px "${BILLING}"`;
    c.fillStyle = rgba(ink, 0.82);
    const tag = (work.tagline || '').toUpperCase();
    const tagY = pos === 'top' ? titleTop + block + h * 0.035 : h * 0.07;
    trackText(c, tag, w / 2, tagY, w * 0.006);

    if (pos === 'bottom') {
      c.font = `500 ${w * 0.021}px "${BILLING}"`;
      c.fillStyle = rgba(ink, 0.88);
      trackText(c, work.cast.slice(0, 3).join('      ').toUpperCase(), w / 2, h * 0.035 + w * 0.005, w * 0.004);
    }

    const bt = billingTop;
    c.fillStyle = rgba(ink, light ? 0.75 : 0.62);
    c.save(); c.translate(w / 2, 0); c.scale(0.78, 1);
    c.font = `300 ${w * 0.0175}px "${BILLING}"`;
    const people = work.cast.concat([work.director || work.creator]);
    const lines = work.type === 'movie'
      ? [`${work.studio} PRESENTS IN ASSOCIATION WITH ${d.works[(a.seed % 7) + 1].title.split(' ').pop()} PICTURES A ${work.studio} PRODUCTION A FILM BY ${work.director}`,
         `${people.slice(0, 4).join('   ')}   "${work.title}"   MUSIC BY ${people[2]}   EDITED BY ${people[1]}   DIRECTOR OF PHOTOGRAPHY ${people[3]}`,
         `PRODUCED BY ${people[0]} ${people[2]}   WRITTEN AND DIRECTED BY ${work.director}`]
      : [`${work.network} ORIGINAL SERIES   CREATED BY ${work.creator}`, `${people.slice(0, 4).join('   ')}   EXECUTIVE PRODUCERS ${people[1]} ${people[3]}`];
    const maxW = (w * 0.86) / 0.78;
    lines.forEach((ln, i) => {
      const text = ln.toUpperCase(), base = w * 0.0175, tr = w * 0.0015;
      c.font = `300 ${base}px "${BILLING}"`;
      const total = measure(c, text, tr);
      if (total > maxW) c.font = `300 ${base * maxW / total}px "${BILLING}"`;
      trackText(c, text, 0, bt + i * w * 0.021, total > maxW ? tr * maxW / total : tr);
    });
    c.restore();
    c.font = `500 ${w * 0.022}px "${BILLING}"`;
    c.fillStyle = rgba(ink, 0.9);
    const foot = work.type === 'movie'
      ? (work.status === 'unreleased' ? `COMING ${work.release ? work.release.toLocaleString('en', { month: 'long' }).toUpperCase() + ' ' + work.release.getFullYear() : work.year}` : `${work.year}`)
      : `${work.network.toUpperCase()}   ·   SEASON ${work.seasons.length}`;
    trackText(c, foot, w / 2, h - h * 0.022, w * 0.008);
    return cv.toDataURL('image/jpeg', 0.86);
  }

  function finish(c, w, h, [g0], grainAmt = 0.075) {
    const v = c.createRadialGradient(w / 2, h * 0.45, Math.min(w, h) * 0.3, w / 2, h / 2, Math.max(w, h) * 0.75);
    v.addColorStop(0, 'rgba(0,0,0,0)'); v.addColorStop(1, 'rgba(0,0,0,0.5)');
    c.fillStyle = v; c.fillRect(0, 0, w, h);
    c.save(); c.globalCompositeOperation = 'overlay'; c.globalAlpha = grainAmt;
    c.fillStyle = c.createPattern(grainTile(), 'repeat'); c.fillRect(0, 0, w, h); c.restore();
  }

  function scene(work, W, H, variant) {
    const cv = document.createElement('canvas');
    cv.width = W; cv.height = H;
    const c = cv.getContext('2d');
    const a = work.art;
    const r = mulberry(a.seed + variant * 7919);
    const motif = MOTIFS[a.motif] ? a.motif : 'smoke';
    if (variant > 0) {
      // Stills reframe the backdrop: a tighter crop at a different point in the scene.
      const big = document.createElement('canvas');
      big.width = W * 1.8; big.height = H * 1.8;
      const bc = big.getContext('2d');
      MOTIFS[motif](bc, big.width, big.height, a.palette, r, work, a.font);
      const ox = r() * (big.width - W), oy = r() * (big.height - H) * 0.8;
      c.drawImage(big, ox, oy, W, H, 0, 0, W, H);
    } else MOTIFS[motif](c, W, H, a.palette, r, work, a.font);
    finish(c, W, H, a.palette, 0.06);
    return cv.toDataURL('image/jpeg', 0.84);
  }

  const cache = new Map();
  const get = (w) => (typeof w === 'string' ? d.W[w] : w);
  const memo = (key, fn) => { if (!cache.has(key)) cache.set(key, fn()); return cache.get(key); };

  const fontsNeeded = [...new Set(Object.values(FONTS).map((f) => `${f.weight} 40px "${f.family}"`))].concat([`300 20px "${BILLING}"`, `400 20px "${BILLING}"`, `500 20px "${BILLING}"`]);

  SS.art = {
    poster: (w) => { w = get(w); return memo('p:' + w.id, () => onesheet(w)); },
    backdrop: (w, opt = {}) => { w = get(w); const W = opt.w || 1600, H = opt.h || 900; return memo(`b:${w.id}:${W}x${H}`, () => scene(w, W, H, 0)); },
    still: (w, n = 1) => { w = get(w); return memo(`s:${w.id}:${n}`, () => scene(w, 480, 270, n)); },
    colors: (w) => { w = get(w); const [g0, g1, ac, ink] = w.art.palette; return { ground: g0, mid: g1, accent: ac, ink, light: lum(g0) > 0.5, glow: mix(g1, ac, 0.6) }; },
    font: (w) => { w = get(w); const f = FONTS[w.art.font] || FONTS.gloock; return { family: `"${f.family}"`, weight: f.weight, upper: f.upper, tracking: f.track + 'em' }; },
    mix, rgba,
    async prewarm(onProgress) {
      await Promise.all(fontsNeeded.map((f) => document.fonts.load(f).catch(() => null)));
      const list = d.works;
      for (let i = 0; i < list.length; i++) {
        SS.art.poster(list[i]);
        if (i % 4 === 3) { onProgress?.((i + 1) / list.length); await new Promise((r) => setTimeout(r, 0)); }
      }
      onProgress?.(1);
    },
  };
})();
