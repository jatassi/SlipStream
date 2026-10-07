/* PROTOTYPE, throwaway page shell over Engine. Settings live in memory only. */
(() => {
  const E = Engine, D = window.DATA;
  const Q = E.QUALITY;
  const $ = s => document.querySelector(s);
  const esc = s => String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
  const gb = b => (b / 1e9).toFixed(b >= 1e11 ? 0 : 1) + ' GB';
  const tb = b => b >= 1e12 ? (b / 1e12).toFixed(2) + ' TB' : gb(b);
  const clone = o => JSON.parse(JSON.stringify(o));

  /* ---------------- default settings (the thing being prototyped) ---------------- */
  const LADDERS = {
    picture: [['dv+hdr10p'], ['dv+hdr10'], ['hdr10p'], ['hdr10'], ['hlg'], ['dv_no_fallback'], ['sdr']],
    audio: [['truehd_atmos', 'dtsx'], ['dtshd_ma', 'truehd', 'pcm', 'flac'], ['ddp_atmos'], ['atmos_undefined', 'dtshd_hra', 'dts', 'ddp'], ['dd'], ['aac', 'opus', 'mp3'], ['unknown']],
    source: [['remux'], ['bluray'], ['webdl'], ['webrip'], ['hdtv'], ['dvd'], ['screener', 'telesync', 'cam']],
    service: [['MA', 'iT'], ['AMZN', 'ATVP'], ['NF', 'HMAX', 'DSNP', 'PMTP', 'PCOK', 'HULU']],
    codec: [['h265'], ['h264'], ['av1'], ['vc1', 'mpeg2']],
  };
  const LADDER_LABEL = k => E.PICTURE_LABEL[k] || E.AUDIO_LABEL[k] || E.CODEC_LABEL[k] || k;
  const sizesFromTrash = () => {
    const out = {};
    const src = (D.trash && D.trash.sizes) || {};
    for (const q of E.QUALITIES) out[q.id] = src[q.id] ? { ...src[q.id] } : { min: 0, preferred: null, max: null };
    return out;
  };
  const DEFAULTS = {
    sizes: sizesFromTrash(),
    renditions: [
      {
        id: 'r2160', name: '2160p HDR', qualities: [['webrip-2160'], ['webdl-2160'], ['bluray-2160'], ['remux-2160']], cutoff: 'remux-2160',
        upgrades: true, pastCutoff: false, minGainTopN: 2,
        filters: { picture: ['dv+hdr10p', 'dv+hdr10', 'hdr10p', 'hdr10', 'hlg'], codecs: null, minChannels: 0, languages: [], unnamedLanguageCounts: true },
        sizeOverride: {},
        priorities: [
          { measure: 'hazards' }, { measure: 'picture', ladder: clone(LADDERS.picture) }, { measure: 'trust', band: 0.1 },
          { measure: 'audio', ladder: clone(LADDERS.audio) }, { measure: 'source', ladder: clone(LADDERS.source) }, { measure: 'size', band: 0.15 },
        ],
        timing: { patience: 'balanced', stepWindowHours: 72, threshold: 0.7, minSamples: 8, horizonDays: 180 },
      },
      {
        id: 'r1080', name: '1080p for every screen', qualities: [['webrip-1080'], ['webdl-1080'], ['bluray-1080'], ['remux-1080']], cutoff: 'webdl-1080',
        upgrades: true, pastCutoff: false, minGainTopN: 2,
        filters: { picture: ['sdr'], codecs: ['h264', 'h265'], minChannels: 0, languages: [], unnamedLanguageCounts: true },
        sizeOverride: {},
        priorities: [
          { measure: 'hazards' }, { measure: 'trust', band: 0.1 }, { measure: 'audio', ladder: clone(LADDERS.audio) },
          { measure: 'codec', ladder: [['h264'], ['h265'], ['av1'], ['vc1', 'mpeg2']] }, { measure: 'size', band: 0.15 },
        ],
        timing: { patience: 'eager', stepWindowHours: 72, threshold: 0.7, minSamples: 8, horizonDays: 180 },
      },
    ],
    profiles: [
      { id: 'p-movies', name: 'Movies', contentType: 'movie', editions: 'any',
        entries: [{ rendition: 'r2160', sourcing: 'download' }, { rendition: 'r1080', sourcing: 'download_then_derive', from: 'r2160', graceDays: 14 }] },
      { id: 'p-series', name: 'Series', contentType: 'series', editions: 'any',
        entries: [{ rendition: 'r2160', sourcing: 'download' }, { rendition: 'r1080', sourcing: 'derive', from: 'r2160' }] },
      { id: 'p-series-1080', name: 'Series, 1080p only', contentType: 'series', editions: 'any',
        entries: [{ rendition: 'r1080', sourcing: 'download' }] },
    ],
    defaults: { movie: 'p-movies', series: 'p-series' },
    workOverrides: {},
    knowledge: { base: true, trash: false, trashKinds: { term: true, hazard: true, reputation: true }, disabled: [], severity: {}, confidence: {} },
    trust: { weakBelow: 0.6, priorStrength: 10, calibrateTerms: true, priors: { tier1: 0.92, tier2: 0.88, tier3: 0.84, tier4: 0.8, tier5: 0.8, tier6: 0.8, tier7: 0.8, tier8: 0.8, preferred: 0.9, scene: 0.8, fansub: 0.75, unknown: 0.75 } },
    verification: { probeBeforeImport: true, onMismatch: 'reject_import', blocklist: true, seedPolicy: 'seed_to_goal', probationUpgrades: true, heuristicsAdvisory: true, unknownSource: 'unknown' },
  };

  let S = clone(DEFAULTS);
  let BASE = clone(DEFAULTS);
  const view = { section: 'impact', rendition: 'r2160', work: null, item: null, prioVariant: 'list', walk: 0, walkStep: 0, claimFilter: '', claimKind: 'all', teach: [], arrProfile: null };

  /* ---------------- derived data ---------------- */
  const appOf = type => (type === 'movie' ? 'radarr' : 'sonarr');
  function packsFor(type, s) {
    const out = [];
    const off = new Set(s.knowledge.disabled);
    const tweak = c => ({ ...c, severity: s.knowledge.severity[c.id] || c.severity, confidence: s.knowledge.confidence[c.id] || c.confidence });
    if (s.knowledge.base) out.push({ ...E.BASE_PACK, claims: E.BASE_PACK.claims.filter(c => !off.has(c.id)).map(tweak) });
    if (s.knowledge.trash && D.trash && D.trash.pack) {
      const app = appOf(type);
      out.push({ ...D.trash.pack, claims: D.trash.pack.claims.filter(c => c.app === app && s.knowledge.trashKinds[c.kind] && !off.has(c.id)).map(tweak) });
    }
    return out;
  }
  const memo = new Map();
  function derive(s) {
    const key = JSON.stringify([s.knowledge, s.trust, s.verification.unknownSource]);
    if (memo.has(key)) return memo.get(key);
    const packs = { movie: packsFor('movie', s), series: packsFor('series', s) };
    const icache = new Map();
    const interp = (name, type, sizeBytes) => {
      const k = type + '|' + name;
      if (!icache.has(k)) icache.set(k, E.interpret(name, packs[type], { sizeBytes }));
      return icache.get(k);
    };
    const settingsLike = { ...s, _reputationByGroup: {} };
    for (const r of D.releases) {
      const it = interp(r.n, r.type);
      if (it.core.group && it.reputations.length) {
        const order = ['tier1', 'preferred', 'tier2', 'tier3', 'tier4', 'tier5', 'tier6', 'tier7', 'tier8', 'scene', 'fansub'];
        const best = it.reputations.slice().sort((a, b) => order.indexOf(a.prior) - order.indexOf(b.prior))[0];
        const cur = settingsLike._reputationByGroup[it.core.group];
        if (!cur || order.indexOf(best.prior) < order.indexOf(cur.prior)) settingsLike._reputationByGroup[it.core.group] = best;
      }
    }
    const observations = [];
    for (const r of D.releases) {
      const probed = r.files.filter(f => f.f && f.f.video);
      if (!probed.length) continue;
      const it = interp(r.n, r.type);
      const per = new Map();
      for (const f of probed) {
        const facts = E.factsFromProbe(f.f);
        const v = E.verify(it, facts);
        for (const c of v.checks) {
          const k = c.key + '|' + c.claimed;
          if (!per.has(k) || c.verdict === 'refuted') per.set(k, c);
        }
      }
      observations.push({ name: r.n, group: it.core.group, type: r.type, checks: [...per.values()] });
    }
    const trust = E.trustTable(observations);
    const claimStats = {};
    for (const o of observations) for (const c of o.checks) {
      if (c.verdict === 'unobservable') continue;
      for (const id of c.sources || []) { const st = claimStats[id] = claimStats[id] || { ok: 0, n: 0 }; st.n++; if (c.verdict === 'confirmed') st.ok++; }
    }
    if (s.trust.calibrateTerms) settingsLike._claimStats = claimStats;
    const out = { packs, interp, settingsLike, observations, trust, claimStats };
    if (memo.size > 4) memo.clear(); memo.set(key, out);
    return out;
  }

  function profileFor(work, s) { return s.profiles.find(p => p.id === (s.workOverrides[work.key] || s.defaults[work.type])) || s.profiles[0]; }
  const rendById = (id, s = S) => s.renditions.find(r => r.id === id);

  function fileView(file, type, s, d) {
    let name = file.n;
    const facts = E.factsFromProbe(file.f);
    if (!facts) return null;
    const v = E.viewFromFile(name, facts, d.trust, d.settingsLike, d.packs[type]);
    if (!v.quality && s.verification.unknownSource !== 'unknown') {
      v.source = s.verification.unknownSource; v.quality = E.qualityOf(v.source, facts.res); v.sourceInferred = true;
    }
    v.file = file;
    return v;
  }
  function candView(c, type, s, d, runtimeMin) {
    const it = d.interp(c.n, type, c.s);
    const v = E.viewFromClaims(it, d.trust, d.settingsLike, { runtimeMin, sizeBytes: c.s });
    v.cand = c;
    return v;
  }

  /* library impact: for every Item, which wanted Renditions are filled by what is held */
  const cacheBy = (fn) => { const m = new Map(); return s => { const k = JSON.stringify(s); if (!m.has(k)) { if (m.size > 6) m.clear(); m.set(k, fn(s)); } return m.get(k); }; };
  const impact = cacheBy(impactRaw);
  function impactRaw(s) {
    const d = derive(s);
    const per = {};
    for (const r of s.renditions) per[r.id] = { wanted: 0, cutoff: 0, below: 0, derivable: 0, missing: 0, bytes: 0, examplesMissing: [] };
    const orphan = []; let orphanBytes = 0, unknownSource = 0;
    const fileFills = new Map();
    for (const w of D.works) {
      const prof = profileFor(w, s);
      const rends = prof.entries.map(e => ({ e, r: rendById(e.rendition, s) })).filter(x => x.r);
      for (const item of w.items) {
        const held = item.held.map(i => fileView(D.files[i], w.type, s, d)).filter(Boolean);
        for (const v of held) if (!v.quality) unknownSource++;
        const filledBy = {};
        for (const { r } of rends) {
          const fillers = held.filter(v => E.gate(v, r, d.settingsLike).length === 0);
          filledBy[r.id] = fillers;
          for (const v of fillers) { const k = v.file.i; if (!fileFills.has(k)) fileFills.set(k, new Set()); fileFills.get(k).add(r.id); }
        }
        for (const { e, r } of rends) {
          const P = per[r.id]; P.wanted++;
          const fl = filledBy[r.id];
          if (fl.length) {
            const best = Math.max(...fl.map(v => E.qualityRank(r, v.quality)));
            if (best >= E.qualityRank(r, r.cutoff)) P.cutoff++; else P.below++;
            P.bytes += Math.max(...fl.map(v => v.sizeBytes || 0));
          } else if ((e.sourcing === 'derive' || e.sourcing === 'download_then_derive') && e.from && filledBy[e.from] && filledBy[e.from].length) P.derivable++;
          else { P.missing++; if (P.examplesMissing.length < 400) P.examplesMissing.push({ w, item, held }); }
        }
        for (const v of held) if (!rends.some(({ r }) => filledBy[r.id].includes(v))) { orphan.push({ w, item, v }); orphanBytes += v.sizeBytes || 0; }
      }
    }
    return { per, orphan, orphanBytes, unknownSource, fileFills };
  }

  /* decisions per Item and Rendition, for replay against a baseline */
  const decisions = cacheBy(decisionsRaw);
  function decisionsRaw(s) {
    const d = derive(s);
    const out = new Map();
    for (const w of D.works) {
      const prof = profileFor(w, s);
      for (const item of w.items) {
        if (item.cands.length < 2) continue;
        const rt = runtimeOf(item);
        const views = item.cands.map(ci => candView(D.cands[ci], w.type, s, d, rt));
        for (const e of prof.entries) {
          const r = rendById(e.rendition, s); if (!r || e.sourcing === 'derive') continue;
          const j = E.judge(views, r, d.settingsLike);
          out.set(w.key + '|' + item.key + '|' + r.id, { w, item, r, winner: j.winner ? j.winner.v.name : null });
        }
      }
    }
    return out;
  }
  const runtimeOf = item => { for (const i of item.held) { const f = D.files[i]; if (f.f && f.f.duration_s) return f.f.duration_s / 60; } for (const ci of item.cands) { const c = D.cands[ci]; if (c.f && c.f.duration_s) return c.f.duration_s / 60; } return null; };

  /* ---------------- settings diff (surface the state) ---------------- */
  function diffPaths(a, b, path = '', out = []) {
    if (JSON.stringify(a) === JSON.stringify(b)) return out;
    if (typeof a !== 'object' || typeof b !== 'object' || a === null || b === null || Array.isArray(a) !== Array.isArray(b)) { out.push({ path, from: a, to: b }); return out; }
    const keys = new Set([...Object.keys(a), ...Object.keys(b)]);
    for (const k of keys) diffPaths(a[k], b[k], path ? path + '.' + k : k, out);
    return out;
  }

  /* ---------------- rendering ---------------- */
  const SECTIONS = [
    ['Overview', [['impact', 'Library impact'], ['try', 'Try it on a title']]],
    ['What you want', [['renditions', 'Renditions'], ['priorities', 'Priorities'], ['profiles', 'Rendition profiles'], ['qualities', 'Qualities and sizes']]],
    ['What is true', [['knowledge', 'Knowledge packs'], ['trust', 'Trust and verification']]],
    ['When', [['timing', 'Timing (Horizon)']]],
    ['Prototype', [['walk', 'Walkthroughs'], ['state', 'Settings state']]],
  ];
  function renderNav() {
    $('#nav').innerHTML = SECTIONS.map(([g, xs]) => `<div class="grp">${g}</div>` + xs.map(([id, l]) => `<button data-act="nav" data-id="${id}" class="${view.section === id ? 'on' : ''}">${l}</button>`).join('')).join('');
  }
  function renderTop() {
    const I = impact(S), I0 = impact(BASE);
    const ch = diffPaths(BASE, S).length;
    const chips = S.renditions.map(r => {
      const p = I.per[r.id], p0 = I0.per[r.id] || {};
      const delta = (k) => p0[k] !== undefined && p0[k] !== p[k] ? ` <span class="muted">(${p[k] - p0[k] > 0 ? '+' : ''}${p[k] - p0[k]})</span>` : '';
      return `<span class="${JSON.stringify(p) !== JSON.stringify(p0) ? 'chg' : ''}">${esc(r.name)}: <b>${p.cutoff}</b> at cutoff${delta('cutoff')}, <b>${p.below}</b> upgradable${delta('below')}, <b>${p.derivable}</b> derivable, <b>${p.missing}</b> missing${delta('missing')}</span>`;
    }).join('');
    $('#impactStrip').innerHTML = chips + `<span>${I.orphan.length} files fill nothing (${tb(I.orphanBytes)})</span>` + (ch ? `<span class="chg">${ch} setting${ch > 1 ? 's' : ''} changed · <button class="b" data-act="nav" data-id="state">view</button> <button class="b" data-act="resetAll">reset</button> <button class="b" data-act="markBase">mark as baseline</button></span>` : '');
  }

  /* ---- impact ---- */
  function renderImpact() {
    const I = impact(S);
    const d = derive(S);
    const dupGroups = new Map();
    for (const f of D.files) if (f.dup != null) { if (!dupGroups.has(f.dup)) dupGroups.set(f.dup, []); dupGroups.get(f.dup).push(f); }
    let dupBytes = 0, dupCount = 0;
    for (const g of dupGroups.values()) if (g.length > 1) { dupCount += g.length - 1; dupBytes += g[0].s * (g.length - 1); }
    const lies = d.observations.filter(o => o.checks.some(c => c.verdict === 'refuted'));
    const rows = S.renditions.map(r => {
      const p = I.per[r.id]; const tot = p.wanted || 1;
      const seg = (n, c) => `<i style="width:${(100 * n / tot).toFixed(1)}%;background:var(${c})"></i>`;
      return `<tr><td><b>${esc(r.name)}</b><div class="small muted">cutoff ${Q[r.cutoff].name}</div></td><td class="n">${p.wanted}</td><td class="n">${p.cutoff}</td><td class="n">${p.below}</td><td class="n">${p.derivable}</td><td class="n">${p.missing}</td><td class="n">${tb(p.bytes)}</td>
      <td><div class="bar">${seg(p.cutoff, '--good')}${seg(p.below, '--warn')}${seg(p.derivable, '--acc')}${seg(p.missing, '--bad')}</div></td></tr>`;
    }).join('');
    return `<h2>Library impact</h2>
      <p class="lead">Every number below is computed from your real library: ${D.meta.libraryFiles} library files and ${D.meta.releases} releases in Downloads, each probed with MediaInfo (${esc(D.meta.probe)}). Change any setting and these recompute. The strip at the top shows the change against the baseline.</p>
      <div class="card"><h3>Renditions across your Items</h3>
      <div class="scroll"><table><thead><tr><th>Rendition</th><th class="n">Items wanting it</th><th class="n">Filled at cutoff</th><th class="n">Filled, upgradable</th><th class="n">Derivable</th><th class="n">Missing</th><th class="n">Bytes held</th><th></th></tr></thead><tbody>${rows}</tbody></table></div>
      <div class="legend"><span><i style="background:var(--good)"></i>at cutoff</span><span><i style="background:var(--warn)"></i>below cutoff</span><span><i style="background:var(--acc)"></i>derivable from another Rendition</span><span><i style="background:var(--bad)"></i>missing</span></div>
      <p class="small muted">An Item is a movie or an episode. A file fills every Rendition whose rules it meets (ADR 0002), judged on probed facts. Source (WEB-DL or Blu-ray) can't be probed, so it comes from the matched release name or the Plex filename. ${I.unknownSource} held files have no source in either, so they fill nothing until you choose how to treat them (Trust and verification).</p></div>
      <div class="grid3">
        <div class="card"><h3>${dupCount} byte-identical copies</h3><p class="small">${tb(dupBytes)} of your library is a second copy of a file already present for the same Item. One file fills every Rendition it meets, so v2 keeps one.</p></div>
        <div class="card"><h3>${I.orphan.length} files fill nothing</h3><p class="small">${tb(I.orphanBytes)}. These held files meet none of the Renditions their Work wants under these settings. Nothing is deleted. Upgrades only delete a file that fills nothing (ADR 0002).</p><details><summary>List</summary>${listOrphans(I.orphan)}</details></div>
        <div class="card"><h3>${lies.length} of ${d.observations.length} releases made a claim the file didn't back up</h3><p class="small">Their names claim something the probe refutes. See Trust and verification.</p></div>
      </div>
      <div class="card"><h3>Missing, by Work</h3>${listMissing(I)}</div>`;
  }
  function listOrphans(orphan) {
    return `<div class="scroll"><table><thead><tr><th>Work</th><th>Item</th><th>File</th><th>Why it fills nothing</th></tr></thead><tbody>${orphan.slice(0, 300).map(o => {
      const prof = profileFor(o.w, S);
      const reasons = prof.entries.map(e => { const r = rendById(e.rendition); return r ? `${r.name}: ${E.gate(o.v, r, derive(S).settingsLike).map(f => f.text).join('; ')}` : ''; }).join('<br>');
      return `<tr><td>${esc(o.w.title)}</td><td>${esc(o.item.key)}</td><td class="rel">${esc(o.v.file.p)}</td><td class="small">${reasons}</td></tr>`;
    }).join('')}</tbody></table></div>${orphan.length > 300 ? `<p class="small muted">First 300 of ${orphan.length}.</p>` : ''}`;
  }
  function listMissing(I) {
    const byWork = new Map();
    for (const r of S.renditions) for (const m of I.per[r.id].examplesMissing) {
      const k = m.w.key; if (!byWork.has(k)) byWork.set(k, { w: m.w, n: {} });
      byWork.get(k).n[r.name] = (byWork.get(k).n[r.name] || 0) + 1;
    }
    const xs = [...byWork.values()].sort((a, b) => a.w.title.localeCompare(b.w.title));
    return `<div class="scroll"><table><thead><tr><th>Work</th>${S.renditions.map(r => `<th class="n">${esc(r.name)}</th>`).join('')}<th></th></tr></thead><tbody>${xs.map(x => `<tr><td>${esc(x.w.title)} <span class="muted small">${x.w.year || ''}</span></td>${S.renditions.map(r => `<td class="n">${x.n[r.name] || ''}</td>`).join('')}<td><button class="b" data-act="tryWork" data-w="${esc(x.w.key)}">Open</button></td></tr>`).join('')}</tbody></table></div>`;
  }

  /* ---- renditions ---- */
  function renderRenditions() {
    const r = rendById(view.rendition) || S.renditions[0];
    const I = impact(S);
    const tabs = S.renditions.map(x => `<button class="${x.id === r.id ? 'on' : ''}" data-act="pickRend" data-id="${x.id}">${esc(x.name)}</button>`).join('') + `<button data-act="addRend">+ New Rendition</button>`;
    const allowed = new Set(r.qualities.flat());
    const qrows = r.qualities.map((tier, i) => `<tr><td class="n">${i + 1}</td><td>${tier.map(id => `<span class="pill acc">${Q[id].name}</span>`).join(' = ')}</td>
      <td class="row"><button class="b" data-act="qMove" data-i="${i}" data-d="1" ${i === r.qualities.length - 1 ? 'disabled' : ''}>higher</button><button class="b" data-act="qMove" data-i="${i}" data-d="-1" ${i === 0 ? 'disabled' : ''}>lower</button>
      ${i > 0 ? `<button class="b" data-act="qTie" data-i="${i}">tie with ${Q[r.qualities[i - 1][0]].name}</button>` : ''}${tier.length > 1 ? `<button class="b" data-act="qSplit" data-i="${i}">split</button>` : ''}
      <button class="b" data-act="qRemove" data-i="${i}">remove</button></td><td>${tier.includes(r.cutoff) ? '<span class="pill good">cutoff</span>' : `<button class="b" data-act="qCutoff" data-id="${tier[0]}">make cutoff</button>`}</td></tr>`).reverse().join('');
    const addable = E.QUALITIES.filter(q => !allowed.has(q.id));
    const contains = S.renditions.filter(o => o.id !== r.id).map(o => {
      let aIn = 0, aAll = 0;
      for (const [, set] of I.fileFills) { if (set.has(r.id)) { aAll++; if (set.has(o.id)) aIn++; } }
      return aAll && aIn === aAll ? `<div class="why warn small">Every file that fills ${esc(r.name)} also fills ${esc(o.name)} in your library. Overlap is allowed (ADR 0002); this is a hint, not an error.</div>` : '';
    }).join('');
    const f = r.filters;
    const pk = E.PICTURE_KEYS.map(k => `<label class="chk"><input type="checkbox" data-act="pic" data-k="${k}" ${f.picture.includes(k) ? 'checked' : ''}>${E.PICTURE_LABEL[k]}</label>`).join('');
    const codecs = ['h265', 'h264', 'av1', 'vc1', 'mpeg2'].map(k => `<label class="chk"><input type="checkbox" data-act="codec" data-k="${k}" ${!f.codecs || f.codecs.includes(k) ? 'checked' : ''}>${E.CODEC_LABEL[k]}</label>`).join('');
    const p = I.per[r.id];
    const used = S.profiles.filter(pr => pr.entries.some(e => e.rendition === r.id)).map(pr => pr.name);
    return `<h2>Renditions</h2><p class="lead">A Rendition is a library-wide encoding target. Its quality order decides first. Its filters are hard rules. Its priorities rank what passes. A Rendition profile picks which Renditions a Work wants.</p>
      <div class="tabs">${tabs}</div>
      <div class="card"><div class="row"><label>Name <input type="text" data-act="rName" value="${esc(r.name)}"></label>
        <label class="chk"><input type="checkbox" data-act="rUpg" ${r.upgrades ? 'checked' : ''}>Upgrades</label>
        <label class="chk"><input type="checkbox" data-act="rPast" ${r.pastCutoff ? 'checked' : ''}>Keep upgrading past cutoff when a top priority improves</label>
        <label>Minimum gain: a top <input type="number" min="1" max="9" data-act="rTopN" value="${r.minGainTopN}"> priority must improve</label>
        ${S.renditions.length > 1 ? `<button class="b" data-act="delRend" ${used.length ? 'disabled title="Used by a profile"' : ''}>Delete</button>` : ''}</div>
        <p class="small muted">Used by ${used.length ? used.map(esc).join(', ') : 'no profile'}. In your library: ${p.cutoff} Items at cutoff, ${p.below} upgradable, ${p.derivable} derivable, ${p.missing} missing.</p>${contains}</div>
      <div class="grid2">
        <div class="card"><h3>Allowed qualities, best first</h3><p class="small muted">Higher in this list is an upgrade. A tie means neither is an upgrade over the other, and priorities decide. A proper or repack of the held quality always replaces it.</p>
          <table><thead><tr><th class="n">#</th><th>Quality</th><th></th><th></th></tr></thead><tbody>${qrows}</tbody></table>
          <div class="row"><select data-act="qAddSel"><option value="">Add a quality…</option>${addable.map(q => `<option value="${q.id}">${q.name}${q.cinema ? ' (cinema source)' : ''}</option>`).join('')}</select>
          <span class="small muted">Presets fill the order:</span><button class="b" data-act="qPreset" data-p="aggressive">Any upgrade</button><button class="b" data-act="qPreset" data-p="resolution">Resolution only</button><button class="b" data-act="qPreset" data-p="balanced">Balanced</button></div></div>
        <div class="card"><h3>Hard filters</h3>
          <div><b class="small">Picture allowed</b><div class="row">${pk}</div><p class="small muted">A claim too weak to trust is judged at its worst plausible value. A "DV" claim from a group whose DV tags often fail counts as whatever is left.</p></div>
          <div><b class="small">Video codec allowed</b><div class="row">${codecs}</div></div>
          <div class="row"><label>Minimum audio channels <input type="number" min="0" max="8" data-act="rCh" value="${f.minChannels || 0}"></label>
          <label>Required audio language (any of) <input type="text" data-act="rLang" placeholder="e.g. en, de" value="${esc((f.languages || []).join(', '))}"></label>
          <label class="chk"><input type="checkbox" data-act="rLangUnnamed" ${f.unnamedLanguageCounts ? 'checked' : ''}>A name with no language counts as meeting it</label></div>
          <div><b class="small">Size per minute</b><p class="small muted">Uses Qualities and sizes. Override for this Rendition:</p>
          <table><thead><tr><th>Quality</th><th class="n">Min MB/min</th><th class="n">Max MB/min</th></tr></thead><tbody>${r.qualities.flat().map(id => { const o = r.sizeOverride[id] || {}; const g = S.sizes[id] || {}; return `<tr><td>${Q[id].name}</td><td class="n"><input type="number" data-act="rSz" data-q="${id}" data-k="min" placeholder="${g.min ?? ''}" value="${o.min ?? ''}"></td><td class="n"><input type="number" data-act="rSz" data-q="${id}" data-k="max" placeholder="${g.max ?? ''}" value="${o.max ?? ''}"></td></tr>`; }).join('')}</tbody></table></div>
        </div>
      </div>
      <div class="card"><h3>What fills ${esc(r.name)} right now</h3>${fillSample(r, I)}</div>`;
  }
  function fillSample(r, I) {
    const d = derive(S);
    const byReason = new Map();
    let n = 0;
    for (const w of D.works) {
      const prof = profileFor(w, S); if (!prof.entries.some(e => e.rendition === r.id)) continue;
      for (const item of w.items) for (const fi of item.held) {
        const v = fileView(D.files[fi], w.type, S, d); if (!v) continue;
        const fails = E.gate(v, r, d.settingsLike);
        const k = fails.length ? fails.map(x => x.rule).join(' + ') : 'fills';
        if (!byReason.has(k)) byReason.set(k, { n: 0, bytes: 0, ex: [] });
        const b = byReason.get(k); b.n++; b.bytes += v.sizeBytes || 0; if (b.ex.length < 4) b.ex.push({ v, fails, w, item }); n++;
      }
    }
    return `<p class="small muted">${n} held files in Works whose profile wants this Rendition, grouped by outcome.</p><div class="scroll"><table><thead><tr><th>Outcome</th><th class="n">Files</th><th class="n">Bytes</th><th>Examples</th></tr></thead><tbody>${[...byReason.entries()].sort((a, b) => b[1].n - a[1].n).map(([k, b]) => `<tr><td>${k === 'fills' ? '<span class="pill good">fills</span>' : `<span class="pill bad">fails: ${esc(k)}</span>`}</td><td class="n">${b.n}</td><td class="n">${tb(b.bytes)}</td><td class="small">${b.ex.map(e => `<div><span class="rel">${esc(e.v.file.n)}</span>${e.fails.length ? ` <span class="muted">→ ${esc(e.fails.map(f => f.text).join('; '))}</span>` : ` <span class="muted">→ ${Q[e.v.quality].name}, ${E.PICTURE_LABEL[e.v.picture]}, ${E.AUDIO_LABEL[e.v.audio]}</span>`}</div>`).join('')}</td></tr>`).join('')}</tbody></table></div>`;
  }

  /* ---- priorities ---- */
  function renderPriorities() {
    const r = rendById(view.rendition) || S.renditions[0];
    const tabs = S.renditions.map(x => `<button class="${x.id === r.id ? 'on' : ''}" data-act="pickRend" data-id="${x.id}">${esc(x.name)}</button>`).join('');
    const vt = `<div class="tabs"><button class="${view.prioVariant === 'list' ? 'on' : ''}" data-act="prioVar" data-v="list">Variant A · Order a list</button><button class="${view.prioVariant === 'teach' ? 'on' : ''}" data-act="prioVar" data-v="teach">Variant B · Teach by choosing between your releases</button></div>`;
    const body = view.prioVariant === 'list' ? prioList(r) : prioTeach(r);
    return `<h2>Priorities</h2><p class="lead">After quality order and revision, a Rendition ranks candidates by these priorities in order. The first priority on which two releases really differ (outside its band) decides, and that row is the explanation. Nothing is summed.</p>
      <div class="tabs">${tabs}</div>${vt}${body}<div class="card"><h3>Live preview on your releases</h3>${prioPreview(r)}</div>`;
  }
  function prioList(r) {
    const used = new Set(r.priorities.map(p => p.measure));
    const items = r.priorities.map((p, i) => {
      const m = E.MEASURES[p.measure];
      let extra = '';
      if (m.ladder) extra = ladderEditor(p, i);
      if (p.band !== undefined) extra += `<label class="small">Band <input type="number" step="0.01" min="0" data-act="pBand" data-i="${i}" value="${p.band}"> <span class="muted">${p.measure === 'size' ? 'fraction: 0.15 = within 15% of each other counts as a tie' : 'reliability difference that still counts as a tie'}</span></label>`;
      if (p.measure === 'language') extra += `<label class="small">Wanted <input type="text" data-act="pLang" data-i="${i}" value="${esc((p.wanted || []).join(', '))}"></label>`;
      return `<div class="it"><span class="k">${i + 1}</span><span class="row" style="gap:2px"><button class="b" data-act="pMove" data-i="${i}" data-d="-1" ${i === 0 ? 'disabled' : ''}>▲</button><button class="b" data-act="pMove" data-i="${i}" data-d="1" ${i === r.priorities.length - 1 ? 'disabled' : ''}>▼</button></span>
        <div><b>${m.label}</b> <span class="small muted">${m.unit}</span><div>${extra}</div></div><button class="b" data-act="pDel" data-i="${i}">remove</button></div>`;
    }).join('');
    const add = Object.entries(E.MEASURES).filter(([k]) => !used.has(k)).map(([k, m]) => `<option value="${k}">${m.label}</option>`).join('');
    return `<div class="card"><div class="prio">${items}</div><div class="row"><select data-act="pAdd"><option value="">Add a priority…</option>${add}</select>
      <span class="small muted">Hidden rule after the list: a file that fills more wanted Renditions wins a full tie, then name order.</span></div></div>`;
  }
  function ladderEditor(p, i) {
    return `<div class="ladder">${p.ladder.map((tier, t) => `${t ? '<span class="gt">›</span>' : ''}<span class="tier">${tier.map(k => `<span class="pill">${esc(LADDER_LABEL(k))}</span>`).join('=')}
      <button class="b" data-act="lMove" data-i="${i}" data-t="${t}" data-d="-1" title="Rank higher" ${t === 0 ? 'disabled' : ''}>‹</button><button class="b" data-act="lMove" data-i="${i}" data-t="${t}" data-d="1" title="Rank lower" ${t === p.ladder.length - 1 ? 'disabled' : ''}>›</button>
      ${t > 0 ? `<button class="b" data-act="lTie" data-i="${i}" data-t="${t}" title="Tie with the tier before">=</button>` : ''}${tier.length > 1 ? `<button class="b" data-act="lSplit" data-i="${i}" data-t="${t}" title="Split">÷</button>` : ''}</span>`).join('')}</div>`;
  }
  function itemsWithChoices(r) {
    const d = derive(S);
    const out = [];
    for (const w of D.works) {
      const prof = profileFor(w, S); if (!prof.entries.some(e => e.rendition === r.id && e.sourcing !== 'derive')) continue;
      for (const item of w.items) {
        if (item.cands.length < 2) continue;
        const rt = runtimeOf(item);
        const views = item.cands.map(ci => candView(D.cands[ci], w.type, S, d, rt));
        const j = E.judge(views, r, d.settingsLike);
        if (j.ranked.length >= 2) out.push({ w, item, j, views });
      }
    }
    return out;
  }
  function prioPreview(r) {
    const xs = itemsWithChoices(r);
    if (!xs.length) return '<p class="small muted">No Item in your library has two releases that both pass this Rendition.</p>';
    const pick = xs.find(x => x.w.key + '|' + x.item.key === view.prioItem) || xs[0];
    const sel = `<select data-act="prioItem">${xs.map(x => `<option value="${esc(x.w.key + '|' + x.item.key)}" ${x === pick ? 'selected' : ''}>${esc(x.w.title)} ${x.item.key === 'movie' ? '' : x.item.key} (${x.j.ranked.length} pass)</option>`).join('')}</select>`;
    return `<div class="row">${sel}<span class="small muted">${xs.length} Items in your library have two or more releases that pass ${esc(r.name)}.</span></div>${rankTable(pick.j, r)}`;
  }
  function rankTable(j, r) {
    const d = derive(S);
    const ranked = j.ranked.map(x => x.v);
    const win = ranked[0];
    const lose = new Map(j.vsWinner.map(x => [x.v, x]));
    const rows = [['quality', 'Quality (Rendition order)', v => Q[v.quality].name], ['revision', 'Revision', v => v.revision || 'none']]
      .concat(r.priorities.map((p, i) => [i, E.MEASURES[p.measure].label, v => E.MEASURES[p.measure].show(v, { sizes: S.sizes })]));
    const head = `<tr><th>Step</th>${ranked.map((v, i) => `<th class="small"><span class="rel">${esc(v.name)}</span> ${i === 0 ? '<span class="pill good">grab</span>' : ''}</th>`).join('')}</tr>`;
    const order = ['quality', 'revision', ...r.priorities.map((_, i) => i)];
    const body = rows.map(([key, label, show]) => `<tr><td class="small"><b>${label}</b></td>${ranked.map((v, ci) => {
      let cls = '';
      if (ci > 0) { const L = lose.get(v); const at = order.indexOf(L.at); const me = order.indexOf(key); if (me === at) cls = 'cell-lost'; else if (me > at) cls = 'cell-after'; }
      else if (j.vsWinner.some(L => L.at === key)) cls = 'cell-won';
      return `<td class="small ${cls}">${esc(show(v))}</td>`;
    }).join('')}</tr>`).join('');
    const sentences = j.vsWinner.map(L => `<li><b>${esc(short(win))}</b> over <b>${esc(short(L.v))}</b>. ${esc(E.explain(win, L.v, L, r, { sizes: S.sizes }))}${flipHint(win, L.v, r)}</li>`).join('');
    const gated = j.gated.length ? `<details><summary>${j.gated.length} filtered out by hard rules</summary><ul class="small">${j.gated.map(g => `<li><span class="rel">${esc(g.v.name)}</span> → ${esc(g.fails.map(f => f.text).join('; '))}</li>`).join('')}</ul></details>` : '';
    return `<div class="scroll"><table>${head}${body}</table></div><ul class="small">${sentences}</ul>${gated}`;
  }
  const short = v => (v.group || '?') + ' ' + (v.quality ? Q[v.quality].name : '');
  function flipHint(a, b, r) {
    const f = E.whatWouldFlip(a, b, r, S);
    return f && f.length ? ` <span class="muted">What would make it win: ${f.map(x => esc(x.move)).join(', or ')}.</span>` : '';
  }

  /* Variant B: teach the order by choosing between real pairs */
  function teachPairs(r) {
    const d = derive(S);
    const byQ = new Map();
    for (const w of D.works) {
      const prof = profileFor(w, S); if (!prof.entries.some(e => e.rendition === r.id)) continue;
      for (const item of w.items) for (const fi of item.held) {
        const v = fileView(D.files[fi], w.type, S, d); if (!v || E.gate(v, r, d.settingsLike).length) continue;
        v.where = w.title + (item.key === 'movie' ? '' : ' ' + item.key);
        const sig = r.priorities.map(p => E.MEASURES[p.measure].show(v, { sizes: S.sizes }).replace(/[0-9.]+ MB\/min.*/, m => Math.round(parseFloat(m) / 25) * 25 + ' MB/min')).join('|');
        if (!byQ.has(v.quality)) byQ.set(v.quality, new Map());
        if (!byQ.get(v.quality).has(sig)) byQ.get(v.quality).set(sig, v);
      }
    }
    const pairs = [];
    for (const m of byQ.values()) {
      const vs = [...m.values()];
      for (let a = 0; a < vs.length; a++) for (let b = a + 1; b < vs.length; b++) {
        const A = vs[a], B = vs[b];
        const diffs = r.priorities.map((p, i) => ({ i, m: p.measure, c: E.MEASURES[p.measure].cmp(A, B, p, { sizes: S.sizes }) })).filter(z => z.c !== 0);
        if (new Set(diffs.map(z => Math.sign(z.c))).size === 2) pairs.push({ x: { w: { title: A.where + ' vs ' + B.where }, item: { key: 'movie' } }, A, B, diffs, key: A.file.p + '|' + B.file.p });
      }
    }
    pairs.sort((p, q) => q.diffs.length - p.diffs.length || p.key.localeCompare(q.key));
    return pairs.slice(0, 60);
  }
  function prioTeach(r) {
    const pairs = teachPairs(r);
    const answered = view.teach.filter(t => t.r === r.id);
    const next = pairs.find(p => !answered.some(t => t.key === p.key));
    const constraints = answered.map(t => t.constraint);
    const solved = solveOrder(r, constraints);
    const card = next ? `<div class="card"><h3>Two real ${esc(Q[next.A.quality].name)} files from your library. Which kind would you rather have in ${esc(r.name)}?</h3>
      <div class="grid2">${[next.A, next.B].map((v, k) => `<div class="card"><span class="small muted">${esc(v.where)}</span><span class="rel">${esc(v.name)}</span><table class="small">${next.diffs.map(z => `<tr><td>${E.MEASURES[z.m].label}</td><td><b>${esc(E.MEASURES[z.m].show(v, { sizes: S.sizes }))}</b></td></tr>`).join('')}</table><button class="b primary" data-act="teachPick" data-k="${k}">This one</button></div>`).join('')}</div>
      <p class="small muted">Different titles on purpose: your library has almost no single Item with two releases of the same quality that trade one priority against another. Judge the attributes, not the movie. Only the priorities where these two differ are shown. Your pick means something it wins on matters more than everything it loses on.</p></div>` : `<div class="card"><p>${pairs.length ? 'You answered every pair your library offers for this Rendition.' : 'Your library has no pair of passing releases for this Rendition that trade one priority against another.'}</p></div>`;
    return `${card}<div class="card"><h3>Order learned from ${answered.length} answer${answered.length === 1 ? '' : 's'}</h3>
      ${solved.conflict ? `<div class="why bad">Your answers conflict: ${esc(solved.conflict)}. Lexicographic priorities can't satisfy both. This is the case for a bounded trade rule, or for a tie band.</div>` : ''}
      <ol>${solved.order.map(m => `<li>${E.MEASURES[m].label}</li>`).join('')}</ol>
      <div class="row"><button class="b primary" data-act="teachApply" ${solved.conflict ? 'disabled' : ''}>Use this order</button><button class="b" data-act="teachReset">Start over</button><span class="small muted">${pairs.length} trade-off pairs available from your library.</span></div></div>`;
  }
  function solveOrder(r, constraints) {
    const ms = r.priorities.map(p => p.measure);
    // each answer: at least one priority the chosen file wins on must rank above every priority it loses on
    const order = [];
    const remaining = ms.slice();
    const req = constraints;
    while (remaining.length) {
      const blocked = new Set();
      for (const c of req) {
        if (c.win.some(w => order.includes(w))) continue;
        for (const l of c.lose) blocked.add(l);
      }
      const free = remaining.filter(m => !blocked.has(m));
      if (!free.length) {
        const c = req.find(c => !c.win.some(w => order.includes(w)));
        return { order: order.concat(remaining), conflict: `one answer needs ${c.win.map(m => E.MEASURES[m].label).join(' or ')} above ${c.lose.map(m => E.MEASURES[m].label).join(', ')}, and another needs the opposite` };
      }
      const pick = free.sort((a, b) => ms.indexOf(a) - ms.indexOf(b))[0];
      order.push(pick); remaining.splice(remaining.indexOf(pick), 1);
    }
    return { order };
  }

  /* ---- profiles ---- */
  function renderProfiles() {
    const rOpts = sel => S.renditions.map(r => `<option value="${r.id}" ${r.id === sel ? 'selected' : ''}>${esc(r.name)}</option>`).join('');
    const cards = S.profiles.map((p, pi) => {
      const works = D.works.filter(w => profileFor(w, S).id === p.id).length;
      return `<div class="card"><div class="row"><input type="text" data-act="prName" data-p="${pi}" value="${esc(p.name)}"><span class="pill">${p.contentType}</span><span class="small muted">${works} Works use it</span></div>
      <table><thead><tr><th>Rendition</th><th>Sourcing</th><th>Derived from</th><th class="n">Grace days</th><th></th></tr></thead><tbody>${p.entries.map((e, ei) => `<tr>
        <td><select data-act="peR" data-p="${pi}" data-e="${ei}">${rOpts(e.rendition)}</select></td>
        <td><select data-act="peS" data-p="${pi}" data-e="${ei}">${[['download', 'Download'], ['derive', 'Derive only'], ['download_then_derive', 'Download, derive after a grace period']].map(([v, l]) => `<option value="${v}" ${e.sourcing === v ? 'selected' : ''}>${l}</option>`).join('')}</select></td>
        <td>${e.sourcing === 'download' ? '<span class="muted">n/a</span>' : `<select data-act="peF" data-p="${pi}" data-e="${ei}">${rOpts(e.from)}</select>`}</td>
        <td class="n">${e.sourcing === 'download_then_derive' ? `<input type="number" min="0" data-act="peG" data-p="${pi}" data-e="${ei}" value="${e.graceDays ?? 14}">` : ''}</td>
        <td><button class="b" data-act="peDel" data-p="${pi}" data-e="${ei}">remove</button></td></tr>`).join('')}</tbody></table>
      <div class="row"><button class="b" data-act="peAdd" data-p="${pi}">Add a Rendition</button><label class="small">Editions wanted <select data-act="prEd" data-p="${pi}"><option value="any" ${p.editions === 'any' ? 'selected' : ''}>Any cut</option><option value="standard" ${p.editions === 'standard' ? 'selected' : ''}>Standard only</option><option value="theatrical,extended" ${p.editions === 'theatrical,extended' ? 'selected' : ''}>Theatrical and Extended</option></select></label></div>
      ${p.entries.some(e => e.sourcing !== 'download' && e.from === e.rendition) ? '<div class="why bad small">A Rendition can\'t derive from itself.</div>' : ''}</div>`;
    }).join('');
    const def = t => `<select data-act="defP" data-t="${t}">${S.profiles.filter(p => p.contentType === t).map(p => `<option value="${p.id}" ${S.defaults[t] === p.id ? 'selected' : ''}>${esc(p.name)}</option>`).join('')}</select>`;
    const ov = Object.entries(S.workOverrides);
    return `<h2>Rendition profiles</h2><p class="lead">A profile is the set of Renditions a Work wants and how each is sourced. Every Work has one, defaulting per Content Type. Requests never change a Work's profile.</p>
      <div class="card row"><span>Default for movies ${def('movie')}</span><span>Default for series ${def('series')}</span><button class="b" data-act="addProf" data-t="movie">New movie profile</button><button class="b" data-act="addProf" data-t="series">New series profile</button></div>
      ${cards}
      <div class="card"><h3>Per-Work overrides</h3>${ov.length ? `<ul>${ov.map(([k, p]) => `<li>${esc(D.works.find(w => w.key === k)?.title || k)} → ${esc(S.profiles.find(x => x.id === p)?.name)} <button class="b" data-act="ovDel" data-k="${esc(k)}">remove</button></li>`).join('')}</ul>` : '<p class="small muted">None. Set one from "Try it on a title".</p>'}</div>`;
  }

  /* ---- qualities and sizes ---- */
  function renderQualities() {
    const d = derive(S);
    const dist = {};
    for (const w of D.works) for (const item of w.items) for (const fi of item.held) {
      const v = fileView(D.files[fi], w.type, S, d); if (!v || !v.quality || !v.mbPerMin) continue;
      (dist[v.quality] = dist[v.quality] || []).push(v.mbPerMin);
    }
    const pct = (xs, p) => { const s = xs.slice().sort((a, b) => a - b); return s[Math.min(s.length - 1, Math.floor(p * s.length))]; };
    const trashS = (D.trash && D.trash.sizes) || {};
    return `<h2>Qualities and sizes</h2><p class="lead">The quality catalog is library-wide. Sizes are megabytes per minute of runtime, so a 3-hour film and a 22-minute episode are judged alike. Defaults come from TRaSH Guides' quality sizes (${esc(D.meta.trashCommit)}). The right side shows what your library actually holds.</p>
      <div class="card scroll"><table><thead><tr><th>Quality</th><th class="n">Min</th><th class="n">Preferred</th><th class="n">Max</th><th class="n">TRaSH min / pref / max</th><th class="n">Your files</th><th class="n">p10</th><th class="n">median</th><th class="n">p90</th><th class="n">Your files outside min..max</th></tr></thead><tbody>${E.QUALITIES.map(q => {
        const s = S.sizes[q.id] || {}; const t = trashS[q.id]; const xs = dist[q.id] || [];
        const out = xs.filter(x => (s.min && x < s.min) || (s.max && x > s.max)).length;
        return `<tr><td>${q.name}${q.cinema ? ' <span class="pill warn">cinema</span>' : ''}</td>${['min', 'preferred', 'max'].map(k => `<td class="n"><input type="number" data-act="sz" data-q="${q.id}" data-k="${k}" value="${s[k] ?? ''}"></td>`).join('')}
        <td class="n small muted">${t ? `${t.min} / ${t.preferred ?? 'no limit'} / ${t.max ?? 'no limit'}` : 'n/a'}</td><td class="n">${xs.length || ''}</td><td class="n">${xs.length ? pct(xs, 0.1).toFixed(0) : ''}</td><td class="n">${xs.length ? pct(xs, 0.5).toFixed(0) : ''}</td><td class="n">${xs.length ? pct(xs, 0.9).toFixed(0) : ''}</td><td class="n">${xs.length ? (out ? `<span class="pill bad">${out}</span>` : '0') : ''}</td></tr>`;
      }).join('')}</tbody></table></div>`;
  }

  /* ---- knowledge ---- */
  function renderKnowledge() {
    const d = derive(S);
    const all = [];
    if (S.knowledge.base || true) for (const c of E.BASE_PACK.claims) all.push({ pack: 'base', c });
    if (D.trash && D.trash.pack) for (const c of D.trash.pack.claims) all.push({ pack: 'trash', c });
    const fireCount = new Map();
    for (const r of D.releases) { const it = d.interp(r.n, r.type); for (const id of it.fired) fireCount.set(id, (fireCount.get(id) || 0) + 1); for (const h of it.hazards) fireCount.set(h.id, (fireCount.get(h.id) || 0) + 1); for (const h of it.reputations) fireCount.set(h.id, (fireCount.get(h.id) || 0) + 1); }
    const q = view.claimFilter.toLowerCase();
    const shown = all.filter(x => (view.claimKind === 'all' || x.c.kind === view.claimKind) && (!q || (x.c.label + ' ' + x.c.id).toLowerCase().includes(q)));
    const exampleResults = E.BASE_PACK.claims.flatMap(c => (c.examples || []).map(ex => ({ c, ex, ok: c.kind === 'term' ? E.interpret(ex, [E.BASE_PACK]).fired.includes(c.id) : E.interpret(ex, [E.BASE_PACK]).hazards.some(h => h.id === c.id) })));
    const failing = exampleResults.filter(x => !x.ok);
    const off = new Set(S.knowledge.disabled);
    const rows = shown.slice(0, 400).map(({ pack, c }) => `<tr class="${off.has(c.id) ? 'cell-after' : ''}"><td><label class="chk"><input type="checkbox" data-act="claimOn" data-id="${esc(c.id)}" ${off.has(c.id) ? '' : 'checked'}></label></td>
      <td><span class="pill">${pack}</span> <span class="pill ${c.kind === 'hazard' ? 'bad' : c.kind === 'reputation' ? 'acc' : ''}">${c.kind}</span></td><td><b>${esc(c.label)}</b><div class="small muted mono">${esc(c.id)}</div></td>
      <td class="small">${c.kind === 'term' ? (c.sets || []).map(s => `${s.key}=${s.value}`).join(', ') : c.kind === 'reputation' ? 'prior ' + esc(c.prior) : `<select data-act="sev" data-id="${esc(c.id)}">${['reject', 'avoid', 'note'].map(s => `<option ${(S.knowledge.severity[c.id] || c.severity) === s ? 'selected' : ''}>${s}</option>`).join('')}</select>${c.severityBasis ? `<div class="muted">${esc(c.severityBasis)}</div>` : ''}`}${(c.mapNotes || []).map(n => `<div class="muted">${esc(n)}</div>`).join('')}</td>
      <td><select data-act="conf" data-id="${esc(c.id)}">${['certain', 'high', 'medium', 'low'].map(s => `<option ${(S.knowledge.confidence[c.id] || c.confidence) === s ? 'selected' : ''}>${s}</option>`).join('')}</select></td>
      <td class="n">${fireCount.get(c.id) || 0}</td><td class="n small">${d.claimStats[c.id] ? `<span class="${d.claimStats[c.id].ok === d.claimStats[c.id].n ? '' : 'pill bad'}">${d.claimStats[c.id].ok}/${d.claimStats[c.id].n}</span>` : ''}</td><td class="small mono">${esc(whenText(c.when))}</td></tr>`).join('');
    const tp = D.trash && D.trash.pack;
    const byKind = k => tp ? tp.claims.filter(c => c.kind === k).length : 0;
    return `<h2>Knowledge packs</h2><p class="lead">Packs hold facts about release names: what a token means (terms), what a group tends to produce (reputations), and what to avoid (hazards). They carry no scores, so no pack decides your taste. Your priorities do. Later packs and your local edits win, and conflicts are shown, never resolved silently.</p>
      <div class="grid2">
        <div class="card"><div class="row"><label class="chk"><input type="checkbox" data-act="packOn" data-p="base" ${S.knowledge.base ? 'checked' : ''}><h3>SlipStream base pack</h3></label><span class="pill">${E.BASE_PACK.version}</span></div>
          <p class="small">${E.BASE_PACK.claims.length} claims, compiled in, works offline. ${exampleResults.length} example tests, ${failing.length ? `<span class="pill bad">${failing.length} failing</span>` : '<span class="pill good">all pass</span>'}.</p>
          ${failing.length ? `<ul class="small">${failing.map(f => `<li>${esc(f.c.label)}: <span class="rel">${esc(f.ex)}</span></li>`).join('')}</ul>` : ''}</div>
        <div class="card"><div class="row"><label class="chk"><input type="checkbox" data-act="packOn" data-p="trash" ${S.knowledge.trash ? 'checked' : ''} ${tp ? '' : 'disabled'}><h3>TRaSH Guides, converted</h3></label><span class="pill">${esc(D.meta.trashCommit)}</span></div>
          ${tp ? `<p class="small">${byKind('term')} terms, ${byKind('hazard')} hazards, ${byKind('reputation')} reputations from ${esc(tp.source ? tp.source.repo : 'TRaSH')}. ${(tp.untranslated || []).length} custom formats didn't translate. Scores were read only to set hazard severity and to suggest an order. They aren't stored.</p>
          <div class="row">${['term', 'hazard', 'reputation'].map(k => `<label class="chk"><input type="checkbox" data-act="trashKind" data-k="${k}" ${S.knowledge.trashKinds[k] ? 'checked' : ''}>${k}s</label>`).join('')}</div>
          ${D.trash.compat ? `<p class="small muted">${esc(D.trash.compat)}</p>` : ''}
          <details><summary>Untranslated (${(tp.untranslated || []).length})</summary><ul class="small">${(tp.untranslated || []).map(u => `<li>${esc(u.app)} · ${esc(u.cf)}: ${esc(u.reason)}</li>`).join('')}</ul></details>
          <details><summary>Suggested priority order from TRaSH profiles</summary>${(D.trash.profiles || []).map(p => `<div class="small"><b>${esc(p.app)} · ${esc(p.name)}</b>: ${esc((p.suggested || []).join(' › '))}</div>`).join('')}</details>` : '<p class="small muted">Converter output not loaded.</p>'}
        </div>
      </div>
      <div class="card"><h3>Replay against your library</h3>${replay()}</div>
      <div class="card"><div class="row"><h3>Claims</h3><input type="text" placeholder="Filter" data-act="claimFilter" value="${esc(view.claimFilter)}"><select data-act="claimKind">${['all', 'term', 'hazard', 'reputation'].map(k => `<option ${view.claimKind === k ? 'selected' : ''}>${k}</option>`).join('')}</select><span class="small muted">${shown.length} shown${shown.length > 400 ? ', first 400 listed' : ''}. "Fires" counts your ${D.releases.length} release names.</span></div>
      <div class="scroll"><table><thead><tr><th>On</th><th>Pack</th><th>Claim</th><th>Says</th><th>Confidence</th><th class="n">Fires</th><th class="n">Held on your files</th><th>When</th></tr></thead><tbody>${rows}</tbody></table></div></div>`;
  }
  const whenText = when => (when || []).map(s => `${s.negate ? 'NOT ' : ''}${s.required ? '' : '~'}${s.implementation.replace('Specification', '')}(${s.decoded ?? s.value})`).join(' & ').slice(0, 160);
  function replay() {
    const a = decisions(BASE), b = decisions(S);
    const changes = [];
    for (const [k, x] of b) { const y = a.get(k); if (!y || y.winner !== x.winner) changes.push({ x, from: y ? y.winner : null }); }
    const I0 = impact(BASE), I1 = impact(S);
    const upg = S.renditions.reduce((n, r) => n + ((I1.per[r.id]?.below || 0) - (I0.per[r.id]?.below || 0)), 0);
    if (!changes.length && !upg) return `<p class="small muted">Against the baseline, no pick changes across ${b.size} Item and Rendition decisions that have two or more releases. Turn a pack, kind or claim on or off to see its effect before you accept it.</p>`;
    return `<p class="small">${changes.length} of ${b.size} picks change. ${upg ? `${upg > 0 ? '+' : ''}${upg} Items become ${upg > 0 ? 'upgradable' : 'no longer upgradable'}.` : ''}</p><div class="scroll"><table><thead><tr><th>Item</th><th>Rendition</th><th>Was</th><th>Now</th></tr></thead><tbody>${changes.slice(0, 200).map(c => `<tr><td>${esc(c.x.w.title)} ${c.x.item.key === 'movie' ? '' : esc(c.x.item.key)}</td><td>${esc(c.x.r.name)}</td><td class="rel">${esc(c.from || 'nothing passes')}</td><td class="rel">${esc(c.x.winner || 'nothing passes')}</td></tr>`).join('')}</tbody></table></div>`;
  }

  /* ---- trust ---- */
  function renderTrust() {
    const d = derive(S);
    const groups = Object.entries(d.trust).map(([g, fams]) => {
      const tot = Object.values(fams).reduce((a, f) => ({ ok: a.ok + f.ok, n: a.n + f.n }), { ok: 0, n: 0 });
      return { g, fams, tot };
    }).sort((a, b) => b.tot.n - a.tot.n);
    const fams = ['picture', 'audio', 'resolution', 'codec', 'language'];
    const rel = (g, f) => E.reliability(g, f, d.trust, d.settingsLike);
    const refuted = d.observations.flatMap(o => o.checks.filter(c => c.verdict === 'refuted').map(c => ({ o, c })));
    const unobs = d.observations.flatMap(o => o.checks.filter(c => c.verdict === 'unobservable').map(c => ({ o, c })));
    const t = S.trust, v = S.verification;
    return `<h2>Trust and verification</h2><p class="lead">A release name is a set of claims. Every download is probed before import, and the probe confirms or refutes each claim. A group earns trust per family of claims from your own files. Below are the real results of probing ${d.observations.length} releases in your Downloads.</p>
      <div class="grid2"><div class="card"><h3>How much a claim is worth</h3>
        <label>A claim counts only if its confidence is at least <input type="number" step="0.05" min="0" max="1" data-act="tw" value="${t.weakBelow}"></label>
        <label>A group's prior is worth <input type="number" min="0" data-act="tk" value="${t.priorStrength}"> of your own checks</label>
        <label class="chk"><input type="checkbox" data-act="tCal" ${t.calibrateTerms ? 'checked' : ''}>Calibrate each term's confidence from your probes (the pack's level counts as a prior of the same weight)</label>
        <div class="row">${Object.keys(t.priors).map(k => `<label class="small">${k} <input type="number" step="0.01" min="0" max="1" data-act="tp" data-k="${k}" value="${t.priors[k]}"></label>`).join('')}</div>
        <p class="small muted">Reliability = (checks held + prior × weight) ÷ (checks + weight). A claim's confidence = the term's confidence × the group's reliability for that family. Picture and audio claims below the threshold are judged at their worst plausible value.</p></div>
      <div class="card"><h3>Verification policy</h3>
        <label class="chk"><input type="checkbox" data-act="vf" data-k="probeBeforeImport" ${v.probeBeforeImport ? 'checked' : ''}>Probe every download before import</label>
        <label>When a hard-filter claim is refuted <select data-act="vs" data-k="onMismatch"><option value="reject_import" ${v.onMismatch === 'reject_import' ? 'selected' : ''}>Don't import into that Rendition</option><option value="import_flagged" ${v.onMismatch === 'import_flagged' ? 'selected' : ''}>Import it and flag it</option></select></label>
        <label class="chk"><input type="checkbox" data-act="vf" data-k="blocklist" ${v.blocklist ? 'checked' : ''}>Blocklist that release for the Item</label>
        <label>The torrent then <select data-act="vs" data-k="seedPolicy"><option value="seed_to_goal" ${v.seedPolicy === 'seed_to_goal' ? 'selected' : ''}>Seeds to the tracker's goal, then is removed</option><option value="remove_now" ${v.seedPolicy === 'remove_now' ? 'selected' : ''}>Is removed now</option></select></label>
        <label class="chk"><input type="checkbox" data-act="vf" data-k="probationUpgrades" ${v.probationUpgrades ? 'checked' : ''}>Upgrades are probationary: the held file is deleted only after the new one is probed and still better</label>
        <label class="chk"><input type="checkbox" data-act="vf" data-k="heuristicsAdvisory" ${v.heuristicsAdvisory ? 'checked' : ''}>Heuristic checks (upscale, fake HDR) only add notes</label>
        <label>Held files whose source no name reveals count as <select data-act="vs" data-k="unknownSource"><option value="unknown" ${v.unknownSource === 'unknown' ? 'selected' : ''}>Unknown (fill nothing)</option><option value="webdl" ${v.unknownSource === 'webdl' ? 'selected' : ''}>WEB-DL at their resolution</option><option value="webrip" ${v.unknownSource === 'webrip' ? 'selected' : ''}>WEBRip at their resolution</option></select></label>
        <p class="small muted">Checked facts: resolution, codec, Dolby Vision presence and profile, HDR10, HDR10+, HLG, Atmos, audio format, channels, tagged audio languages. Source and streaming service can't be probed.</p></div></div>
      <div class="card"><h3>Groups, from your probes</h3><div class="scroll"><table><thead><tr><th>Group</th><th>Prior</th>${fams.map(f => `<th class="n">${f}</th>`).join('')}<th class="n">reliability (picture)</th></tr></thead><tbody>${groups.map(({ g, fams: fm }) => {
        const pr = rel(g, 'picture');
        return `<tr><td class="mono">${esc(g)}</td><td class="small">${esc(pr.prior.prior)}${pr.prior.from ? ` <span class="muted">(${esc(pr.prior.from)})</span>` : ''}</td>${fams.map(f => fm[f] ? `<td class="n">${fm[f].ok === fm[f].n ? '' : '<span class="pill bad">'}${fm[f].ok}/${fm[f].n}${fm[f].ok === fm[f].n ? '' : '</span>'}</td>` : '<td class="n muted">·</td>').join('')}<td class="n">${pr.value.toFixed(2)}</td></tr>`;
      }).join('')}</tbody></table></div></div>
      <div class="card"><h3>${refuted.length} refuted claim${refuted.length === 1 ? '' : 's'}</h3><div class="scroll"><table><thead><tr><th>Release</th><th>Claimed</th><th>Probe found</th></tr></thead><tbody>${refuted.map(({ o, c }) => `<tr><td class="rel">${esc(o.name)}</td><td>${esc(c.claimed)}</td><td>${esc(c.actual)}</td></tr>`).join('')}</tbody></table></div></div>
      <div class="card"><h3>${unobs.length} claims the probe couldn't check</h3><p class="small muted">A header-only MediaInfo parse sometimes stops before a stream's extension: DTS-HD MA can show as its DTS core, and a TrueHD or DD+ track can show no Atmos marker. These claims count neither for nor against the group. For held files, the name fills in within the same codec family.</p><details><summary>List</summary><table class="small"><tbody>${unobs.map(({ o, c }) => `<tr><td class="rel">${esc(o.name)}</td><td>${esc(c.claimed)}</td><td>${esc(c.actual)}</td></tr>`).join('')}</tbody></table></details></div>
      <div class="card"><h3>Present but not named</h3>${extrasTable(d)}</div>`;
  }
  function extrasTable(d) {
    const rows = [];
    for (const r of D.releases) {
      const f = r.files.find(x => x.f && x.f.video); if (!f) continue;
      const v = E.verify(d.interp(r.n, r.type), E.factsFromProbe(f.f));
      if (v.extras.length) rows.push({ n: r.n, x: v.extras });
    }
    return `<p class="small muted">Not a refutation. A name that undersells its file ranks lower than it should. ${rows.length} releases.</p><div class="scroll"><table><tbody>${rows.map(r => `<tr><td class="rel">${esc(r.n)}</td><td class="small">${esc(r.x.join(', '))}</td></tr>`).join('')}</tbody></table></div>`;
  }

  /* ---- timing ---- */
  function releaseLog() {
    const d = derive(S);
    const log = [];
    const r = rendById(view.rendition) || S.renditions[0];
    for (const w of D.works) for (const item of w.items) {
      const cs = item.cands.map(ci => D.cands[ci]).filter(c => c.t);
      if (cs.length < 2) continue;
      const stg = cs.map(c => { const it = d.interp(c.n, w.type); const q = E.qualityOf(it.rel.source, it.rel.res); return { c, q, s: q ? E.stageOf(q, r) : null, svc: (it.beliefs.service && [...it.beliefs.service.keys()][0]) || null }; }).filter(x => x.s);
      stg.sort((a, b) => a.c.t - b.c.t);
      const first = stg[0]; if (!first) continue;
      for (const x of stg.slice(1)) if (E.STAGES.indexOf(x.s) > E.STAGES.indexOf(first.s) || (x.s === first.s && E.qualityRank(r, x.q) > E.qualityRank(r, first.q)))
        log.push({ w, item, from: first.s, to: x.s, fromQ: first.q, toQ: x.q, lagHours: (x.c.t - first.c.t) / 3600, cohort: first.svc || w.type, a: first.c.n, b: x.c.n, wasted: first.c.s || 0 });
    }
    return log;
  }
  function renderTiming() {
    const r = rendById(view.rendition) || S.renditions[0];
    const tabs = S.renditions.map(x => `<button class="${x.id === r.id ? 'on' : ''}" data-act="pickRend" data-id="${x.id}">${esc(x.name)}</button>`).join('');
    const t = r.timing;
    const log = releaseLog();
    const wasted = log.reduce((a, x) => a + x.wasted, 0);
    const pairs = new Map();
    for (const x of log) { const k = x.from + '→' + x.to; pairs.set(k, (pairs.get(k) || 0) + 1); }
    const fc = [...pairs.keys()].map(k => { const [a, b] = k.split('→'); return { k, f: E.forecast(log, a, b, null, t) }; });
    return `<h2>Timing (Horizon)</h2><p class="lead">Each Item × Edition × Rendition is a Target with an Arc: Open, Holding, Settled, Closed or Dormant. Patience decides whether to take a stepping stone or wait for what usually follows. Forecasts come from the Release Log, which records when each release first appears on your indexers.</p>
      <div class="tabs">${tabs}</div>
      <div class="grid2"><div class="card"><h3>Patience for ${esc(r.name)}</h3>
        ${Object.entries(E.PATIENCE).map(([k, p]) => `<label class="chk"><input type="radio" name="pat" data-act="tPat" data-k="${k}" ${t.patience === k ? 'checked' : ''}><b>${p.label}</b> <span class="small muted">${p.text}</span></label>`).join('')}
        <label>Step window <input type="number" min="1" data-act="tNum" data-k="stepWindowHours" value="${t.stepWindowHours}"> hours</label>
        <label>Wait when the next stage is at least <input type="number" step="0.05" min="0" max="1" data-act="tNum" data-k="threshold" value="${t.threshold}"> likely within the window</label>
        <label>Forecast only from <input type="number" min="1" data-act="tNum" data-k="minSamples" value="${t.minSamples}"> or more examples; fewer acts as Eager</label>
        <label>Close the Arc <input type="number" min="1" data-act="tNum" data-k="horizonDays" value="${t.horizonDays}"> days after the last anchor with no stage pending</label>
        <p class="small muted">Stages for this Rendition: ${E.STAGES.map(s => E.STAGE_LABEL[s]).join(' › ')}. Its final stage is ${E.STAGE_LABEL[E.terminalStage(r)]}. Portal requests can carry a watch-by date that caps any wait.</p></div>
      <div class="card"><h3>What your library already shows</h3>
        <p class="small">The Release Log doesn't exist yet, so no real forecast is possible. As a stand-in, this uses <b>when each download finished</b> (file times), which lags when releases appeared. ${log.length} times an Item got a later, better download after an earlier one. The earlier downloads total <b>${tb(wasted)}</b>.</p>
        <table class="small"><thead><tr><th>Step</th><th class="n">n</th><th class="n">within window</th><th class="n">median lag</th><th>Verdict at these settings</th></tr></thead><tbody>${fc.map(({ k, f }) => `<tr><td>${k.split('→').map(s => E.STAGE_LABEL[s]).join(' › ')}</td><td class="n">${f.n}</td><td class="n">${f.ok ? Math.round(f.p * 100) + '%' : ''}</td><td class="n">${f.ok ? f.median.toFixed(0) + ' h' : ''}</td><td>${!f.ok ? `<span class="pill warn">below ${t.minSamples} examples, acts as Eager</span>` : f.p >= t.threshold ? '<span class="pill good">would wait</span>' : '<span class="pill">would grab</span>'}</td></tr>`).join('')}</tbody></table></div></div>
      <div class="card"><h3>Interim downloads later replaced</h3><div class="scroll"><table><thead><tr><th>Item</th><th>First</th><th>Then</th><th class="n">Lag</th></tr></thead><tbody>${log.sort((a, b) => a.lagHours - b.lagHours).map(x => `<tr><td>${esc(x.w.title)} ${x.item.key === 'movie' ? '' : esc(x.item.key)}</td><td class="rel">${esc(x.a)}</td><td class="rel">${esc(x.b)}</td><td class="n">${x.lagHours < 48 ? x.lagHours.toFixed(1) + ' h' : (x.lagHours / 24).toFixed(0) + ' d'}</td></tr>`).join('')}</tbody></table></div></div>`;
  }

  /* ---- try it ---- */
  function renderTry() {
    const works = D.works.slice().sort((a, b) => a.title.localeCompare(b.title));
    const w = works.find(x => x.key === view.work) || works.find(x => x.items.some(i => i.cands.length > 1)) || works[0];
    const item = w.items.find(i => i.key === view.item) || w.items.find(i => i.cands.length) || w.items[0];
    const d = derive(S);
    const prof = profileFor(w, S);
    const rt = runtimeOf(item);
    const views = item.cands.map(ci => candView(D.cands[ci], w.type, S, d, rt));
    const held = item.held.map(i => fileView(D.files[i], w.type, S, d)).filter(Boolean);
    const arrProfiles = (D.trash && D.trash.profiles || []).filter(p => p.app === appOf(w.type));
    const arrFor = r => arrProfiles.find(p => p.name === view.arrProfile) || arrProfiles.find(p => p.name === (/2160/.test(r.cutoff) ? (w.type === 'movie' ? 'Remux + WEB 2160p' : 'WEB-2160p') : (w.type === 'movie' ? 'HD Bluray + WEB' : 'WEB-1080p'))) || arrProfiles[0];
    const ap0 = view.arrProfile ? arrProfiles.find(p => p.name === view.arrProfile) : null;
    const sections = prof.entries.map(e => {
      const r = rendById(e.rendition); if (!r) return '';
      if (e.sourcing === 'derive') {
        const src = rendById(e.from);
        const srcFill = src && held.some(v => E.gate(v, src, d.settingsLike).length === 0);
        const own = held.filter(v => E.gate(v, r, d.settingsLike).length === 0);
        return `<div class="card"><h3>${esc(r.name)} <span class="pill">derived from ${esc(src ? src.name : '?')}</span></h3><p class="small">${own.length ? `Already filled by a held file (${esc(own[0].file.p.split('/').slice(-1)[0])}), so nothing is derived. ` : ''}${srcFill ? `Derived by transcoding the held ${esc(src.name)} file. Releases are never searched for this Rendition.` : `Waits for ${esc(src ? src.name : '?')} to be filled, then transcodes it.`}</p></div>`;
      }
      const j = E.judge(views, r, d.settingsLike);
      const fillers = held.filter(v => E.gate(v, r, d.settingsLike).length === 0);
      const best = fillers.sort((a, b) => E.qualityRank(r, b.quality) - E.qualityRank(r, a.quality))[0];
      let up = '';
      if (j.winner) {
        const cand = j.winner.v;
        let wants = !best;
        if (!best) up = `<div class="why good">Nothing held fills ${esc(r.name)}. Grab <span class="rel">${esc(cand.name)}</span>.</div>`;
        else if (best.name === cand.name || best.file.s === cand.cand.s) up = `<div class="why">The held file is this same release.</div>`;
        else {
          const u = E.upgrade(best, cand, r, d.settingsLike);
          wants = u.action === 'upgrade';
          up = `<div class="why ${u.action === 'upgrade' ? 'good' : ''}"><b>${u.action === 'upgrade' ? 'Upgrade' : 'Keep the held file'}.</b> ${esc(u.why)}</div>`;
          if (u.action === 'upgrade' && cand.cand.f && S.verification.probationUpgrades) {
            const nf = E.factsFromProbe(cand.cand.f);
            const nv = E.viewFromFile(cand.name, nf, d.trust, d.settingsLike, d.packs[w.type]);
            const pr = E.probation(best, nv, r, d.settingsLike);
            up += `<div class="why ${pr.keepNew ? 'good' : 'bad'} small">Probation, using the real probe of that download: ${esc(pr.why)}</div>`;
          }
        }
        if (wants) {
          const arc = E.arcDecision({ rendition: r, held: best, best: cand, log: releaseLog(), cohort: null });
          up += `<div class="why warn small">Timing, ${esc(E.PATIENCE[r.timing.patience].label)} patience: ${esc(arc.action)}. ${esc(arc.why)}</div>`;
        }
      } else up = `<div class="why bad">No release passes ${esc(r.name)}.</div>`;
      let arr = '';
      const ap = ap0 || arrFor(r);
      if (ap && D.trash.cfIndex) {
        const scored = views.map(v => ({ v, s: E.arrScore(v.interp, ap, D.trash.cfIndex) }));
        const ok = scored.filter(x => x.s.total >= (ap.minFormatScore || 0) && ap.qualityRank && ap.qualityRank[x.v.quality] !== undefined);
        ok.sort((a, b) => (ap.qualityRank[b.v.quality] - ap.qualityRank[a.v.quality]) || (b.s.total - a.s.total));
        arr = `<details><summary>Sonarr/Radarr with TRaSH profile "${esc(ap.name)}" would pick: ${ok[0] ? esc(ok[0].v.name) : 'nothing'}</summary><table class="small"><thead><tr><th>Release</th><th class="n">Score</th><th>Formats matched</th></tr></thead><tbody>${scored.sort((a, b) => b.s.total - a.s.total).map(x => `<tr><td class="rel">${esc(x.v.name)}</td><td class="n">${x.s.total}</td><td>${esc(x.s.hits.map(h => `${h.name} ${h.score > 0 ? '+' : ''}${h.score}`).join(', '))}</td></tr>`).join('')}</tbody></table><p class="small muted">Ranked by the profile's quality order, then score. Minimum score ${ap.minFormatScore ?? 0}, upgrade until ${ap.cutoffFormatScore ?? 'n/a'}. Approximation: language and indexer-flag conditions read the name only, an unnamed language counts as English, and English is taken as the original language.</p></details>`;
      }
      return `<div class="card"><h3>${esc(r.name)} <span class="pill">${e.sourcing === 'download_then_derive' ? `download, derive after ${e.graceDays} days` : 'download'}</span></h3>
        <div class="small">Held: ${held.length ? held.map(v => `<div><span class="rel">${esc(v.file.p.split('/').slice(-1)[0])}</span> → ${E.gate(v, r, d.settingsLike).length ? `<span class="pill bad">doesn't fill</span> <span class="muted">${esc(E.gate(v, r, d.settingsLike).map(f => f.text).join('; '))}</span>` : `<span class="pill good">fills</span> <span class="muted">${Q[v.quality].name}, ${E.PICTURE_LABEL[v.picture]}${v.dvProfile ? '' : ''}, ${E.AUDIO_LABEL[v.audio]}</span>`}</div>`).join('') : 'nothing'}</div>
        ${j.ranked.length ? rankTable(j, r) : (j.gated.length ? `<details open><summary>All ${j.gated.length} releases filtered out</summary><ul class="small">${j.gated.map(g => `<li><span class="rel">${esc(g.v.name)}</span> → ${esc(g.fails.map(f => f.text).join('; '))}</li>`).join('')}</ul></details>` : '')}
        ${up}${arr}</div>`;
    }).join('');
    const wsel = `<select data-act="tryW">${works.map(x => `<option value="${esc(x.key)}" ${x === w ? 'selected' : ''}>${esc(x.title)} ${x.year || ''} · ${x.items.reduce((n, i) => n + i.cands.length, 0)} releases</option>`).join('')}</select>`;
    const isel = `<select data-act="tryI">${w.items.map(i => `<option value="${esc(i.key)}" ${i === item ? 'selected' : ''}>${esc(i.key)} · ${i.cands.length} rel · ${i.held.length} held</option>`).join('')}</select>`;
    const psel = `<select data-act="ovSet" data-w="${esc(w.key)}">${S.profiles.filter(p => p.contentType === w.type).map(p => `<option value="${p.id}" ${p.id === prof.id ? 'selected' : ''}>${esc(p.name)}</option>`).join('')}</select>`;
    const asel = arrProfiles.length ? `<label class="small">Compare with TRaSH profile <select data-act="arrP"><option value="">Matched to each Rendition</option>${arrProfiles.map(p => `<option ${p === ap0 ? 'selected' : ''}>${esc(p.name)}</option>`).join('')}</select></label>` : '';
    return `<h2>Try it on a title</h2><p class="lead">Candidates are the real releases in your Downloads for this Item. Judging uses only what the name and your trust table say, as it would before a grab. Probation then checks the winner against its real probe.</p>
      <div class="card row">${wsel}${isel}<label class="small">Profile ${psel}</label>${asel}</div>${sections}`;
  }

  /* ---- walkthroughs ---- */
  const WALKS = (window.WALKS || []);
  function renderWalk() {
    const tabs = WALKS.map((w, i) => `<button class="${view.walk === i ? 'on' : ''}" data-act="walk" data-i="${i}">${esc(w.title)}</button>`).join('');
    const w = WALKS[view.walk];
    if (!w) return '<h2>Walkthroughs</h2><p class="lead">None defined.</p>';
    return `<h2>Walkthroughs</h2><p class="lead">Each one resets the settings and then changes them one step at a time. Watch the strip at the top and the panel each step opens.</p><div class="tabs">${tabs}</div>
      <div class="card"><h3>${esc(w.title)}</h3><p>${w.text}</p><div class="row"><button class="b primary" data-act="walk" data-i="${view.walk}">Start this walkthrough</button><span class="small muted">Its steps dock at the bottom of the screen and stay there while each step opens the panel it changes.</span></div></div>`;
  }

  /* ---- state ---- */
  function renderState() {
    const df = diffPaths(DEFAULTS, S);
    return `<h2>Settings state</h2><p class="lead">The whole settings object this prototype edits, and how it differs from the defaults. This shape is the thing under review.</p>
      <div class="card"><h3>${df.length} changes from defaults</h3>${df.length ? `<pre class="json">${df.map(x => `${esc(x.path)}\n  <span class="diff-del">- ${esc(JSON.stringify(x.from))}</span>\n  <span class="diff-add">+ ${esc(JSON.stringify(x.to))}</span>`).join('\n')}</pre>` : ''}</div>
      <div class="card"><div class="row"><h3>Full settings</h3><button class="b" data-act="copy">Copy JSON</button></div><pre class="json" id="stateJson">${esc(JSON.stringify(S, null, 2))}</pre></div>`;
  }

  /* ---------------- actions ---------------- */
  const cur = () => rendById(view.rendition) || S.renditions[0];
  const move = (arr, i, d) => { const j = i + d; if (j < 0 || j >= arr.length) return; [arr[i], arr[j]] = [arr[j], arr[i]]; };
  const ACT = {
    nav: el => { view.section = el.dataset.id; },
    resetAll: () => { S = clone(BASE); },
    markBase: () => { BASE = clone(S); },
    pickRend: el => { view.rendition = el.dataset.id; },
    addRend: () => { const r = clone(S.renditions[0]); r.id = 'r' + Math.random().toString(36).slice(2, 7); r.name = 'New Rendition'; S.renditions.push(r); view.rendition = r.id; },
    delRend: () => { S.renditions = S.renditions.filter(r => r.id !== view.rendition); view.rendition = S.renditions[0].id; },
    rName: el => { cur().name = el.value; },
    rUpg: el => { cur().upgrades = el.checked; },
    rPast: el => { cur().pastCutoff = el.checked; },
    rTopN: el => { cur().minGainTopN = Math.max(1, +el.value || 1); },
    qMove: el => { move(cur().qualities, +el.dataset.i, +el.dataset.d); },
    qTie: el => { const q = cur().qualities, i = +el.dataset.i; q[i - 1] = q[i - 1].concat(q[i]); q.splice(i, 1); },
    qSplit: el => { const q = cur().qualities, i = +el.dataset.i; const t = q[i]; q.splice(i, 1, ...t.map(x => [x])); },
    qRemove: el => { const r = cur(); const t = r.qualities.splice(+el.dataset.i, 1)[0]; if (t.includes(r.cutoff)) r.cutoff = r.qualities.length ? r.qualities[r.qualities.length - 1][0] : null; },
    qCutoff: el => { cur().cutoff = el.dataset.id; },
    qAddSel: el => { if (!el.value) return; const r = cur(); r.qualities.push([el.value]); r.qualities.sort((a, b) => E.QUALITIES.findIndex(q => q.id === a[0]) - E.QUALITIES.findIndex(q => q.id === b[0])); },
    qPreset: el => {
      const r = cur(); const ids = r.qualities.flat().sort((a, b) => E.QUALITIES.findIndex(q => q.id === a) - E.QUALITIES.findIndex(q => q.id === b));
      if (el.dataset.p === 'aggressive') r.qualities = ids.map(x => [x]);
      if (el.dataset.p === 'resolution') { const by = new Map(); for (const id of ids) { const k = Q[id].res; if (!by.has(k)) by.set(k, []); by.get(k).push(id); } r.qualities = [...by.values()]; }
      if (el.dataset.p === 'balanced') { const out = []; for (const id of ids) { const last = out[out.length - 1]; const q = Q[id]; if (last && Q[last[0]].res === q.res && ['webdl', 'webrip'].includes(q.source) && ['webdl', 'webrip'].includes(Q[last[0]].source)) last.push(id); else out.push([id]); } r.qualities = out; }
      if (!r.qualities.flat().includes(r.cutoff)) r.cutoff = r.qualities[r.qualities.length - 1][0];
    },
    pic: el => { const f = cur().filters; f.picture = el.checked ? [...new Set(f.picture.concat(el.dataset.k))] : f.picture.filter(k => k !== el.dataset.k); },
    codec: el => { const f = cur().filters; const all = ['h265', 'h264', 'av1', 'vc1', 'mpeg2']; const s = new Set(f.codecs || all); el.checked ? s.add(el.dataset.k) : s.delete(el.dataset.k); f.codecs = s.size === all.length ? null : [...s]; },
    rCh: el => { cur().filters.minChannels = +el.value || 0; },
    rLang: el => { cur().filters.languages = el.value.split(/[ ,]+/).map(x => x.trim().toLowerCase()).filter(Boolean); },
    rLangUnnamed: el => { cur().filters.unnamedLanguageCounts = el.checked; },
    rSz: el => { const r = cur(); const o = r.sizeOverride[el.dataset.q] = r.sizeOverride[el.dataset.q] || {}; if (el.value === '') delete o[el.dataset.k]; else o[el.dataset.k] = +el.value; if (!Object.keys(o).length) delete r.sizeOverride[el.dataset.q]; },
    pMove: el => { move(cur().priorities, +el.dataset.i, +el.dataset.d); },
    pDel: el => { cur().priorities.splice(+el.dataset.i, 1); },
    pAdd: el => { if (!el.value) return; const p = { measure: el.value }; if (LADDERS[el.value]) p.ladder = clone(LADDERS[el.value]); if (['trust'].includes(el.value)) p.band = 0.1; if (el.value === 'size') p.band = 0.15; if (el.value === 'language') { p.wanted = ['en']; p.assumeUnnamedIsWanted = true; } cur().priorities.push(p); },
    pBand: el => { cur().priorities[+el.dataset.i].band = +el.value; },
    pLang: el => { cur().priorities[+el.dataset.i].wanted = el.value.split(/[ ,]+/).filter(Boolean); },
    lMove: el => { const L = cur().priorities[+el.dataset.i].ladder; move(L, +el.dataset.t, +el.dataset.d); },
    lTie: el => { const L = cur().priorities[+el.dataset.i].ladder, t = +el.dataset.t; L[t - 1] = L[t - 1].concat(L[t]); L.splice(t, 1); },
    lSplit: el => { const L = cur().priorities[+el.dataset.i].ladder, t = +el.dataset.t; L.splice(t, 1, ...L[t].map(x => [x])); },
    prioVar: el => { view.prioVariant = el.dataset.v; },
    prioItem: el => { view.prioItem = el.value; },
    teachPick: el => {
      const r = cur(); const p = teachPairs(r).find(p => !view.teach.some(t => t.r === r.id && t.key === p.key));
      if (!p) return; const chosenA = el.dataset.k === '0';
      const win = p.diffs.filter(z => (z.c > 0) === chosenA).map(z => z.m), lose = p.diffs.filter(z => (z.c > 0) !== chosenA).map(z => z.m);
      view.teach.push({ r: r.id, key: p.key, constraint: { win, lose } });
    },
    teachApply: () => { const r = cur(); const s = solveOrder(r, view.teach.filter(t => t.r === r.id).map(t => t.constraint)); if (!s.conflict) r.priorities = s.order.map(m => r.priorities.find(p => p.measure === m)); },
    teachReset: () => { view.teach = view.teach.filter(t => t.r !== cur().id); },
    prName: el => { S.profiles[+el.dataset.p].name = el.value; },
    peR: el => { S.profiles[+el.dataset.p].entries[+el.dataset.e].rendition = el.value; },
    peS: el => { const e = S.profiles[+el.dataset.p].entries[+el.dataset.e]; e.sourcing = el.value; if (el.value !== 'download' && !e.from) e.from = S.renditions.find(r => r.id !== e.rendition)?.id; if (el.value === 'download_then_derive' && e.graceDays == null) e.graceDays = 14; },
    peF: el => { S.profiles[+el.dataset.p].entries[+el.dataset.e].from = el.value; },
    peG: el => { S.profiles[+el.dataset.p].entries[+el.dataset.e].graceDays = +el.value; },
    peDel: el => { S.profiles[+el.dataset.p].entries.splice(+el.dataset.e, 1); },
    peAdd: el => { const p = S.profiles[+el.dataset.p]; const r = S.renditions.find(r => !p.entries.some(e => e.rendition === r.id)) || S.renditions[0]; p.entries.push({ rendition: r.id, sourcing: 'download' }); },
    prEd: el => { S.profiles[+el.dataset.p].editions = el.value; },
    defP: el => { S.defaults[el.dataset.t] = el.value; },
    addProf: el => { const t = el.dataset.t; S.profiles.push({ id: 'p' + Math.random().toString(36).slice(2, 7), name: 'New ' + t + ' profile', contentType: t, editions: 'any', entries: [{ rendition: S.renditions[0].id, sourcing: 'download' }] }); },
    ovDel: el => { delete S.workOverrides[el.dataset.k]; },
    ovSet: el => { const w = D.works.find(x => x.key === el.dataset.w); if (el.value === S.defaults[w.type]) delete S.workOverrides[w.key]; else S.workOverrides[w.key] = el.value; },
    sz: el => { const s = S.sizes[el.dataset.q]; s[el.dataset.k] = el.value === '' ? null : +el.value; },
    packOn: el => { S.knowledge[el.dataset.p] = el.checked; },
    trashKind: el => { S.knowledge.trashKinds[el.dataset.k] = el.checked; },
    claimOn: el => { const id = el.dataset.id; S.knowledge.disabled = el.checked ? S.knowledge.disabled.filter(x => x !== id) : S.knowledge.disabled.concat(id); },
    sev: el => { S.knowledge.severity[el.dataset.id] = el.value; },
    conf: el => { S.knowledge.confidence[el.dataset.id] = el.value; },
    claimFilter: el => { view.claimFilter = el.value; },
    claimKind: el => { view.claimKind = el.value; },
    tw: el => { S.trust.weakBelow = +el.value; },
    tk: el => { S.trust.priorStrength = +el.value; },
    tCal: el => { S.trust.calibrateTerms = el.checked; },
    tp: el => { S.trust.priors[el.dataset.k] = +el.value; },
    vf: el => { S.verification[el.dataset.k] = el.checked; },
    vs: el => { S.verification[el.dataset.k] = el.value; },
    tPat: el => { cur().timing.patience = el.dataset.k; },
    tNum: el => { cur().timing[el.dataset.k] = +el.value; },
    tryW: el => { view.work = el.value; view.item = null; },
    tryI: el => { view.item = el.value; },
    tryWork: el => { view.work = el.dataset.w; view.item = null; view.section = 'try'; },
    arrP: el => { view.arrProfile = el.value; },
    walk: el => { view.walk = +el.dataset.i; view.walkStep = 0; view.walkActive = true; },
    walkClose: () => { view.walkActive = false; },
    walkStep: el => {
      const w = WALKS[view.walk], i = +el.dataset.i;
      if (i === 0) { S = clone(DEFAULTS); BASE = clone(DEFAULTS); }
      w.steps[i].run({ S, view, E, D, clone, LADDERS });
      view.walkStep = i + 1;
    },
    copy: () => { const t = JSON.stringify(S, null, 2); (navigator.clipboard ? navigator.clipboard.writeText(t) : Promise.reject()).catch(() => { const r = document.createRange(); r.selectNode($('#stateJson')); getSelection().removeAllRanges(); getSelection().addRange(r); }); },
  };
  const RENDER = { impact: renderImpact, renditions: renderRenditions, priorities: renderPriorities, profiles: renderProfiles, qualities: renderQualities, knowledge: renderKnowledge, trust: renderTrust, timing: renderTiming, try: renderTry, walk: renderWalk, state: renderState };
  function renderDock() {
    const w = WALKS[view.walk];
    const dock = $('#dock');
    if (!view.walkActive || !w) { dock.hidden = true; return; }
    dock.hidden = false;
    dock.innerHTML = `<div class="row"><b>${esc(w.title)}</b><span class="small muted">${view.walkStep}/${w.steps.length}</span><button class="b" data-act="walkClose">close</button></div>
      <div class="row">${w.steps.map((s, i) => `<button class="b ${i < view.walkStep ? 'done' : ''} ${i === view.walkStep ? 'primary' : ''}" data-act="walkStep" data-i="${i}" ${i > view.walkStep ? 'disabled' : ''}>${i + 1}. ${esc(s.label)}</button>`).join('')}</div>
      ${view.walkStep >= w.steps.length ? `<div class="why good small">${w.done}</div>` : ''}`;
  }
  function render() {
    renderNav(); renderTop(); renderDock();
    const sx = window.scrollX, sy = window.scrollY;
    $('#main').innerHTML = RENDER[view.section]();
    window.scrollTo(sx, sy);
  }
  function handle(e) {
    const el = e.target.closest('[data-act]'); if (!el) return;
    const a = ACT[el.dataset.act]; if (!a) return;
    const isField = el.matches('input[type=text], input[type=number]');
    if (e.type === 'click' && (el.matches('input, select') && !el.matches('input[type=checkbox], input[type=radio]'))) return;
    if (e.type === 'change' && !el.matches('input, select')) return;
    if (e.type === 'input' && !isField) return;
    if (e.type === 'click' && el.matches('input[type=checkbox], input[type=radio]')) return;
    a(el);
    if (e.type === 'input' && el.dataset.act === 'claimFilter') { const pos = el.selectionStart; render(); const n = $('[data-act=claimFilter]'); if (n) { n.focus(); n.setSelectionRange(pos, pos); } return; }
    if (e.type === 'input') return;
    render();
  }
  document.addEventListener('click', handle);
  document.addEventListener('change', handle);
  document.addEventListener('input', handle);
  window.__proto = { get S() { return S; }, impact, decisions, derive, render, view };
  render();
})();
