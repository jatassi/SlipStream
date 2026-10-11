// PROTOTYPE fixture for the Work page at scale. Library state, numbers and the TV Editions are invented.
//
// A Work is a tree the core owns: levels from its Content Type, Items at the leaves. Every Item holds one cell
// per Edition wanted × Target in the Work's Profile. A cell is exactly one of:
//   met          a file fills the Target for that Edition and its score reached the cutoff
//   upgrade      a file fills it, below the cutoff, and a better release has been found
//   missing      nothing fills it; `why` says what happens next (searching, not-found, waiting, grace, queued)
//   downloading  a job is acquiring it; `stage` is Downloading, Transfer or Import
//   converting   a Conversion is making it from another Target's file
//   failed       the last attempt failed at `stage`, with its cause and the recovery already taken
//   unaired      the Content Type's availability date gates search (calendar, Q14)

export const targets = [
  { id: '2160p-hdr', name: '2160p HDR', short: '2160p HDR' },
  { id: '1080p-sdr', name: '1080p SDR', short: '1080p' },
  { id: '720p-mobile', name: '720p Mobile', short: '720p' },
];

export const profiles = {
  archive: {
    id: 'archive', name: 'Archive',
    targets: [
      { target: '2160p-hdr', sourcing: 'downloaded', label: 'Downloaded' },
      { target: '1080p-sdr', sourcing: 'download-then-derive', from: '2160p-hdr', grace: '3 days', label: 'Downloaded, or derived from 2160p HDR after 3 days' },
      { target: '720p-mobile', sourcing: 'derived', from: '1080p-sdr', label: 'Derived from 1080p SDR' },
    ],
  },
};

const tmdb = (kind, id) => ({ poster: `art/${kind}/${id}_poster.jpg`, backdrop: `art/${kind}/${id}_backdrop.jpg`, logo: `art/${kind}/${id}_logo.png` });

const UHD = { video: 'HEVC · HDR10', audio: 'TrueHD Atmos 7.1' };
const FHD = { video: 'HEVC · SDR', audio: 'EAC3 5.1 + AAC 2.0' };
const HD = { video: 'H.264 · SDR · 3 Mb/s', audio: 'AAC 2.0' };

// Live jobs the cells point at. scale.html adds them to the ticker on the Work screen only, so Home keeps its own pipeline.
export const jobs = [
  { id: 'j-av-ext-dl', stage: 'Downloading', node: 'seedbox-ams', status: 'running', progress: 0.44, rate: 31.2, unit: 'MB/s', etaMin: 12 },
  { id: 'j-av-thr-conv', stage: 'Conversion', node: 'forge', status: 'running', progress: 0.27, rate: 212, unit: 'fps', etaMin: 9 },
  { id: 'j-tl-s1e8-conv', stage: 'Conversion', node: 'forge', status: 'running', progress: 0.81, rate: 236, unit: 'fps', etaMin: 2 },
  { id: 'j-tl-s2e3-uhd', stage: 'Downloading', node: 'seedbox-ams', status: 'running', progress: 0.58, rate: 18.4, unit: 'MB/s', etaMin: 3 },
];

export const avatar = {
  id: 'avatar', contentType: 'movie', title: 'Avatar', year: 2009, runtime: '2h 42m · 2h 58m · 3h 8m', rating: 'PG-13', genres: ['Science Fiction', 'Adventure'],
  overview: 'A paraplegic Marine dispatched to the moon Pandora on a unique mission becomes torn between following orders and protecting the world he feels is his home.',
  art: tmdb('movie', 19995), profile: 'archive', monitored: true,
  editions: [
    { id: 'theatrical', name: 'Theatrical', note: 'Profile default · 2h 42m' },
    { id: 'special', name: 'Special Edition', note: 'Wanted on this Work · 2h 50m' },
    { id: 'extended', name: "Extended Collector's", note: 'Wanted on this Work · 2h 58m' },
  ],
  tree: [{
    kind: 'item', id: 'avatar-movie',
    cells: {
      theatrical: {
        '2160p-hdr': { state: 'met', score: 2210, cutoff: 2000, size: '64.1 GB', ...UHD, source: 'downloaded', file: 'Avatar.2009.2160p.UHD.BluRay.REMUX.HDR10.TrueHD.Atmos.7.1-FraMeSToR.mkv' },
        '1080p-sdr': { state: 'met', score: 1640, cutoff: 1500, size: '14.2 GB', ...FHD, source: 'downloaded', file: 'Avatar.2009.1080p.BluRay.x265.DDP5.1-GROUP.mkv' },
        '720p-mobile': { state: 'converting', job: 'j-av-thr-conv', from: '1080p SDR file', node: 'forge', settings: 'H.264 · 3 Mb/s · AAC stereo', estSize: '≈ 3.6 GB' },
      },
      special: {
        '2160p-hdr': { state: 'upgrade', score: 1420, cutoff: 2000, size: '22.8 GB', video: 'HEVC · HDR10', audio: 'EAC3 5.1', source: 'downloaded', file: 'Avatar.2009.Special.Edition.2160p.WEB-DL.HDR10.DDP5.1-GROUP.mkv', found: { release: 'Avatar.2009.Special.Edition.2160p.UHD.BluRay.REMUX.HDR10.TrueHD.7.1-EPSiLON', score: 2180, size: '61.7 GB', indexer: 'Aurora' } },
        '1080p-sdr': { state: 'met', score: 1550, cutoff: 1500, size: '11.9 GB', ...FHD, source: 'derived', file: 'Avatar.2009.Special.Edition.1080p.SDR.mkv' },
        '720p-mobile': { state: 'missing', why: 'queued', detail: 'Queued for Conversion on forge after the Theatrical one' },
      },
      extended: {
        '2160p-hdr': { state: 'failed', stage: 'Import', at: '2026-10-08T20:52:00', cause: 'Probe found 2h 42m of video; the release said Extended (2h 58m).', recovery: 'Blocklisted the release. Searching again every 6 hours.', release: 'Avatar.2009.Extended.2160p.UHD.BluRay.HDR10.DTS-HD.MA.7.1-NOGRP' },
        '1080p-sdr': { state: 'downloading', job: 'j-av-ext-dl', release: 'Avatar.2009.Extended.Collectors.Edition.1080p.BluRay.x265.DTS-HD.MA.5.1-GROUP', size: '18.3 GB', score: 1580 },
        '720p-mobile': { state: 'missing', why: 'waiting', detail: 'Waits for the 1080p SDR file' },
      },
    },
  }],
};

const S1 = ['When You\'re Lost in the Darkness', 'Infected', 'Long, Long Time', 'Please Hold to My Hand', 'Endure and Survive', 'Kin', 'Left Behind', 'When We Are in Need', 'Look for the Light'];
const S2 = [['Future Days', '2026-09-28'], ['Through the Valley', '2026-10-07'], ['The Path', '2026-10-07'], ['Day One', '2026-10-12'], ['Feel Her Love', '2026-10-19'], ['The Price', '2026-10-26'], ['Convergence', '2026-11-02']];

const met = (score, source = 'downloaded', spec = UHD, size = '6.2 GB') => ({ state: 'met', score, cutoff: score - 100, size, ...spec, source });
const rel = (s, e, t) => `The.Last.of.Us.S${String(s).padStart(2, '0')}E${String(e).padStart(2, '0')}.${t}`;

function s1Cells(e) {
  const c = {
    standard: { '2160p-hdr': met(1890, 'downloaded', UHD, '7.4 GB'), '1080p-sdr': met(1620, 'downloaded', FHD, '2.9 GB'), '720p-mobile': met(1200, 'derived', HD, '1.1 GB') },
    extended: { '2160p-hdr': met(1950, 'downloaded', UHD, '8.1 GB'), '1080p-sdr': met(1500, 'derived', FHD, '3.2 GB'), '720p-mobile': met(1180, 'derived', HD, '1.2 GB') },
    directors: { '2160p-hdr': { state: 'missing', why: 'not-found', detail: 'No Director\'s cut released for this episode · last searched 2 h ago' }, '1080p-sdr': { state: 'missing', why: 'waiting', detail: 'Waits for a 2160p HDR file or a 1080p release' }, '720p-mobile': { state: 'missing', why: 'waiting', detail: 'Waits for the 1080p SDR file' } },
  };
  if (e === 1) c.directors = { '2160p-hdr': met(1910, 'downloaded', UHD, '9.6 GB'), '1080p-sdr': met(1500, 'derived', FHD, '3.6 GB'), '720p-mobile': met(1150, 'derived', HD, '1.3 GB') };
  if (e === 3) c.standard['2160p-hdr'] = { state: 'upgrade', score: 1310, cutoff: 1790, size: '4.1 GB', video: 'HEVC · HDR10', audio: 'EAC3 5.1', source: 'downloaded', found: { release: rel(1, 3, '2160p.UHD.BluRay.REMUX.DV.HDR10.TrueHD.7.1-FraMeSToR'), score: 1940, size: '14.9 GB', indexer: 'Aurora' } };
  if (e === 7) c.extended['2160p-hdr'] = { state: 'upgrade', score: 1480, cutoff: 1850, size: '5.0 GB', video: 'HEVC · HDR10', audio: 'EAC3 5.1', source: 'downloaded', found: { release: rel(1, 7, 'Extended.2160p.UHD.BluRay.HDR10.TrueHD.7.1-GROUP'), score: 1960, size: '12.2 GB', indexer: 'Torrentleech' } };
  if (e === 5) {
    c.extended['2160p-hdr'] = { state: 'failed', stage: 'Downloading', at: '2026-10-08T18:05:00', cause: 'The torrent stalled at 61% with no seeders for 6 hours.', recovery: 'Removed the grab and searching again. 3 other releases are waiting on approval rules.', release: rel(1, 5, 'Extended.2160p.BluRay.HDR10.DDP5.1-SMURF') };
    c.extended['1080p-sdr'] = { state: 'missing', why: 'grace', detail: 'Derives from 2160p HDR once it lands, or downloads a 1080p release first' };
    c.extended['720p-mobile'] = { state: 'missing', why: 'waiting', detail: 'Waits for the 1080p SDR file' };
  }
  if (e === 8) c.extended['720p-mobile'] = { state: 'converting', job: 'j-tl-s1e8-conv', from: '1080p SDR file', node: 'forge', settings: 'H.264 · 3 Mb/s · AAC stereo', estSize: '≈ 1.2 GB' };
  return c;
}

function s2Cells(e) {
  const none = (why, detail) => ({ '2160p-hdr': { state: 'missing', why, detail }, '1080p-sdr': { state: 'missing', why: 'waiting', detail: 'Waits for a 2160p HDR file or a 1080p release' }, '720p-mobile': { state: 'missing', why: 'waiting', detail: 'Waits for the 1080p SDR file' } });
  const later = () => none('searching', 'Extended cuts usually arrive with the Blu-ray · searching daily');
  if (e === 1) return {
    standard: { '2160p-hdr': met(1880, 'downloaded', UHD, '7.9 GB'), '1080p-sdr': met(1600, 'downloaded', FHD, '3.0 GB'), '720p-mobile': met(1190, 'derived', HD, '1.1 GB') },
    extended: later(), directors: none('not-found', 'No Director\'s cut announced · last searched 2 h ago'),
  };
  if (e === 2) return {
    standard: { '2160p-hdr': met(1880, 'downloaded', UHD, '7.2 GB'), '1080p-sdr': { state: 'missing', why: 'grace', detail: 'Searching for a 1080p release until Oct 10, then derives from 2160p HDR · 2 days left' }, '720p-mobile': { state: 'missing', why: 'waiting', detail: 'Waits for the 1080p SDR file' } },
    extended: later(), directors: none('not-found', 'No Director\'s cut announced · last searched 2 h ago'),
  };
  if (e === 3) return {
    standard: { '2160p-hdr': { state: 'downloading', job: 'j-tl-s2e3-uhd', release: rel(2, 3, '2160p.MAX.WEB-DL.DV.HDR10.DDP5.1.Atmos-FLUX'), size: '8.6 GB', score: 1870 }, '1080p-sdr': { state: 'downloading', job: 'j-tlou-xfer', release: rel(2, 3, '1080p.MAX.WEB-DL.DDP5.1.Atmos.H.264-FLUX'), size: '5.2 GB', score: 1610 }, '720p-mobile': { state: 'missing', why: 'waiting', detail: 'Waits for the 1080p SDR file' } },
    extended: later(), directors: none('not-found', 'No Director\'s cut announced · last searched 2 h ago'),
  };
  return null;
}

const unaired = (airs) => Object.fromEntries(['standard', 'extended', 'directors'].map((ed) => [ed, Object.fromEntries(targets.map((t) => [t.id, { state: 'unaired', airs }]))]));

export const tlou = {
  id: 'tlou-scale', contentType: 'series', title: 'The Last of Us', year: 2023, network: 'HBO', rating: 'TV-MA', genres: ['Drama', 'Science Fiction'],
  overview: 'Twenty years after modern civilization has been destroyed, Joel, a hardened survivor, is hired to smuggle Ellie, a 14-year-old girl, out of an oppressive quarantine zone.',
  art: tmdb('series', 100088), profile: 'archive', monitored: true,
  editions: [
    { id: 'standard', name: 'Standard', note: 'Profile default' },
    { id: 'extended', name: 'Extended', note: 'Wanted on this Work' },
    { id: 'directors', name: "Director's", note: 'Wanted on this Work' },
  ],
  tree: [
    { kind: 'level', level: 'season', n: 1, children: S1.map((title, i) => ({ kind: 'item', id: `tl-1-${i + 1}`, n: i + 1, title, airs: null, cells: s1Cells(i + 1) })) },
    { kind: 'level', level: 'season', n: 2, children: S2.map(([title, airs], i) => {
      const cells = s2Cells(i + 1) ?? unaired(airs);
      return { kind: 'item', id: `tl-2-${i + 1}`, n: i + 1, title, airs, cells };
    }) },
  ],
};

export const works = { avatar, tlou };
