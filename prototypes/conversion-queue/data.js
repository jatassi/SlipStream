// PROTOTYPE fixture for the Conversion queue and the backfill dialog. Library state, people, Nodes and numbers are invented.
// Artwork is v1's local cache, copied by run.sh into ./art and never committed.

export const now = new Date('2026-10-08T21:40:00');

export const admin = { name: 'Jackson', initials: 'JA' };

// ───────────────────────────── Content Type descriptors
// Core renders every Work, Item and group from these. No surface may branch on a concrete Content Type id.
// layout hints:
//   itemInRow   'omit'  the Work has one Item, so a row names the Work (plus the Edition when it isn't Standard)
//               'show'  a row names the Item under the Work, formatted by `item`
//   collapse    a level key: in long lists, consecutive Conversions for Items under one node of that level fold into one group row
//   pickUnit    what the backfill dialog selects: always the Work, with `size` describing how many Items it covers
//   thumb       artwork kind for row thumbnails; hero is the kind used for the ambient tint and headers
export const contentTypes = {
  movie: {
    id: 'movie', label: 'Movie', plural: 'Movies',
    levels: [],
    artwork: { thumb: 'poster', hero: 'backdrop', ratio: '2 / 3' },
    fields: ['year', 'runtime'],
    layout: { itemInRow: 'omit', collapse: null, pickUnit: 'work' },
    item: () => null,
    size: () => '1 file',
    group: () => null,
  },
  series: {
    id: 'series', label: 'Series', plural: 'Series',
    levels: [{ key: 'season', label: 'Season', short: 'S' }, { key: 'episode', label: 'Episode', short: 'E' }],
    artwork: { thumb: 'poster', hero: 'backdrop', ratio: '2 / 3' },
    fields: ['year', 'network'],
    layout: { itemInRow: 'show', collapse: 'season', pickUnit: 'work' },
    item: (it) => `S${String(it.season).padStart(2, '0')}E${String(it.episode).padStart(2, '0')}${it.title ? ` · ${it.title}` : ''}`,
    size: (w) => `${w.seasons} seasons · ${w.episodes} episodes`,
    group: (it) => `Season ${it.season}`,
  },
};

const art = (dir, id) => ({ poster: `art/${dir}/${id}_poster.jpg`, backdrop: `art/${dir}/${id}_backdrop.jpg`, logo: `art/${dir}/${id}_logo.png` });
const mv = (id, tmdb, title, year, runtime) => ({ id, type: 'movie', title, year, runtime, art: art('movie', tmdb) });
const tv = (id, tmdb, title, year, network, seasons, episodes) => ({ id, type: 'series', title, year, network, seasons, episodes, art: art('series', tmdb) });

export const works = Object.fromEntries([
  mv('dune-2', 693134, 'Dune: Part Two', 2024, '2h 46m'),
  mv('oppenheimer', 872585, 'Oppenheimer', 2023, '3h 0m'),
  mv('interstellar', 157336, 'Interstellar', 2014, '2h 49m'),
  mv('top-gun-maverick', 361743, 'Top Gun: Maverick', 2022, '2h 11m'),
  mv('eeaao', 545611, 'Everything Everywhere All at Once', 2022, '2h 19m'),
  mv('the-batman', 414906, 'The Batman', 2022, '2h 57m'),
  mv('black-panther', 284054, 'Black Panther', 2018, '2h 15m'),
  mv('the-matrix', 603, 'The Matrix', 1999, '2h 16m'),
  mv('the-dark-knight', 155, 'The Dark Knight', 2008, '2h 32m'),
  mv('the-prestige', 1124, 'The Prestige', 2006, '2h 10m'),
  mv('dune', 438631, 'Dune', 2021, '2h 35m'),
  mv('br2049', 335984, 'Blade Runner 2049', 2017, '2h 44m'),
  mv('arrival', 329865, 'Arrival', 2016, '1h 56m'),
  mv('fury-road', 76341, 'Mad Max: Fury Road', 2015, '2h 0m'),
  mv('inception', 27205, 'Inception', 2010, '2h 28m'),
  mv('parasite', 496243, 'Parasite', 2019, '2h 13m'),
  mv('budapest', 120467, 'The Grand Budapest Hotel', 2014, '1h 40m'),
  mv('whiplash', 244786, 'Whiplash', 2014, '1h 47m'),
  mv('la-la-land', 313369, 'La La Land', 2016, '2h 9m'),
  mv('gravity', 49047, 'Gravity', 2013, '1h 31m'),
  mv('1917', 530915, '1917', 2019, '1h 59m'),
  mv('tenet', 577922, 'Tenet', 2020, '2h 30m'),
  mv('no-time-to-die', 370172, 'No Time to Die', 2021, '2h 43m'),
  mv('avatar-2', 76600, 'Avatar: The Way of Water', 2022, '3h 12m'),
  mv('spider-verse-2', 569094, 'Spider-Man: Across the Spider-Verse', 2023, '2h 20m'),
  mv('barbie', 346698, 'Barbie', 2023, '1h 54m'),
  mv('flower-moon', 466420, 'Killers of the Flower Moon', 2023, '3h 26m'),
  mv('poor-things', 792307, 'Poor Things', 2023, '2h 21m'),
  mv('holdovers', 840430, 'The Holdovers', 2023, '2h 13m'),
  mv('past-lives', 666277, 'Past Lives', 2023, '1h 46m'),
  mv('godzilla-minus-one', 940721, 'Godzilla Minus One', 2023, '2h 5m'),
  mv('furiosa', 786892, 'Furiosa: A Mad Max Saga', 2024, '2h 29m'),
  mv('civil-war', 929590, 'Civil War', 2024, '1h 49m'),
  mv('alien-romulus', 945961, 'Alien: Romulus', 2024, '1h 59m'),
  mv('wicked', 402431, 'Wicked', 2024, '2h 41m'),
  mv('gladiator-2', 558449, 'Gladiator II', 2024, '2h 28m'),
  mv('conclave', 974576, 'Conclave', 2024, '2h 0m'),
  mv('anora', 1064213, 'Anora', 2024, '2h 19m'),
  mv('the-substance', 933260, 'The Substance', 2024, '2h 21m'),
  mv('mi-dead-reckoning', 575264, 'Mission: Impossible – Dead Reckoning Part One', 2023, '2h 44m'),
  mv('john-wick-4', 603692, 'John Wick: Chapter 4', 2023, '2h 49m'),
  mv('joker', 475557, 'Joker', 2019, '2h 2m'),
  mv('pulp-fiction', 680, 'Pulp Fiction', 1994, '2h 34m'),
  mv('forrest-gump', 13, 'Forrest Gump', 1994, '2h 22m'),
  mv('blade-runner', 78, 'Blade Runner', 1982, '1h 57m'),
  mv('2001', 62, '2001: A Space Odyssey', 1968, '2h 29m'),
  mv('alien', 348, 'Alien', 1979, '1h 57m'),
  mv('the-godfather', 238, 'The Godfather', 1972, '2h 55m'),
  tv('severance', 95396, 'Severance', 2022, 'Apple TV+', 2, 19),
  tv('the-boys', 76479, 'The Boys', 2019, 'Prime Video', 4, 32),
  tv('hotd', 94997, 'House of the Dragon', 2022, 'HBO', 2, 18),
  tv('tlou', 100088, 'The Last of Us', 2023, 'HBO', 2, 16),
  tv('bcs', 60059, 'Better Call Saul', 2015, 'AMC', 6, 63),
  tv('got', 1399, 'Game of Thrones', 2011, 'HBO', 8, 73),
  tv('narcos', 63351, 'Narcos', 2015, 'Netflix', 3, 30),
].map((w) => [w.id, w]));

// ───────────────────────────── Targets, Conversion settings, Profiles

export const targets = {
  '2160p-hdr': { id: '2160p-hdr', name: '2160p HDR', sourcing: 'downloaded' },
  '1080p-sdr': { id: '1080p-sdr', name: '1080p SDR', sourcing: 'derived', from: '2160p-hdr', settings: { version: 3, summary: 'HEVC · 1080p · SDR · keep audio + stereo AAC · MKV', toneMap: true } },
  '1080p': { id: '1080p', name: '1080p', sourcing: 'downloaded', settings: { version: 2, summary: 'Download steps · remux to MKV · add loudness-normalized stereo', adjust: true } },
  '720p-mobile': { id: '720p-mobile', name: '720p mobile', sourcing: 'derived', from: '1080p-sdr', settings: { version: 1, summary: 'H.264 · 720p · SDR · stereo AAC · MP4', toneMap: true } },
};

export const profiles = [
  { id: 'cinema', name: 'Cinema', contentType: 'movie', isDefault: true, targets: ['2160p-hdr', '1080p-sdr'], works: 48 },
  { id: 'tv', name: 'TV Standard', contentType: 'series', isDefault: true, targets: ['1080p'], works: 7 },
  { id: 'archive', name: 'Archive 1080p', contentType: 'movie', isDefault: false, targets: ['1080p'], works: 131 },
];

// ───────────────────────────── Converting Nodes
// proven: capability keys this Node has passed a trial run for on its current ffmpeg and driver.
// A Conversion's `needs` must be in a Node's proven set, and the Node must be online, for the Node to be capable.
export const capabilities = {
  'hevc10-hdr10-sdr': 'HEVC 10-bit HDR10 → SDR',
  'dv5-sdr': 'Dolby Vision Profile 5 → SDR (DV-aware tone mapper)',
  'dv7-81': 'Dolby Vision Profile 7 → 8.1',
  'h264-720p': 'H.264 720p encode',
  'remux-audio': 'Remux and audio steps',
};

export const nodes = [
  {
    id: 'forge', name: 'forge', kind: 'Worker', where: 'Home · LAN', status: 'online',
    hw: 'RTX 4070 · NVENC · CUDA', ffmpeg: 'jellyfin-ffmpeg 7.1.1-3 · driver 560.35',
    proven: ['hevc10-hdr10-sdr', 'dv7-81', 'h264-720p', 'remux-audio'],
    trials: [{ cap: 'dv5-sdr', result: 'failed', at: '2026-10-07T09:12:00', note: 'CUDA tone mapper rejected DV P5 after the driver update to 560.35' }],
    slots: 3, cap: 3, activeHours: null, scratchFreeGB: 212,
    mediaServer: { name: 'Jellyfin', transcoding: 1, detail: 'transcoding for Maya on Apple TV' },
    speed: '≈ 6.2× realtime on 2160p HDR → 1080p',
  },
  {
    id: 'atlas', name: 'atlas', kind: 'Controller', where: 'Home · LAN', status: 'online',
    hw: 'Ryzen 9 7950X · software x265', ffmpeg: 'jellyfin-ffmpeg 7.1.1-3',
    proven: ['hevc10-hdr10-sdr', 'dv7-81', 'h264-720p', 'remux-audio'],
    trials: [],
    slots: 1, cap: 1, activeHours: { from: '01:00', to: '07:00' }, scratchFreeGB: 1800,
    mediaServer: null,
    speed: '≈ 0.9× realtime on 2160p HDR → 1080p',
  },
  {
    id: 'studio', name: 'studio', kind: 'Worker', where: 'Home · LAN', status: 'offline', offlineSince: '2026-10-08T18:02:00',
    hw: 'Mac mini M2 Pro · VideoToolbox · Metal', ffmpeg: 'jellyfin-ffmpeg 7.1.1-3',
    proven: ['hevc10-hdr10-sdr', 'dv5-sdr', 'h264-720p', 'remux-audio'],
    trials: [],
    slots: 2, cap: 2, activeHours: null, scratchFreeGB: 340,
    mediaServer: null,
    speed: '≈ 4.1× realtime on 2160p HDR → 1080p',
  },
];

// ───────────────────────────── Conversions
// priority, highest first: bump (admin moved it up), new (a new Item), rederive (after a source upgrade), backfill
export const priorities = [
  { id: 'bump', label: 'Moved up' },
  { id: 'new', label: 'New' },
  { id: 'rederive', label: 'After upgrade' },
  { id: 'backfill', label: 'Backfill' },
];

// kind: 'derive' makes a derived Target's file; 'adjust' runs a downloaded Target's remux and audio steps while the torrent seeds the original.
// state is one of:
//   { is: 'running',   node, progress, fps, etaMin }
//   { is: 'suspended', node, progress, cause }                 Media Server on that Node is transcoding; resumes on its own
//   { is: 'waiting',   reason, node?, ... }                     reason: queued | active-hours | source-conversion | disk-space | no-capable-node
//   { is: 'failed',    attempts: [{ node, at, error }] }        failed on two Nodes; nothing retries until the source changes or the admin retries
// Every waiting Conversion shows as "converting" on the Work's Edition × Target grid, with this reason.
const C = (o) => ({ settingsVersion: 1, edition: 'Standard', ...o });

export const conversions = [
  C({ id: 'c-dune', work: 'dune-2', target: '1080p-sdr', kind: 'derive', priority: 'new', settingsVersion: 3,
    source: { name: '2160p HDR file', spec: 'HEVC 10-bit · DV P7 + HDR10 · 74.6 GB' }, estSize: '9.8 GB',
    state: { is: 'running', node: 'forge', progress: 0.62, fps: 148, etaMin: 6 } }),
  C({ id: 'c-tlou', work: 'tlou', item: { season: 2, episode: 3, title: 'The Path' }, target: '1080p', kind: 'adjust', priority: 'new', settingsVersion: 2,
    source: { name: 'downloaded file', spec: 'H.264 · DDP 5.1 Atmos · 5.2 GB' }, estSize: '5.3 GB', steps: ['Remux to MKV', 'Add stereo track'],
    state: { is: 'running', node: 'forge', progress: 0.81, fps: 1240, etaMin: 1 } }),
  C({ id: 'c-opp', work: 'oppenheimer', target: '1080p-sdr', kind: 'derive', priority: 'new', settingsVersion: 3,
    source: { name: '2160p HDR file', spec: 'HEVC 10-bit · HDR10 · 78.4 GB' }, estSize: '11.2 GB',
    state: { is: 'suspended', node: 'forge', progress: 0.34, cause: 'Jellyfin on forge is transcoding for Maya on Apple TV', since: '2026-10-08T21:31:00' } }),
  C({ id: 'c-int', work: 'interstellar', target: '1080p-sdr', kind: 'derive', priority: 'bump', settingsVersion: 3,
    source: { name: '2160p HDR file', spec: 'HEVC 10-bit · HDR10 · 61.0 GB' }, estSize: '10.1 GB',
    state: { is: 'waiting', reason: 'queued', node: 'forge', startsAt: '2026-10-08T21:47:00', finishesAt: '2026-10-08T22:14:00' } }),
  C({ id: 'c-bat-adj', work: 'the-batman', target: '2160p-hdr', kind: 'adjust', priority: 'new', settingsVersion: 1,
    source: { name: 'downloaded file', spec: 'HEVC 10-bit · DV P7 FEL + HDR10 · 66.1 GB' }, estSize: '64.8 GB', steps: ['DV Profile 7 → 8.1', 'Add stereo track'],
    state: { is: 'waiting', reason: 'queued', node: 'forge', startsAt: '2026-10-08T21:48:00', finishesAt: '2026-10-08T22:02:00' } }),
  C({ id: 'c-bat', work: 'the-batman', target: '1080p-sdr', kind: 'derive', priority: 'new', settingsVersion: 3,
    source: { name: '2160p HDR file', spec: 'after its own Conversion' }, estSize: '10.6 GB',
    state: { is: 'waiting', reason: 'source-conversion', waitsOn: 'c-bat-adj', node: 'forge', startsAt: '2026-10-08T22:02:00', finishesAt: '2026-10-08T22:31:00' } }),
  C({ id: 'c-tgm', work: 'top-gun-maverick', target: '1080p-sdr', kind: 'derive', priority: 'rederive', settingsVersion: 3,
    source: { name: 'upgraded 2160p HDR file', spec: 'HEVC 10-bit · HDR10 · 58.3 GB · was WEB-DL' }, estSize: '8.9 GB',
    state: { is: 'waiting', reason: 'queued', node: 'forge', startsAt: '2026-10-08T22:14:00', finishesAt: '2026-10-08T22:37:00' } }),
  C({ id: 'c-bp', work: 'black-panther', target: '1080p-sdr', kind: 'derive', priority: 'rederive', settingsVersion: 3,
    source: { name: 'upgraded 2160p HDR file', spec: 'HEVC 10-bit · HDR10 · 55.0 GB' }, estSize: '8.4 GB',
    state: { is: 'waiting', reason: 'disk-space', node: 'atlas', needGB: 11, freeGB: 4, detail: 'atlas scratch has 4 GB free and needs 11 GB. It requeues when space frees; this is not a failure.' } }),
  C({ id: 'c-eeaao', work: 'eeaao', target: '1080p-sdr', kind: 'derive', priority: 'new', settingsVersion: 3, needs: 'dv5-sdr',
    source: { name: '2160p HDR file', spec: 'HEVC 10-bit · Dolby Vision Profile 5 · WEB-DL · 21.4 GB' }, estSize: '6.0 GB',
    state: { is: 'waiting', reason: 'no-capable-node', cause: 'dv5-sdr', since: '2026-10-08T20:58:00' } }),
  C({ id: 'c-civil', work: 'civil-war', target: '1080p-sdr', kind: 'derive', priority: 'new', settingsVersion: 3, needs: 'dv5-sdr',
    source: { name: '2160p HDR file', spec: 'HEVC 10-bit · Dolby Vision Profile 5 · WEB-DL · 18.9 GB' }, estSize: '5.1 GB',
    state: { is: 'waiting', reason: 'no-capable-node', cause: 'dv5-sdr', since: '2026-10-08T19:40:00' } }),
  C({ id: 'c-anora', work: 'anora', target: '1080p-sdr', kind: 'derive', priority: 'rederive', settingsVersion: 3, needs: 'dv5-sdr',
    source: { name: 'upgraded 2160p HDR file', spec: 'HEVC 10-bit · Dolby Vision Profile 5 · WEB-DL · 16.2 GB' }, estSize: '5.6 GB',
    state: { is: 'waiting', reason: 'no-capable-node', cause: 'dv5-sdr', since: '2026-10-08T12:15:00' } }),
  C({ id: 'c-tdk', work: 'the-dark-knight', target: '1080p-sdr', kind: 'derive', priority: 'new', settingsVersion: 3,
    source: { name: '2160p HDR file', spec: 'HEVC 10-bit · HDR10 · 63.7 GB' }, estSize: '9.4 GB',
    state: { is: 'failed', attempts: [
      { node: 'forge', at: '2026-10-08T17:02:00', error: 'Output check: output runs 1:58:12, source runs 2:32:20. Truncated.' },
      { node: 'forge', at: '2026-10-08T17:31:00', error: 'Output check: output runs 1:58:12, source runs 2:32:20. Truncated.' },
      { node: 'atlas', at: '2026-10-08T19:06:00', error: '[hevc @ 0x7f3c] Invalid NAL unit size (2911 > 1024). Error while decoding stream #0:0: Invalid data found when processing input at 01:58:12' },
    ], verdict: 'The source file is damaged at 1:58:12. Both Nodes stop at the same frame.' } }),
];

// Backfill: lowest priority, placed after everything above. Shown as a group because there are many.
// Remake of Cinema's 1080p SDR after its Conversion settings went to v3, plus the TV Standard download steps gaining a stereo track.
export const backfill = [
  { id: 'bf-1080p-v3', label: '1080p SDR remake, settings v3', target: '1080p-sdr', reason: 'Settings changed on Oct 7: stereo AAC added. You picked 18 of 46 files to remake.',
    works: ['dune', 'br2049', 'arrival', 'fury-road', 'inception', 'parasite', 'tenet', 'no-time-to-die', 'avatar-2', 'spider-verse-2', 'barbie', 'flower-moon', 'poor-things', 'godzilla-minus-one', 'furiosa', 'alien-romulus', 'wicked', 'gladiator-2'],
    nodes: { forge: 12, atlas: 6 }, estHours: 7.5, estGB: 162 },
  { id: 'bf-tv-v2', label: 'TV Standard stereo track, settings v2', target: '1080p', reason: 'Download steps gained a loudness-normalized stereo track on Oct 8.',
    items: [
      { work: 'severance', season: 1, episodes: 9 },
      { work: 'severance', season: 2, episodes: 7 },
      { work: 'bcs', season: 6, episodes: 13 },
    ],
    nodes: { forge: 29 }, estHours: 1.4, estGB: 118 },
];

export const today = { finished: 11, failed: 1, hoursQueued: 9.6 };

// ───────────────────────────── Backfill dialog
// Cinema gains 720p mobile, derived from 1080p SDR. The dialog opens on save in the Profile editor.
export const backfillAsk = {
  profile: 'cinema',
  addedTarget: '720p-mobile',
  contentType: 'movie',
  destination: { kind: 'create', name: 'Cinema (without 720p mobile)' },
  existingMatch: null,
  perWork: { minutes: 9, gb: 1.6 },
  nodes: [
    { node: 'forge', share: 0.78, note: 'NVENC H.264' },
    { node: 'atlas', share: 0.22, note: '01:00–07:00 only' },
    { node: 'studio', share: 0, note: 'offline since 18:02' },
  ],
  disk: { root: '/Volumes/Vault/Media', freeTB: 14.8 },
  noCapable: [{ work: 'eeaao', cause: 'dv5-sdr' }, { work: 'civil-war', cause: 'dv5-sdr' }, { work: 'anora', cause: 'dv5-sdr' }],
  // Picking aids. playedOnMobile comes from the Playback report: Works played on a phone or tablet in the last 30 days.
  playedOnMobile: ['dune-2', 'oppenheimer', 'interstellar', 'barbie', 'the-batman', 'spider-verse-2', 'top-gun-maverick', 'inception', 'past-lives', 'holdovers', 'wicked', 'conclave'],
  requestedBy: { 'barbie': 'maya', 'wicked': 'priya', 'conclave': 'sam', 'past-lives': 'maya', 'anora': 'sam' },
  addedRecently: ['dune-2', 'oppenheimer', 'interstellar', 'civil-war', 'anora', 'conclave', 'wicked', 'gladiator-2', 'the-substance', 'alien-romulus'],
};

export const members = { maya: 'Maya Chen', sam: 'Sam Okafor', priya: 'Priya Raman' };

// The 48 Works in Cinema, in library order.
export const cinemaWorks = Object.values(works).filter((w) => w.type === 'movie').map((w) => w.id);

// Estimate for backfilling n Works at the lowest priority, after the 9.6 h already queued.
// forge runs 3 at a time at ≈ 9 min each. atlas joins past 20 Works at ≈ 34 min each, only 01:00–07:00 (its first window opens in 3 h 20 min).
export function estimate(n) {
  const atlas = n > 20 ? Math.round((n - 20) * 0.4) : 0, forge = n - atlas;
  const forgeH = (forge * 9) / 3 / 60;
  const atlasH = atlas ? 3.33 + Math.floor((atlas * 34) / 60 / 6) * 24 + ((atlas * 34) / 60) % 6 : 0;
  return { works: n, hours: today.hoursQueued + Math.max(forgeH, atlasH - today.hoursQueued), gb: Math.round(n * backfillAsk.perWork.gb), byNode: { forge, atlas } };
}
