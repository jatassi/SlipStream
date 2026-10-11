// PROTOTYPE · the one decision function the manual search sheet shows (#87 Q5, Q15). Judges each Release once per Target against every Want its Coverage touches. Throwaway.

export const KINDS = {
  fill: { rank: 0, word: 'Would fill', short: 'Fills', icon: 'Tick02', cls: 'v-fill' },
  soft: { rank: 1, word: 'Soft rejection', short: 'Soft', icon: 'Alert02', cls: 'v-soft' },
  meets: { rank: 2, word: 'Meets, not better', short: 'Meets', icon: 'Equal', cls: 'v-meets' },
  block: { rank: 3, word: 'Blocklisted', short: 'Blocklisted', icon: 'Unavailable', cls: 'v-hard' },
  hard: { rank: 4, word: 'Rejected', short: 'Rejected', icon: 'Cancel01', cls: 'v-hard' },
};

export const STATES = {
  failed: { rank: 0, word: 'Failed' },
  upgrade: { rank: 1, word: 'Upgrade' },
  missing: { rank: 2, word: 'Missing' },
  met: { rank: 3, word: 'Met' },
};

const QUALITY = ['Remux', 'BluRay', 'WEB-DL', 'WEBRip', 'HDTV'];
const CAMS = ['CAM', 'Telesync', 'TS'];

export const fmt = {
  size: (gb) => (gb < 1 ? `${Math.round(gb * 1000)} MB` : `${gb.toFixed(1)} GB`),
  age: (m) => (m < 60 ? `${m} min` : m < 60 * 48 ? `${Math.round(m / 60)} h` : `${Math.round(m / 1440)} d`),
  score: (n) => (n < 0 ? `−${Math.abs(n).toLocaleString('en-US')}` : n.toLocaleString('en-US')),
  signed: (n) => (n < 0 ? `−${Math.abs(n).toLocaleString('en-US')}` : `+${n.toLocaleString('en-US')}`),
};

export const itemById = (fx, id) => fx.items.find((i) => i.id === id);
export const targetById = (fx, id) => fx.targets.find((t) => t.id === id);
export const cellKey = (item, edition, target) => `${item}|${edition}|${target}`;
export const parseCell = (key) => { const [item, edition, target] = key.split('|'); return { item, edition, target, key }; };

// "E01–E06", "S01E01–S02E10", "Movie"
export function rangeLabel(fx, ids) {
  if (fx.work.kind === 'movie') return 'Movie';
  const its = ids.map((id) => itemById(fx, id)).sort((a, b) => a.season - b.season || a.ep - b.ep);
  const runs = [];
  for (const it of its) {
    const last = runs[runs.length - 1];
    if (last && last.season === it.season && last.to + 1 === it.ep) last.to = it.ep;
    else runs.push({ season: it.season, from: it.ep, to: it.ep });
  }
  const seasons = new Set(its.map((i) => i.season));
  const p = (n) => String(n).padStart(2, '0');
  return runs.map((r) => {
    const pre = seasons.size > 1 ? `S${p(r.season)}` : '';
    return r.from === r.to ? `${pre}E${p(r.from)}` : `${pre}E${p(r.from)}–E${p(r.to)}`;
  }).join(', ');
}

export function coverageLabel(fx, r) {
  if (fx.work.kind === 'movie') return 'Movie';
  const lvls = fx.levels.filter((l) => {
    const its = fx.items.filter((i) => i.level === l.id).map((i) => i.id);
    return its.every((id) => r.coverage.includes(id));
  });
  const covered = new Set(r.coverage);
  const wholeOnly = lvls.length && fx.items.filter((i) => covered.has(i.id)).every((i) => lvls.some((l) => l.id === i.level));
  if (wholeOnly && lvls.length === 1) return `${lvls[0].short} pack`;
  if (wholeOnly) return `${lvls[0].short}–${lvls[lvls.length - 1].short} packs`;
  if (r.coverage.length === 1) return itemById(fx, r.coverage[0]).short;
  const s = itemById(fx, r.coverage[0]);
  return `S${String(s.season).padStart(2, '0')}${rangeLabel(fx, r.coverage)}`;
}

export const cfScore = (fx, r, t) => r.cfs.reduce((n, name) => n + ((fx.cf[name] || {})[t.id] || 0), 0);
export const qualityName = (r) => `${r.source} ${r.res}`;
export const qualityRank = (r) => QUALITY.indexOf(r.source) + 1;

// A Release shaped for another Target is not a fault, so those rejections read neutral.
const SHAPE = ['Not 2160p', 'Not 1080p', 'Not 720p', 'Not HDR', 'Not SDR'];

// One verdict for one Release at one Target.
export function judge(fx, r, t) {
  const score = cfScore(fx, r, t);
  const base = { score, fillItems: [], staleItems: [] };
  const hard = [];
  if (r.lang !== 'English') hard.push([r.lang, `${r.lang} audio only, ${t.name} needs English`]);
  if (CAMS.includes(r.source)) hard.push([r.source === 'CAM' ? 'CAM' : 'Telesync', `${r.source} source is never allowed`]);
  if (r.res !== t.res) hard.push([`Not ${t.res}`, `${r.res}, and ${t.name} takes ${t.res} only`]);
  if (r.res === t.res && t.hdr === 'required' && !r.hdr) hard.push(['Not HDR', `SDR, and ${t.name} needs HDR`]);
  if (r.res === t.res && t.hdr === 'forbid' && r.hdr) hard.push(['Not SDR', `${r.hdr}, and ${t.name} needs SDR`]);
  if (!fx.editions.includes(r.edition)) hard.push([r.edition, `Parsed as ${r.edition}, which this Work doesn’t want`]);
  if (hard.length) return { ...base, kind: 'hard', shape: hard.every((h) => SHAPE.includes(h[0])), short: hard[0][0], full: hard.map((h) => h[1]).join('. '), reasons: hard };
  if (r.blocklisted) return { ...base, kind: 'block', short: 'Blocklisted', full: `Blocklisted ${r.blocklisted.on}. ${r.blocklisted.cause}` };

  for (const id of r.coverage) {
    const c = fx.cells[cellKey(id, r.edition, t.id)];
    if (!c) continue;
    const better = c.state === 'missing' || c.state === 'failed' || (c.state === 'upgrade' && score > c.score);
    (better ? base.fillItems : base.staleItems).push(id);
  }
  const n = r.coverage.length;
  if (!base.fillItems.length) {
    const c = fx.cells[cellKey(r.coverage[0], r.edition, t.id)];
    const full = r.current ? 'This is the file the cell already holds'
      : n > 1 ? `Not better than the files for ${rangeLabel(fx, base.staleItems)}`
        : c && c.state === 'met' ? `The cell is Met with score ${fmt.score(c.score)}, cutoff reached`
          : `Score ${fmt.score(score)} doesn’t beat the current ${fmt.score(c.score)}`;
    return { ...base, kind: 'meets', short: 'Meets', full };
  }

  const per = r.size / n;
  const soft = [];
  if (score < t.minScore) soft.push(['Low score', `Score ${fmt.score(score)}, below the ${fmt.score(t.minScore)} minimum`]);
  if (per > t.size[1]) soft.push(['Too big', `${fmt.size(per)}${n > 1 ? ' an Item' : ''}, over the ${t.size[1]} GB maximum`]);
  if (per < t.size[0]) soft.push(['Too small', `${fmt.size(per)}${n > 1 ? ' an Item' : ''}, under the ${t.size[0]} GB minimum`]);
  if (r.titleMatch < 85) soft.push([`Title ${r.titleMatch}%`, `Title matches ${r.titleMatch}%, under the 85% needed`]);
  if (soft.length) return { ...base, kind: 'soft', short: soft[0][0], full: soft.map((s) => s[1]).join('. '), reasons: soft };
  if (t.hold && r.age < t.hold) {
    const left = t.hold - r.age;
    return { ...base, kind: 'soft', hold: left, short: `Held ${left} min`, full: `Held ${left} min more for a better Release. A manual grab skips the Hold`, reasons: [['Hold', 'Hold']] };
  }
  const c0 = fx.cells[cellKey(r.coverage[0], r.edition, t.id)];
  const full = n > 1 && base.staleItems.length ? `Fills ${rangeLabel(fx, base.fillItems)}. ${rangeLabel(fx, base.staleItems)} keep their files, which are as good`
    : n > 1 ? `Fills ${n} Items: ${rangeLabel(fx, base.fillItems)}`
    : c0.state === 'upgrade' ? `Upgrades the current file, ${fmt.score(c0.score)} → ${fmt.score(score)}`
      : c0.state === 'failed' ? 'Replaces the failed grab'
        : 'Fills a missing cell';
  return { ...base, kind: 'fill', short: 'Fills', full };
}

// Grabbing fills every Target the release would fill. A manual grab skips the Hold, so held verdicts count.
export const fills = (v) => v.kind === 'fill' || (v.kind === 'soft' && v.hold != null);

// ───────────────────────────── scope

export function normScope(fx, sc) {
  const s = { kind: sc.kind || 'work', level: sc.level || fx.defaultLevel, item: sc.item || fx.defaultItem || 'movie', cell: sc.cell || fx.defaultCell };
  if (fx.work.kind === 'movie' && (s.kind === 'level' || s.kind === 'item')) s.kind = 'work';
  if (s.kind === 'cell') s.item = parseCell(s.cell).item;
  if (s.kind === 'item' || s.kind === 'cell') s.level = (itemById(fx, s.item) || {}).level || s.level;
  return s;
}

export function scopeOptions(fx, sc) {
  const c = parseCell(sc.cell);
  const t = targetById(fx, c.target);
  const cellLabel = fx.work.kind === 'movie' ? `${c.edition} · ${t.name}` : `${itemById(fx, c.item).short} · ${fx.editions.length > 1 ? `${c.edition} · ` : ''}${t.name}`;
  if (fx.work.kind === 'movie') return [{ kind: 'work', label: fx.work.title }, { kind: 'cell', label: cellLabel }];
  const lvl = fx.levels.find((l) => l.id === sc.level);
  return [
    { kind: 'work', label: fx.work.title },
    { kind: 'level', label: lvl.label },
    { kind: 'item', label: itemById(fx, sc.item).short },
    { kind: 'cell', label: cellLabel },
  ];
}

export function scopeItems(fx, sc) {
  if (sc.kind === 'work') return fx.items.map((i) => i.id);
  if (sc.kind === 'level') return fx.items.filter((i) => i.level === sc.level).map((i) => i.id);
  return [sc.item];
}
export const scopeTargets = (fx, sc) => (sc.kind === 'cell' ? [targetById(fx, parseCell(sc.cell).target)] : fx.targets);
export const scopeEditions = (fx, sc) => (sc.kind === 'cell' ? [parseCell(sc.cell).edition] : fx.editions);

export function scopeCells(fx, sc) {
  const out = [];
  for (const id of scopeItems(fx, sc)) for (const ed of scopeEditions(fx, sc)) for (const t of scopeTargets(fx, sc)) {
    const key = cellKey(id, ed, t.id);
    if (fx.cells[key]) out.push({ key, item: itemById(fx, id), edition: ed, target: t, ...fx.cells[key] });
  }
  return out;
}

export function scopeTitle(fx, sc) {
  const o = scopeOptions(fx, sc).find((x) => x.kind === sc.kind);
  return o ? o.label : fx.work.title;
}

// ───────────────────────────── results

export function evaluate(fx, sc) {
  const items = new Set(scopeItems(fx, sc));
  const eds = scopeEditions(fx, sc);
  const tids = scopeTargets(fx, sc).map((t) => t.id);
  return fx.releases.filter((r) => r.coverage.some((id) => items.has(id))).map((r) => {
    const v = Object.fromEntries(fx.targets.map((t) => [t.id, judge(fx, r, t)]));
    const fillTargets = fx.targets.filter((t) => fills(v[t.id]));
    const wantedTargets = fillTargets.filter((t) => tids.includes(t.id) && eds.includes(r.edition) && v[t.id].fillItems.some((id) => items.has(id)));
    const allHard = fx.targets.every((t) => v[t.id].kind === 'hard');
    const best = Math.min(...fx.targets.map((t) => KINDS[v[t.id].kind].rank));
    const top = Math.max(...fx.targets.map((t) => v[t.id].score));
    return { r, v, fillTargets, wanted: wantedTargets.length > 0, wantedTargets, allHard, best, top, coverage: coverageLabel(fx, r), indexer: null };
  });
}

export const cellVerdict = (fx, row, key) => {
  const c = parseCell(key);
  const v = row.v[c.target];
  if (row.r.edition !== c.edition && v.kind !== 'hard') return { ...v, kind: 'hard', shape: true, short: 'Edition', full: `Holds ${row.r.edition}, not ${c.edition}` };
  if (!row.r.coverage.includes(c.item) && v.kind !== 'hard') return { ...v, kind: 'hard', shape: true, short: 'Coverage', full: `Doesn’t cover ${itemById(fx, c.item).short}` };
  if (v.kind === 'fill' && !v.fillItems.includes(c.item)) return { ...v, kind: 'meets', short: 'Meets', full: 'Doesn’t beat this cell’s file' };
  return v;
};

const rankCmp = (a, b) => b.fillTargets.length - a.fillTargets.length || a.best - b.best || b.top - a.top || qualityRank(a.r) - qualityRank(b.r) || b.r.seeders - a.r.seeders;

export const SORTS = {
  best: { label: 'Best first', cmp: rankCmp },
  size: { label: 'Size', cmp: (a, b) => b.r.size - a.r.size },
  seeders: { label: 'Seeders', cmp: (a, b) => b.r.seeders - a.r.seeders },
  age: { label: 'Age', cmp: (a, b) => a.r.age - b.r.age },
  indexer: { label: 'Indexer', cmp: (a, b) => a.r.indexer.localeCompare(b.r.indexer) || rankCmp(a, b) },
  score: { label: 'Score', cmp: (a, b) => b.top - a.top },
};

export function rankForTarget(rows, tid) {
  return [...rows].sort((a, b) => KINDS[a.v[tid].kind].rank - KINDS[b.v[tid].kind].rank || b.v[tid].score - a.v[tid].score || qualityRank(a.r) - qualityRank(b.r) || b.r.seeders - a.r.seeders || (b.r.freeleech ? 1 : 0) - (a.r.freeleech ? 1 : 0) || a.r.age - b.r.age);
}

export function rankForCell(fx, rows, key) {
  return [...rows].map((row) => ({ row, cv: cellVerdict(fx, row, key) })).sort((a, b) => KINDS[a.cv.kind].rank - KINDS[b.cv.kind].rank || b.cv.score - a.cv.score || qualityRank(a.row.r) - qualityRank(b.row.r) || b.row.r.seeders - a.row.r.seeders);
}

export function grabLabel(fx, row) {
  const ts = row.fillTargets;
  if (!ts.length) return null;
  const items = new Set(ts.flatMap((t) => row.v[t.id].fillItems));
  const n = items.size;
  return `Grab · fills ${ts.map((t) => t.name).join(', ')}${n > 1 ? ` for ${n} Items` : ''}`;
}

export function cellStateLine(fx, c) {
  const t = c.target;
  if (c.state === 'met') return 'Met, cutoff reached';
  if (c.state === 'upgrade') return `Met at ${fmt.score(c.score)}, upgrading to ${fmt.score(t.upgradeUntil)}`;
  if (c.state === 'failed') return 'Failed';
  return 'Missing';
}
