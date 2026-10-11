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

export const rankCmp = (a, b) => b.fillTargets.length - a.fillTargets.length || a.best - b.best || b.top - a.top || qualityRank(a.r) - qualityRank(b.r) || b.r.seeders - a.r.seeders;

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

// ───────────────────────────── round 2: plain words, needs and the plan (D, E, F). Same verdicts as above, said as outcomes.

const NEW = new Set(['missing', 'failed']);
const p2 = (n) => String(n).padStart(2, '0');
const and = (xs) => (xs.length < 2 ? xs.join('') : `${xs.slice(0, -1).join(', ')} and ${xs[xs.length - 1]}`);
const plural = (n, one, many) => (n === 1 ? one : many);

export const openCells = (fx) => Object.keys(fx.cells).filter((k) => fx.cells[k].state !== 'met');
export const needKeys = (fx, sc) => scopeCells(fx, sc).filter((c) => c.state !== 'met').map((c) => c.key);
export const metKeys = (fx, sc) => scopeCells(fx, sc).filter((c) => c.state === 'met').map((c) => c.key);

// "720p low bitrate", or "2160p HDR (Extended)" when the Work has more than one Edition.
export const targetNoun = (fx, tid, ed) => `${targetById(fx, tid).name}${fx.editions.length > 1 ? ` (${ed})` : ''}`;

// "E05, E06, E08–E10", "all 10 episodes", "all of Season 2", "S02E07–E10"
export function itemsPhrase(fx, ids, sc) {
  const set = new Set(ids);
  if (fx.work.kind === 'movie') return '';
  const lvl = fx.levels.find((l) => { const its = fx.items.filter((i) => i.level === l.id); return its.length === set.size && its.every((i) => set.has(i.id)); });
  if (lvl) return sc.kind === 'level' ? `all ${set.size} episodes` : `all of ${lvl.label}`;
  const its = [...set].map((id) => itemById(fx, id)).sort((a, b) => a.season - b.season || a.ep - b.ep);
  const pre = sc.kind === 'work' || new Set(its.map((i) => i.season)).size > 1;
  const runs = [];
  for (const it of its) { const l = runs[runs.length - 1]; if (l && l.season === it.season && l.to + 1 === it.ep) l.to = it.ep; else runs.push({ season: it.season, from: it.ep, to: it.ep }); }
  const e = (r, n, first) => `${pre && first ? `S${p2(r.season)}` : ''}E${p2(n)}`;
  return runs.map((r) => (r.from === r.to ? e(r, r.from, true) : r.to === r.from + 1 ? `${e(r, r.from, true)}, ${e(r, r.to, true)}` : `${e(r, r.from, true)}–${e(r, r.to, false)}`)).join(', ');
}

function byTarget(fx, keys) {
  const m = new Map();
  for (const k of keys) {
    const c = parseCell(k);
    const id = `${c.target}|${c.edition}`;
    if (!m.has(id)) m.set(id, { tid: c.target, ed: c.edition, keys: [] });
    m.get(id).keys.push(k);
  }
  const tix = (g) => fx.targets.findIndex((t) => t.id === g.tid) * 10 + fx.editions.indexOf(g.ed);
  return [...m.values()].sort((a, b) => tix(a) - tix(b));
}

// "720p low bitrate for all 10 episodes", "1080p for E08", "2160p HDR (Standard)"
export function keysPhrase(fx, keys, sc, mix = false) {
  return and(byTarget(fx, keys).map((g) => {
    const its = itemsPhrase(fx, g.keys.map((k) => parseCell(k).item), sc);
    const n = g.keys.filter((k) => NEW.has(fx.cells[k].state)).length;
    const m = mix && n && n < g.keys.length ? ` (${n} new, ${g.keys.length - n} ${plural(g.keys.length - n, 'upgrade', 'upgrades')})` : '';
    return `${targetNoun(fx, g.tid, g.ed)}${its ? ` for ${its}` : ''}${m}`;
  }));
}

// The open cells a Release would fill, cleanly (a Hold counts, since a manual grab skips it) or only past a soft rejection.
export function gainsOf(fx, row, keys) {
  const clean = [], soft = [];
  for (const key of keys) {
    const cv = cellVerdict(fx, row, key);
    if (fills(cv)) clean.push(key);
    else if (cv.kind === 'soft' && cv.fillItems.includes(parseCell(key).item)) soft.push(key);
  }
  return { clean, soft };
}

function gainSentence(fx, keys, sc) {
  const groups = byTarget(fx, keys);
  const fresh = groups.filter((g) => g.keys.some((k) => NEW.has(fx.cells[k].state))).flatMap((g) => g.keys);
  const better = groups.filter((g) => !g.keys.some((k) => NEW.has(fx.cells[k].state))).flatMap((g) => g.keys);
  if (fresh.length) return `Gets you ${keysPhrase(fx, fresh, sc, true)}${better.length ? ` and a better ${keysPhrase(fx, better, sc)}` : ''}`;
  return `A better ${keysPhrase(fx, better, sc)}${fx.work.kind === 'movie' ? ' than the one you have' : ''}`;
}

function cantWhy(fx, r) {
  if (r.lang !== 'English') return { code: 'lang', why: `${r.lang} audio`, lang: r.lang };
  if (CAMS.includes(r.source)) return { code: 'cam', why: 'camera recording' };
  if (!fx.editions.includes(r.edition)) return { code: 'edition', why: `labelled ${r.edition}, which you don’t collect` };
  if (!fx.targets.some((t) => t.res === r.res)) return { code: 'res', why: `you don’t keep ${r.res}` };
  return { code: 'hdr', why: r.hdr ? `${r.res} with HDR, which you keep without` : `${r.res} without HDR` };
}

function softWhy(fx, r, v) {
  const w = v.reasons[0][0];
  const n = r.coverage.length;
  const per = `${fmt.size(r.size / n)}${n > 1 ? ' an episode' : ''}`;
  if (w === 'Too big') return { code: 'big', why: `it’s larger than you allow (${per})` };
  if (w === 'Too small') return { code: 'small', why: `it’s smaller than you allow (${per})` };
  if (w === 'Low score') return { code: 'score', why: 'it’s below the quality you set' };
  return { code: 'title', why: `its name doesn’t match ${fx.work.title} well` };
}

// One sentence of what grabbing a Release gets you, judged against the open cells in `focus`.
// kind: new | better (SlipStream would grab) · soft | blocked | cant | have | elsewhere (it wouldn't).
export function outcome(fx, row, focus, sc) {
  const { r, v } = row;
  const g = gainsOf(fx, row, openCells(fx));
  const inF = new Set(focus);
  const clean = g.clean.filter((k) => inF.has(k));
  const also = g.clean.filter((k) => !inF.has(k));
  const alsoText = also.length ? `Also gets you ${keysPhrase(fx, also, sc)}` : '';
  if (clean.length) {
    const held = fx.targets.map((t) => v[t.id]).find((x) => x.hold != null && fills(x));
    return { kind: clean.some((k) => NEW.has(fx.cells[k].state)) ? 'new' : 'better', text: gainSentence(fx, clean, sc), also: alsoText, note: held ? `SlipStream would wait ${held.hold} more min for a better one. Grabbing now skips the wait.` : '', keys: clean, alsoKeys: also };
  }
  const base = { keys: [], alsoKeys: also, also: '', note: '' };
  if (r.blocklisted && !row.allHard) return { ...base, kind: 'blocked', code: 'blocked', text: `Blocked on ${r.blocklisted.on}: ${r.blocklisted.plain}` };
  if (row.allHard) { const c = cantWhy(fx, r); return { ...base, kind: 'cant', ...c, text: `Can’t use: ${c.why}` }; }
  const soft = g.soft.filter((k) => inF.has(k));
  if (soft.length) {
    const t = fx.targets.find((x) => soft.some((k) => parseCell(k).target === x.id));
    const c = softWhy(fx, r, v[t.id]);
    return { ...base, kind: 'soft', ...c, keys: soft, text: `Gets you ${keysPhrase(fx, soft, sc)}, but ${c.why}` };
  }
  if (also.length) return { ...base, kind: 'elsewhere', code: 'elsewhere', text: `Gets you ${keysPhrase(fx, also, sc)} instead` };
  return { ...base, kind: 'have', code: 'have', text: r.current ? 'This is the copy you already have' : 'Nothing you don’t already have' };
}
export const wouldGrab = (o) => o.kind === 'new' || o.kind === 'better';

// What SlipStream wouldn't grab, grouped by reason, each group said as a sentence.
const BUCKET = { res: 'off', hdr: 'off', edition: 'off', big: 'limits', small: 'limits', score: 'limits', title: 'limits' };
const HIDDEN = {
  have: (n) => `${n} ${plural(n, 'is', 'are')} no better than what you have`,
  off: (n) => `${n} ${plural(n, 'isn’t', 'aren’t')} a version you keep`,
  lang: (n, rows) => (new Set(rows.map((x) => x.row.r.lang)).size === 1 ? `${n} ${plural(n, 'has', 'have')} ${rows[0].row.r.lang} audio` : `${n} ${plural(n, 'is', 'are')} in another language`),
  cam: (n) => `${n} ${plural(n, 'is a camera recording', 'are camera recordings')}`,
  limits: (n) => `${n} ${plural(n, 'is', 'are')} outside the limits you set`,
  blocked: (n) => `${n} ${plural(n, 'is', 'are')} blocked`,
  elsewhere: (n) => `${n} ${plural(n, 'gets', 'get')} you something else`,
};
export function hiddenGroups(fx, items) {
  const m = new Map();
  for (const x of items) { const b = BUCKET[x.out.code] || x.out.code; if (!m.has(b)) m.set(b, []); m.get(b).push(x); }
  return [...m.entries()].map(([code, rows]) => ({ code, rows, sentence: HIDDEN[code](rows.length, rows, fx) })).sort((a, b) => b.rows.length - a.rows.length);
}

// A Release as people talk about it: "Season 2 pack · 720p web", "4K Remux · Dolby Vision · Atmos".
const RES_W = { '2160p': '4K', '1080p': '1080p', '720p': '720p' };
const SRC_W = { Remux: 'Remux', BluRay: 'Blu-ray', 'WEB-DL': 'web', WEBRip: 'web rip', HDTV: 'TV recording' };
const HDR_W = { 'DV HDR10': 'Dolby Vision', HDR10: 'HDR', 'HDR10+': 'HDR10+' };
export function describe(fx, r) {
  const parts = [];
  if (fx.work.kind !== 'movie') {
    const lv = fx.levels.filter((l) => fx.items.filter((i) => i.level === l.id).every((i) => r.coverage.includes(i.id)));
    const its = r.coverage.map((id) => itemById(fx, id)).sort((a, b) => a.season - b.season || a.ep - b.ep);
    if (lv.length === 1 && its.length === fx.items.filter((i) => i.level === lv[0].id).length) parts.push(`${lv[0].label} pack`);
    else if (lv.length > 1) parts.push(`Seasons ${lv[0].label.replace('Season ', '')}–${lv[lv.length - 1].label.replace('Season ', '')} pack`);
    else if (its.length === 1) parts.push(`Episode ${its[0].ep}`);
    else parts.push(`Episodes ${its[0].ep}–${its[its.length - 1].ep}`);
  }
  if (r.edition !== 'Standard') parts.push(r.edition);
  parts.push(CAMS.includes(r.source) ? 'Camera recording' : `${RES_W[r.res]} ${SRC_W[r.source]}`);
  if (r.hdr) parts.push(HDR_W[r.hdr]);
  else if (r.res === '2160p') parts.push('no HDR');
  if (/Atmos/.test(r.audio)) parts.push('Atmos');
  if (r.lang !== 'English') parts.push(`${r.lang} audio`);
  const health = r.seeders === 0 ? { word: 'No seeders', low: true } : r.seeders < 10 ? { word: `Few seeders (${r.seeders})`, low: true } : { word: `${r.seeders} seeders`, low: false };
  return { parts, lead: parts.join(' · '), size: r.size >= 10 ? `${Math.round(r.size)} GB` : fmt.size(r.size), health };
}

// "1080p web", from a file name, for "you have a 1080p web copy".
export function describeFile(name) {
  const res = (name.match(/2160p|1080p|720p/) || [''])[0];
  const src = /REMUX/i.test(name) ? 'Remux' : /BluRay/i.test(name) ? 'Blu-ray' : /WEBRip/i.test(name) ? 'web rip' : /WEB/i.test(name) ? 'web' : '';
  return `${RES_W[res] || res} ${src}`.trim();
}

// Needs: the open cells in a scope, merged by state and Target into ranges, worst first.
export const NEED_WORD = { failed: 'Failed', missing: 'Missing', upgrade: 'Could be better', met: 'Already fine' };
const NEED_ORDER = ['failed', 'missing', 'upgrade', 'met'];
export function needGroups(fx, keys, sc) {
  const m = new Map();
  for (const k of keys) {
    const c = parseCell(k);
    const st = fx.cells[k].state;
    const id = `${st}|${c.target}|${c.edition}`;
    if (!m.has(id)) m.set(id, { id, state: st, tid: c.target, ed: c.edition, keys: [] });
    m.get(id).keys.push(k);
  }
  const tix = (g) => fx.targets.findIndex((t) => t.id === g.tid) * 10 + fx.editions.indexOf(g.ed);
  return [...m.values()].sort((a, b) => NEED_ORDER.indexOf(a.state) - NEED_ORDER.indexOf(b.state) || tix(a) - tix(b)).map((g) => ({ ...g, word: NEED_WORD[g.state], label: keysPhrase(fx, g.keys, sc) }));
}

// The plan (D): greedy. Take the release covering the most unfilled needs; on a tie, the one that fills more open cells
// anywhere (so a pack wins when it fills at least as much), then the better-ranked. Forced picks go first, excluded ones never.
export function buildPlan(fx, sc, rows, forced = [], excluded = new Set()) {
  const need = needKeys(fx, sc);
  const all = openCells(fx);
  const info = [...rows].sort(rankCmp).map((row, pos) => {
    const g = gainsOf(fx, row, all);
    return { row, pos, focus: g.clean.filter((k) => need.includes(k)), total: g.clean.length };
  }).filter((x) => x.focus.length);
  const left = new Set(need);
  const steps = [];
  const take = (x) => {
    const covers = x.focus.filter((k) => left.has(k));
    if (!covers.length) return;
    covers.forEach((k) => left.delete(k));
    steps.push({ row: x.row, covers });
  };
  for (const id of forced) { const x = info.find((y) => y.row.r.id === id); if (x) take(x); }
  for (;;) {
    let best = null, bn = 0;
    for (const x of info) {
      if (excluded.has(x.row.r.id) || steps.some((s) => s.row === x.row)) continue;
      const n = x.focus.filter((k) => left.has(k)).length;
      if (n && (!best || n > bn || (n === bn && (x.total > best.total || (x.total === best.total && x.pos < best.pos))))) { best = x; bn = n; }
    }
    if (!best) break;
    take(best);
  }
  const picked = new Set(steps.map((s) => s.row.r.id));
  for (const s of steps) {
    const ov = (x) => x.focus.filter((k) => s.covers.includes(k)).length;
    s.alts = info.filter((x) => !picked.has(x.row.r.id) && ov(x)).sort((a, b) => ov(b) - ov(a) || a.pos - b.pos).map((x) => x.row);
  }
  return { steps, missing: [...left], need };
}

export function planHeadline(fx, plan, sc) {
  const n = plan.steps.length;
  if (!plan.need.length) return 'Nothing here is missing';
  if (!n) return 'Nothing found yet that SlipStream would grab';
  const grabs = `${n} grab${n > 1 ? 's get' : ' gets'} you`;
  if (!plan.missing.length) return `${grabs} everything`;
  const gs = byTarget(fx, plan.missing);
  return `${grabs} everything except ${gs.length > 2 ? `${gs.length} things` : keysPhrase(fx, plan.missing, sc)}`;
}

// Equal coverage reads best in episode order; the same episode falls back to rank.
export function choiceCmp(fx, rank) {
  const first = (r) => Math.min(...r.coverage.map((id) => { const it = itemById(fx, id); return it.season ? it.season * 100 + it.ep : 0; }));
  return (a, b) => b.out.keys.length - a.out.keys.length || (fx.work.kind === 'movie' ? 0 : first(a.row.r) - first(b.row.r)) || rank.get(a.row.r.id) - rank.get(b.row.r.id);
}
