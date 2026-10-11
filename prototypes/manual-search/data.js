// PROTOTYPE fixture for the manual search sheet (#107). Releases, Indexers, scores and states are invented. Throwaway.

const ART = '../ui-direction/';
const art = (kind, id) => ({ poster: `${ART}art/${kind}/${id}_poster.jpg`, backdrop: `${ART}art/${kind}/${id}_backdrop.jpg` });

export const now = new Date('2026-10-11T21:40:00');
export const admin = { name: 'Jackson', initials: 'JA' };

export const clients = [
  { id: 'bi-ams', name: 'Built-in on seedbox-ams', node: 'seedbox-ams' },
  { id: 'qb-forge', name: 'qBittorrent on forge', node: 'forge' },
  { id: 'bi-atlas', name: 'Built-in on atlas', node: 'atlas' },
];

// Indexer answers arrive after `delay` ms. Tracker routing names the default Download Client per Tracker.
export const indexers = [
  { id: 'aurora', name: 'Aurora', tracker: 'Aurora', kind: 'private', delay: 650, outcome: 'ok', client: 'bi-ams' },
  { id: 'lumen', name: 'Lumen', tracker: 'Lumen', kind: 'public', delay: 1050, outcome: 'ok', client: 'qb-forge' },
  { id: 'nimbus', name: 'Nimbus', tracker: 'Nimbus', kind: 'private', delay: 1500, outcome: 'ok', client: 'bi-ams' },
  { id: 'halcyon', name: 'Halcyon', tracker: 'Halcyon', kind: 'private', delay: 1850, outcome: 'error', msg: 'Rate limited, retry 21:52' },
  { id: 'orbit', name: 'Orbit', tracker: 'Orbit', kind: 'private', delay: 2350, outcome: 'ok', client: 'qb-forge' },
  { id: 'meridian', name: 'Meridian', tracker: 'Meridian', kind: 'public', delay: 2900, outcome: 'challenge', msg: 'Challenge solver unreachable' },
];

// ───────────────────────────── The Batman: one Item, two Editions, Profile Cinema
const batmanCF = {
  'Remux Tier 01': { uhd: 1950, hd: 1950 },
  'UHD Bluray Tier 01': { uhd: 1800 },
  'HD Bluray Tier 01': { hd: 1800 },
  'WEB Tier 01': { uhd: 1700, hd: 1700 },
  'WEB Tier 02': { uhd: 1650, hd: 1650 },
  'DV HDR10': { uhd: 1500 },
  'HDR10+': { uhd: 600 },
  'HDR10': { uhd: 500 },
  'TrueHD Atmos': { uhd: 300, hd: 300 },
  'DTS-HD MA': { uhd: 250, hd: 250 },
  'DD+ Atmos': { uhd: 200, hd: 200 },
  'MA': { uhd: 50, hd: 50 },
  'Repack': { uhd: 5, hd: 5 },
  'x265 (HD)': { hd: -1000 },
  'LQ': { uhd: -10000, hd: -10000 },
};

const B = (id, name, o) => ({ id, name, lang: 'English', titleMatch: 100, edition: 'Standard', coverage: ['movie'], seeders: 0, leechers: 0, ...o });

const batman = {
  id: 'batman',
  label: 'The Batman',
  work: { id: 'the-batman', title: 'The Batman', year: 2022, kind: 'movie', meta: ['2022', '2h 56m', 'PG-13', 'Crime · Mystery'], art: art('movie', 414906), overview: 'Batman ventures into Gotham City’s underworld when a sadistic killer leaves behind a trail of cryptic clues.' },
  profile: { name: 'Cinema' },
  targets: [
    { id: 'uhd', name: '2160p HDR', res: '2160p', hdr: 'required', sourcing: 'Downloaded', minScore: 0, cutoff: 3000, upgradeUntil: 4000, size: [15, 80], hold: 0 },
    { id: 'hd', name: '1080p SDR', res: '1080p', hdr: 'forbid', sourcing: 'Downloaded, then derived from 2160p HDR', minScore: 0, cutoff: 1800, upgradeUntil: 2400, size: [4, 40], hold: 60 },
  ],
  editions: ['Standard', 'Extended'],
  levels: [],
  items: [{ id: 'movie', label: 'The Batman', short: 'Movie' }],
  cells: {
    'movie|Standard|uhd': { state: 'missing' },
    'movie|Standard|hd': { state: 'upgrade', score: 1900, file: 'The.Batman.2022.1080p.MA.WEB-DL.DDP5.1.Atmos.H.264-CMRG.mkv' },
    'movie|Extended|uhd': { state: 'missing' },
    'movie|Extended|hd': { state: 'missing' },
  },
  defaultCell: 'movie|Standard|uhd',
  cf: batmanCF,
  releases: [
    B('r01', 'The.Batman.2022.2160p.UHD.BluRay.REMUX.DV.HDR10.HEVC.TrueHD.7.1.Atmos-FraMeSToR', { indexer: 'aurora', size: 74.2, seeders: 212, leechers: 9, age: 60 * 24 * 410, res: '2160p', source: 'Remux', codec: 'HEVC', hdr: 'DV HDR10', audio: 'TrueHD Atmos 7.1', group: 'FraMeSToR', cfs: ['Remux Tier 01', 'DV HDR10', 'TrueHD Atmos'] }),
    B('r02', 'The.Batman.2022.2160p.MA.WEB-DL.DDP5.1.Atmos.DV.HDR10.H.265-FLUX', { indexer: 'nimbus', size: 22.6, seeders: 388, leechers: 14, age: 60 * 24 * 560, res: '2160p', source: 'WEB-DL', codec: 'HEVC', hdr: 'DV HDR10', audio: 'DD+ Atmos 5.1', group: 'FLUX', freeleech: true, cfs: ['WEB Tier 01', 'DV HDR10', 'DD+ Atmos', 'MA'] }),
    B('r03', 'The.Batman.2022.1080p.BluRay.DTS-HD.MA.7.1.x264-DON', { indexer: 'aurora', size: 19.4, seeders: 141, leechers: 3, age: 60 * 24 * 402, res: '1080p', source: 'BluRay', codec: 'AVC', hdr: null, audio: 'DTS-HD MA 7.1', group: 'DON', cfs: ['HD Bluray Tier 01', 'DTS-HD MA'] }),
    B('r04', 'The.Batman.2022.1080p.MA.WEB-DL.DDP5.1.Atmos.H.264-CMRG', { indexer: 'nimbus', size: 9.1, seeders: 96, leechers: 1, age: 60 * 24 * 561, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ Atmos 5.1', group: 'CMRG', cfs: ['WEB Tier 02', 'DD+ Atmos', 'MA'], current: true }),
    B('r05', 'The.Batman.2022.2160p.UHD.BluRay.x265.10bit.HDR10.TrueHD.7.1.Atmos-TERMiNAL', { indexer: 'lumen', size: 31.8, seeders: 57, leechers: 6, age: 60 * 24 * 398, res: '2160p', source: 'BluRay', codec: 'HEVC', hdr: 'HDR10', audio: 'TrueHD Atmos 7.1', group: 'TERMiNAL', cfs: ['HDR10', 'TrueHD Atmos'] }),
    B('r06', 'The.Batman.2022.REPACK.1080p.BluRay.DTS-HD.MA.7.1.x264-DON', { indexer: 'orbit', size: 19.6, seeders: 4, leechers: 22, age: 20, res: '1080p', source: 'BluRay', codec: 'AVC', hdr: null, audio: 'DTS-HD MA 7.1', group: 'DON', cfs: ['HD Bluray Tier 01', 'DTS-HD MA', 'Repack'] }),
    B('r07', 'The.Batman.2022.2160p.UHD.BluRay.x265.10bit.HDR.DTS-HD.MA.7.1-SWTYBLZ', { indexer: 'aurora', size: 28.3, seeders: 18, leechers: 0, age: 60 * 24 * 395, res: '2160p', source: 'BluRay', codec: 'HEVC', hdr: 'HDR10', audio: 'DTS-HD MA 7.1', group: 'SWTYBLZ', cfs: ['HDR10', 'DTS-HD MA'], blocklisted: { on: 'Oct 6', cause: 'Unregistered on Aurora, twice 30 min apart' } }),
    B('r08', 'The.Batman.2022.Open.Matte.2160p.WEB-DL.DDP5.1.DV.HDR10.H.265-HONE', { indexer: 'orbit', size: 18.9, seeders: 33, leechers: 2, age: 60 * 24 * 120, res: '2160p', source: 'WEB-DL', codec: 'HEVC', hdr: 'DV HDR10', audio: 'DD+ 5.1', group: 'HONE', edition: 'Open Matte', misparsed: true, cfs: ['WEB Tier 02', 'DV HDR10'] }),
    B('r09', 'The.Batman.2022.iTALiAN.HDCAM.x264-iLLuSiON', { indexer: 'lumen', size: 1.4, seeders: 6, leechers: 1, age: 60 * 24 * 940, res: '720p', source: 'CAM', codec: 'AVC', hdr: null, audio: 'AAC 2.0', group: 'iLLuSiON', lang: 'Italian', cfs: ['LQ'] }),
    B('r10', 'The.Batman.2022.1080p.TELESYNC.x264.AAC-NOGRP', { indexer: 'lumen', size: 2.2, seeders: 11, leechers: 0, age: 60 * 24 * 945, res: '1080p', source: 'Telesync', codec: 'AVC', hdr: null, audio: 'AAC 2.0', group: 'NOGRP', cfs: ['LQ'] }),
    B('r11', 'The.Batman.2022.EXTENDED.2160p.WEB-DL.DDP5.1.HDR10.H.265-EDITH', { indexer: 'nimbus', size: 21.7, seeders: 74, leechers: 5, age: 60 * 24 * 31, res: '2160p', source: 'WEB-DL', codec: 'HEVC', hdr: 'HDR10', audio: 'DD+ 5.1', group: 'EDITH', edition: 'Extended', cfs: ['WEB Tier 02', 'HDR10'] }),
    B('r12', 'The.Batman.2022.Extended.Cut.1080p.WEB-DL.DDP5.1.H.264-EDITH', { indexer: 'nimbus', size: 8.8, seeders: 61, leechers: 2, age: 60 * 24 * 31, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'EDITH', edition: 'Extended', cfs: ['WEB Tier 02'] }),
    B('r13', 'The.Batman.2022.2160p.UHD.BluRay.REMUX.HDR10.HEVC.DTS-HD.MA.7.1-ZQ', { indexer: 'orbit', size: 92.4, seeders: 28, leechers: 1, age: 60 * 24 * 388, res: '2160p', source: 'Remux', codec: 'HEVC', hdr: 'HDR10', audio: 'DTS-HD MA 7.1', group: 'ZQ', cfs: ['Remux Tier 01', 'HDR10', 'DTS-HD MA'] }),
    B('r14', 'Batman.2022.Part.I.2160p.WEB-DL.DDP5.1.DV.HDR10.H.265-PiRaTeS', { indexer: 'lumen', size: 16.1, seeders: 42, leechers: 7, age: 60 * 24 * 212, res: '2160p', source: 'WEB-DL', codec: 'HEVC', hdr: 'DV HDR10', audio: 'DD+ 5.1', group: 'PiRaTeS', titleMatch: 74, cfs: ['DV HDR10'] }),
    B('r15', 'The.Batman.2022.EXTENDED.1080p.WEBRip.x265.10bit-RARBG', { indexer: 'lumen', size: 3.9, seeders: 803, leechers: 40, age: 60 * 24 * 30, res: '1080p', source: 'WEBRip', codec: 'HEVC', hdr: null, audio: 'AAC 5.1', group: 'RARBG', edition: 'Extended', cfs: ['x265 (HD)'] }),
    B('r16', 'The.Batman.2022.720p.BluRay.x264.DTS-SPARKS', { indexer: 'aurora', size: 8.4, seeders: 37, leechers: 0, age: 60 * 24 * 402, res: '720p', source: 'BluRay', codec: 'AVC', hdr: null, audio: 'DTS 5.1', group: 'SPARKS', cfs: [] }),
    B('r17', 'The.Batman.2022.2160p.MA.WEB-DL.DDP5.1.SDR.H.265-NTb', { indexer: 'nimbus', size: 15.2, seeders: 22, leechers: 1, age: 60 * 24 * 560, res: '2160p', source: 'WEB-DL', codec: 'HEVC', hdr: null, audio: 'DD+ 5.1', group: 'NTb', cfs: ['WEB Tier 01', 'MA'] }),
    B('r18', 'The.Batman.2022.1080p.BluRay.REMUX.AVC.TrueHD.7.1.Atmos-FraMeSToR', { indexer: 'aurora', size: 35.6, seeders: 88, leechers: 2, age: 60 * 24 * 409, res: '1080p', source: 'Remux', codec: 'AVC', hdr: null, audio: 'TrueHD Atmos 7.1', group: 'FraMeSToR', cfs: ['Remux Tier 01', 'TrueHD Atmos'] }),
    B('r19', 'The.Batman.2022.2160p.AMZN.WEB-DL.DDP5.1.HDR10P.H.265-NTb', { indexer: 'orbit', size: 19.8, seeders: 47, leechers: 3, age: 60 * 24 * 300, res: '2160p', source: 'WEB-DL', codec: 'HEVC', hdr: 'HDR10+', audio: 'DD+ 5.1', group: 'NTb', cfs: ['WEB Tier 01', 'HDR10+'] }),
    B('r20', 'The.Batman.2022.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb', { indexer: 'orbit', size: 8.2, seeders: 52, leechers: 1, age: 60 * 24 * 300, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'NTb', cfs: ['WEB Tier 01'] }),
    B('r21', 'The.Batman.2022.1080p.HMAX.WEB-DL.DD5.1.H.264-SMURF', { indexer: 'lumen', size: 7.6, seeders: 129, leechers: 4, age: 60 * 24 * 563, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD 5.1', group: 'SMURF', cfs: ['WEB Tier 02'] }),
    B('r22', 'The.Batman.2022.2160p.WEBRip.x265.10bit.HDR10.DDP5.1-SWAXX', { indexer: 'lumen', size: 17.3, seeders: 301, leechers: 25, age: 60 * 24 * 558, res: '2160p', source: 'WEBRip', codec: 'HEVC', hdr: 'HDR10', audio: 'DD+ 5.1', group: 'SWAXX', cfs: ['HDR10'] }),
    B('r23', 'The.Batman.2022.1080p.BluRay.x265.10bit.DTS-HD.MA.7.1-RARBG', { indexer: 'lumen', size: 6.3, seeders: 655, leechers: 31, age: 60 * 24 * 401, res: '1080p', source: 'BluRay', codec: 'HEVC', hdr: null, audio: 'DTS-HD MA 7.1', group: 'RARBG', cfs: ['DTS-HD MA', 'x265 (HD)'] }),
    B('r24', 'The.Batman.2022.UHD.BluRay.2160p.DV.HDR10.TrueHD.Atmos.7.1.HEVC.REMUX-BLURANiUM', { indexer: 'nimbus', size: 78.9, seeders: 64, leechers: 4, age: 60 * 24 * 404, res: '2160p', source: 'Remux', codec: 'HEVC', hdr: 'DV HDR10', audio: 'TrueHD Atmos 7.1', group: 'BLURANiUM', cfs: ['Remux Tier 01', 'DV HDR10', 'TrueHD Atmos'] }),
  ],
};

// ───────────────────────────── Severance: two seasons, Profile TV Standard
const S1 = ['Good News About Hell', 'Half Loop', 'In Perpetuity', 'The You You Are', 'The Grim Barbarity of Optics and Design', 'Hide and Seek', 'Defiant Jazz', 'What’s for Dinner?', 'The We We Are'];
const S2 = ['Hello, Ms. Cobel', 'Goodbye, Mrs. Selvig', 'Who Is Alive?', 'Woe’s Hollow', 'Trojan’s Horse', 'Attila', 'Chikhai Bardo', 'Sweet Vitriol', 'After Hours', 'Cold Harbor'];
const pad = (n) => String(n).padStart(2, '0');
const sevItems = [
  ...S1.map((t, i) => ({ id: `s01e${pad(i + 1)}`, level: 's01', season: 1, ep: i + 1, short: `S01E${pad(i + 1)}`, label: t })),
  ...S2.map((t, i) => ({ id: `s02e${pad(i + 1)}`, level: 's02', season: 2, ep: i + 1, short: `S02E${pad(i + 1)}`, label: t })),
];
const sevCells = {};
for (const it of sevItems) {
  const set = (t, c) => { sevCells[`${it.id}|Standard|${t}`] = c; };
  if (it.season === 1) { set('hd', { state: 'met', score: 2000 }); set('lo', { state: 'met', score: 2200 }); continue; }
  const e = it.ep;
  if (e <= 4) { set('hd', { state: 'met', score: 2000 }); set('lo', { state: 'upgrade', score: 1750 }); } else if (e <= 6) { set('hd', { state: 'met', score: 2000 }); set('lo', { state: 'missing' }); } else if (e === 7) { set('hd', { state: 'upgrade', score: 1100 }); set('lo', { state: 'upgrade', score: 1750 }); } else if (e <= 9) { set('hd', { state: 'missing' }); set('lo', { state: 'missing' }); } else { set('hd', { state: 'failed', note: 'Release fault: unregistered on Aurora' }); set('lo', { state: 'missing' }); }
}
const s2 = (a, b = a) => sevItems.filter((i) => i.season === 2 && i.ep >= a && i.ep <= b).map((i) => i.id);
const s1all = sevItems.filter((i) => i.season === 1).map((i) => i.id);

const S = (id, name, o) => ({ id, name, lang: 'English', titleMatch: 100, edition: 'Standard', seeders: 0, leechers: 0, ...o });

const severance = {
  id: 'severance',
  label: 'Severance',
  work: { id: 'severance', title: 'Severance', year: 2022, kind: 'series', meta: ['2022', 'Apple TV+', 'TV-MA', 'Drama · Mystery'], art: art('series', 95396), overview: 'Mark leads a team of office workers whose memories have been surgically divided between their work and personal lives.' },
  profile: { name: 'TV Standard' },
  targets: [
    { id: 'hd', name: '1080p', res: '1080p', hdr: 'any', sourcing: 'Downloaded', minScore: 0, cutoff: 1800, upgradeUntil: 2000, size: [1, 8], hold: 90 },
    { id: 'lo', name: '720p low bitrate', res: '720p', hdr: 'forbid', sourcing: 'Downloaded', minScore: 0, cutoff: 1700, upgradeUntil: 2200, size: [0.2, 1.2], hold: 0 },
  ],
  editions: ['Standard'],
  levels: [{ id: 's01', label: 'Season 1', short: 'S01' }, { id: 's02', label: 'Season 2', short: 'S02' }],
  items: sevItems,
  cells: sevCells,
  defaultLevel: 's02',
  defaultItem: 's02e08',
  defaultCell: 's02e08|Standard|hd',
  cf: {
    'WEB Tier 01': { hd: 1700, lo: 1650 },
    'WEB Tier 02': { hd: 1650, lo: 1650 },
    'WEB Tier 03': { hd: 1600, lo: 1600 },
    'ATVP': { hd: 100, lo: 100 },
    'DD+ Atmos': { hd: 100 },
    'x265 (HD)': { hd: -1000, lo: 400 },
    'Repack': { hd: 5, lo: 5 },
    'LQ': { hd: -10000, lo: -10000 },
  },
  releases: [
    S('s01', 'Severance.S02.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', { indexer: 'aurora', coverage: s2(1, 10), size: 38.4, seeders: 164, leechers: 6, age: 60 * 24 * 238, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ Atmos 5.1', group: 'FLUX', cfs: ['WEB Tier 01', 'ATVP', 'DD+ Atmos'] }),
    S('s02', 'Severance.S02.2160p.ATVP.WEB-DL.DDP5.1.Atmos.DV.HDR10.H.265-FLUX', { indexer: 'aurora', coverage: s2(1, 10), size: 96.1, seeders: 88, leechers: 3, age: 60 * 24 * 238, res: '2160p', source: 'WEB-DL', codec: 'HEVC', hdr: 'DV HDR10', audio: 'DD+ Atmos 5.1', group: 'FLUX', cfs: ['WEB Tier 01', 'ATVP'] }),
    S('s03', 'Severance.S02.720p.ATVP.WEB-DL.DDP5.1.H.265-MiNX', { indexer: 'lumen', coverage: s2(1, 10), size: 4.1, seeders: 412, leechers: 19, age: 60 * 24 * 236, res: '720p', source: 'WEB-DL', codec: 'HEVC', hdr: null, audio: 'DD+ 5.1', group: 'MiNX', cfs: ['WEB Tier 02', 'ATVP', 'x265 (HD)'] }),
    S('s05', 'Severance.S01.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', { indexer: 'aurora', coverage: s1all, size: 31.2, seeders: 203, leechers: 2, age: 60 * 24 * 1300, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ Atmos 5.1', group: 'FLUX', cfs: ['WEB Tier 01', 'ATVP', 'DD+ Atmos'] }),
    S('s06', 'Severance.S01-S02.1080p.ATVP.WEB-DL.DDP5.1.H.264-playWEB', { indexer: 'nimbus', coverage: [...s1all, ...s2(1, 10)], size: 64.8, seeders: 71, leechers: 8, age: 60 * 24 * 230, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'playWEB', cfs: ['WEB Tier 01', 'ATVP'] }),
    S('s10', 'Severance.S02E08.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', { indexer: 'nimbus', coverage: s2(8), size: 3.9, seeders: 97, leechers: 2, age: 60 * 24 * 252, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ Atmos 5.1', group: 'FLUX', freeleech: true, cfs: ['WEB Tier 01', 'ATVP', 'DD+ Atmos'] }),
    S('s11', 'Severance.S02E08.720p.ATVP.WEB-DL.DDP5.1.H.265-MiNX', { indexer: 'lumen', coverage: s2(8), size: 0.42, seeders: 233, leechers: 4, age: 60 * 24 * 252, res: '720p', source: 'WEB-DL', codec: 'HEVC', hdr: null, audio: 'DD+ 5.1', group: 'MiNX', cfs: ['WEB Tier 02', 'ATVP', 'x265 (HD)'] }),
    S('s12', 'Severance.S02E08.1080p.WEB.H264-SuccessfulCrab', { indexer: 'lumen', coverage: s2(8), size: 2.6, seeders: 318, leechers: 12, age: 60 * 24 * 252, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'SuccessfulCrab', cfs: ['WEB Tier 03'] }),
    S('s13', 'Severance.S02E09.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', { indexer: 'orbit', coverage: s2(9), size: 4.2, seeders: 3, leechers: 41, age: 50, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ Atmos 5.1', group: 'FLUX', cfs: ['WEB Tier 01', 'ATVP', 'DD+ Atmos'] }),
    S('s14', 'Severance.S02E09.720p.ATVP.WEB-DL.DDP5.1.H.265-MiNX', { indexer: 'lumen', coverage: s2(9), size: 0.45, seeders: 140, leechers: 6, age: 60 * 24 * 245, res: '720p', source: 'WEB-DL', codec: 'HEVC', hdr: null, audio: 'DD+ 5.1', group: 'MiNX', cfs: ['WEB Tier 02', 'ATVP', 'x265 (HD)'] }),
    S('s15', 'Severance.S02E10.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', { indexer: 'aurora', coverage: s2(10), size: 4.6, seeders: 0, leechers: 0, age: 60 * 24 * 238, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ Atmos 5.1', group: 'FLUX', cfs: ['WEB Tier 01', 'ATVP', 'DD+ Atmos'], blocklisted: { on: 'Oct 9', cause: 'Unregistered on Aurora, twice 30 min apart' } }),
    S('s16', 'Severance.S02E10.1080p.WEB.H264-NHTFS', { indexer: 'lumen', coverage: s2(10), size: 2.9, seeders: 276, leechers: 8, age: 60 * 24 * 238, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'NHTFS', cfs: ['WEB Tier 03'] }),
    S('s17', 'Severance.S02E07.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', { indexer: 'aurora', coverage: s2(7), size: 4.0, seeders: 85, leechers: 1, age: 60 * 24 * 259, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ Atmos 5.1', group: 'FLUX', cfs: ['WEB Tier 01', 'ATVP', 'DD+ Atmos'] }),
    S('s18', 'Severance.S02E07.1080p.WEBRip.x264-ION10', { indexer: 'lumen', coverage: s2(7), size: 1.3, seeders: 512, leechers: 20, age: 60 * 24 * 259, res: '1080p', source: 'WEBRip', codec: 'AVC', hdr: null, audio: 'AAC 5.1', group: 'ION10', cfs: [] }),
    S('s19', 'Severance.S02E09.Uncut.1080p.WEB.H264-GRACE', { indexer: 'orbit', coverage: s2(9), size: 2.8, seeders: 19, leechers: 2, age: 60 * 24 * 244, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'GRACE', edition: 'Uncut', misparsed: true, cfs: ['WEB Tier 03'] }),
    S('s20', 'Scissione.S02E08.iTALiAN.1080p.ATVP.WEB-DL.DDP5.1.H.264-iDN_CreW', { indexer: 'lumen', coverage: s2(8), size: 2.7, seeders: 24, leechers: 1, age: 60 * 24 * 250, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'iDN_CreW', lang: 'Italian', titleMatch: 41, cfs: ['ATVP'] }),
    S('s21', 'Severance.S02E05.720p.ATVP.WEB-DL.DDP5.1.H.264-NTb', { indexer: 'nimbus', coverage: s2(5), size: 1.1, seeders: 46, leechers: 0, age: 60 * 24 * 266, res: '720p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'NTb', cfs: ['WEB Tier 01', 'ATVP'] }),
    S('s22', 'Severance.S02E06.720p.ATVP.WEB-DL.DDP5.1.H.264-NTb', { indexer: 'nimbus', coverage: s2(6), size: 1.1, seeders: 44, leechers: 1, age: 60 * 24 * 259, res: '720p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'NTb', cfs: ['WEB Tier 01', 'ATVP'] }),
    S('s23', 'Severance.S02E08.1080p.ATVP.WEB-DL.DDP5.1.H.265-HHWEB', { indexer: 'orbit', coverage: s2(8), size: 0.62, seeders: 58, leechers: 3, age: 60 * 24 * 251, res: '1080p', source: 'WEB-DL', codec: 'HEVC', hdr: null, audio: 'DD+ 5.1', group: 'HHWEB', cfs: ['WEB Tier 02', 'ATVP', 'x265 (HD)'] }),
    S('s24', 'Severed.S02E08.1080p.WEB.H264-RAWR', { indexer: 'lumen', coverage: s2(8), size: 2.4, seeders: 12, leechers: 0, age: 60 * 24 * 251, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'RAWR', titleMatch: 68, cfs: ['WEB Tier 03'] }),
    S('s25', 'Severance.S02E04.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', { indexer: 'aurora', coverage: s2(4), size: 4.1, seeders: 77, leechers: 0, age: 60 * 24 * 273, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ Atmos 5.1', group: 'FLUX', cfs: ['WEB Tier 01', 'ATVP', 'DD+ Atmos'] }),
    S('s26', 'Severance.S02E01-E03.1080p.ATVP.WEB-DL.DDP5.1.H.264-NTb', { indexer: 'nimbus', coverage: s2(1, 3), size: 11.4, seeders: 21, leechers: 1, age: 60 * 24 * 280, res: '1080p', source: 'WEB-DL', codec: 'AVC', hdr: null, audio: 'DD+ 5.1', group: 'NTb', cfs: ['WEB Tier 01', 'ATVP'] }),
    S('s27', 'Severance.S02E08.1080p.WEBRip.x264-GalaxyTV', { indexer: 'lumen', coverage: s2(8), size: 0.9, seeders: 640, leechers: 33, age: 60 * 24 * 251, res: '1080p', source: 'WEBRip', codec: 'AVC', hdr: null, audio: 'AAC 2.0', group: 'GalaxyTV', cfs: ['LQ'] }),
    S('s28', 'Severance.S02E10.720p.ATVP.WEB-DL.DDP5.1.H.265-MiNX', { indexer: 'orbit', coverage: s2(10), size: 0.47, seeders: 52, leechers: 2, age: 60 * 24 * 238, res: '720p', source: 'WEB-DL', codec: 'HEVC', hdr: null, audio: 'DD+ 5.1', group: 'MiNX', cfs: ['WEB Tier 02', 'ATVP', 'x265 (HD)'] }),
  ],
};

export const fixtures = { movie: batman, series: severance };
