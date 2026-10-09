// PROTOTYPE fixture. Library state, people, Nodes and numbers are invented. Artwork is v1's local cache (see run.sh).

const movie = (id) => ({ poster: `art/movie/${id}_poster.jpg`, backdrop: `art/movie/${id}_backdrop.jpg`, logo: `art/movie/${id}_logo.png` });
const series = (id) => ({ poster: `art/series/${id}_poster.jpg`, backdrop: `art/series/${id}_backdrop.jpg`, logo: `art/series/${id}_logo.png` });

export const now = new Date('2026-10-08T21:40:00');

export const admin = { name: 'Jackson', email: 'jackson@example.com', initials: 'JA' };

export const members = [
  { id: 'maya', name: 'Maya Chen', initials: 'MC' },
  { id: 'sam', name: 'Sam Okafor', initials: 'SO' },
  { id: 'priya', name: 'Priya Raman', initials: 'PR' },
];

export const targets = [
  { id: '2160p-hdr', name: '2160p HDR', sourcing: 'downloaded' },
  { id: '1080p-sdr', name: '1080p SDR', sourcing: 'derived', from: '2160p-hdr' },
  { id: '1080p', name: '1080p', sourcing: 'downloaded' },
];

export const profiles = [
  { id: 'cinema', name: 'Cinema', targets: ['2160p-hdr', '1080p-sdr'] },
  { id: 'tv', name: 'TV Standard', targets: ['1080p'] },
];

export const nodes = [
  { id: 'atlas', name: 'atlas', kind: 'Controller', where: 'Home · LAN', status: 'online', cpu: 18, roles: ['Storage', 'Metadata', 'Media Server link', 'Subtitles', 'Notifying', 'Serving'], detail: 'Library root /Volumes/Vault/Media' },
  { id: 'forge', name: 'forge', kind: 'Worker', where: 'Home · LAN', status: 'online', cpu: 71, gpu: 'RTX 4070 · NVENC proven', roles: ['Converting'], detail: '2 Conversions running' },
  { id: 'seedbox-ams', name: 'seedbox-ams', kind: 'Worker', where: 'Amsterdam · remote', status: 'online', cpu: 9, roles: ['Searching', 'Downloading'], detail: 'Seeding 412 torrents · ratio 2.4' },
  { id: 'relay-fra', name: 'relay-fra', kind: 'Worker', where: 'Frankfurt · VPS', status: 'degraded', cpu: 3, roles: ['Relaying', 'Serving'], detail: 'Plex remote access unconfirmed' },
];

export const works = {
  'dune-2': {
    id: 'dune-2', type: 'movie', title: 'Dune: Part Two', year: 2024, runtime: '2h 46m', rating: 'PG-13', genres: ['Science Fiction', 'Adventure'],
    overview: 'Paul Atreides unites with Chani and the Fremen while seeking revenge against the conspirators who destroyed his family.',
    art: movie(693134), profile: 'cinema', editions: ['Standard'], monitored: true, addedBy: null,
    targets: [
      { target: '2160p-hdr', state: 'met', file: 'Dune.Part.Two.2024.2160p.UHD.BluRay.REMUX.DV.HDR10.TrueHD.Atmos.7.1-FraMeSToR.mkv', size: '74.6 GB', video: 'HEVC · Dolby Vision P7 + HDR10', audio: 'TrueHD Atmos 7.1', score: 2350, cutoff: true, seeding: 'Seeding on seedbox-ams · ratio 1.8' },
      { target: '1080p-sdr', state: 'converting', job: 'j-dune-conv', from: '2160p HDR file', via: 'forge · NVENC HEVC · tone map HDR10 → SDR', estSize: '9.8 GB' },
    ],
    subtitles: [{ lang: 'English', note: 'synced · −320 ms' }, { lang: 'Spanish', note: 'synced · +40 ms' }],
    history: [
      { at: '2026-10-06T21:04:00', kind: 'grabbed', text: 'Grabbed for 2160p HDR · score 2350 · from Aurora', node: 'seedbox-ams' },
      { at: '2026-10-06T23:15:00', kind: 'downloaded', text: 'Downloaded 74.6 GB in 2h 11m', node: 'seedbox-ams' },
      { at: '2026-10-06T23:26:00', kind: 'transfer', text: 'Transfer to atlas · 11m · BLAKE3 verified', node: 'atlas' },
      { at: '2026-10-06T23:27:00', kind: 'imported', text: 'Probed and imported · meets 2160p HDR', node: 'atlas' },
      { at: '2026-10-08T21:12:00', kind: 'conversion', text: 'Conversion started for 1080p SDR', node: 'forge' },
    ],
  },
  'oppenheimer': { id: 'oppenheimer', type: 'movie', title: 'Oppenheimer', year: 2023, art: movie(872585), profile: 'cinema' },
  'interstellar': { id: 'interstellar', type: 'movie', title: 'Interstellar', year: 2014, art: movie(157336), profile: 'cinema' },
  'top-gun-maverick': { id: 'top-gun-maverick', type: 'movie', title: 'Top Gun: Maverick', year: 2022, art: movie(361743), profile: 'cinema' },
  'eeaao': { id: 'eeaao', type: 'movie', title: 'Everything Everywhere All at Once', year: 2022, art: movie(545611), profile: 'cinema' },
  'the-batman': { id: 'the-batman', type: 'movie', title: 'The Batman', year: 2022, art: movie(414906), profile: 'cinema' },
  'black-panther': { id: 'black-panther', type: 'movie', title: 'Black Panther', year: 2018, art: movie(284054), profile: 'cinema' },
  'the-matrix': { id: 'the-matrix', type: 'movie', title: 'The Matrix', year: 1999, art: movie(603), profile: 'cinema' },
  'the-dark-knight': { id: 'the-dark-knight', type: 'movie', title: 'The Dark Knight', year: 2008, art: movie(155), profile: 'cinema' },
  'the-prestige': { id: 'the-prestige', type: 'movie', title: 'The Prestige', year: 2006, art: movie(1124), profile: 'cinema' },
  'severance': { id: 'severance', type: 'series', title: 'Severance', year: 2022, network: 'Apple TV+', art: series(95396), profile: 'tv' },
  'the-boys': { id: 'the-boys', type: 'series', title: 'The Boys', year: 2019, network: 'Prime Video', art: series(76479), profile: 'tv' },
  'hotd': { id: 'hotd', type: 'series', title: 'House of the Dragon', year: 2022, network: 'HBO', art: series(94997), profile: 'tv' },
  'tlou': { id: 'tlou', type: 'series', title: 'The Last of Us', year: 2023, network: 'HBO', art: series(100088), profile: 'tv' },
  'bcs': { id: 'bcs', type: 'series', title: 'Better Call Saul', year: 2015, network: 'AMC', art: series(60059), profile: 'tv' },
  'got': { id: 'got', type: 'series', title: 'Game of Thrones', year: 2011, network: 'HBO', art: series(1399), profile: 'tv' },
  'narcos': { id: 'narcos', type: 'series', title: 'Narcos', year: 2015, network: 'Netflix', art: series(63351), profile: 'tv' },
};

// Stages a job moves through. A job sits at one stage; `progress` is 0..1 within it. live.js advances running jobs.
export const stages = ['Searching', 'Downloading', 'Transfer', 'Import', 'Conversion', 'Available'];

export const jobs = [
  { id: 'j-sev-dl', work: 'severance', item: 'S02E07 · Chikhai Bardo', target: '1080p', stage: 'Downloading', node: 'seedbox-ams', status: 'running', progress: 0.71, rate: 24.6, unit: 'MB/s', etaMin: 1, size: '3.9 GB', release: 'Severance.S02E07.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', score: 1850, requestedBy: 'maya' },
  { id: 'j-dune-conv', work: 'dune-2', item: 'Standard', target: '1080p SDR', stage: 'Conversion', node: 'forge', status: 'running', progress: 0.62, rate: 148, unit: 'fps', etaMin: 6, size: '≈ 9.8 GB', release: 'from 2160p HDR · NVENC HEVC · tone map HDR10 → SDR' },
  { id: 'j-tlou-xfer', work: 'tlou', item: 'S02E03 · The Path', target: '1080p', stage: 'Transfer', node: 'seedbox-ams → atlas', status: 'running', progress: 0.38, rate: 112, unit: 'MB/s', etaMin: 1, size: '5.2 GB', release: '4 connections · resumable chunks' },
  { id: 'j-opp-dl', work: 'oppenheimer', item: 'Standard', target: '2160p HDR', stage: 'Downloading', node: 'seedbox-ams', status: 'running', progress: 0.23, rate: 8.1, unit: 'MB/s', etaMin: 41, size: '78.4 GB', release: 'Oppenheimer.2023.2160p.UHD.BluRay.REMUX.HDR10.DTS-HD.MA.5.1-EPSiLON', score: 2100 },
  { id: 'j-int-imp', work: 'interstellar', item: 'Standard', target: '2160p HDR', stage: 'Import', node: 'atlas', status: 'running', progress: 0.9, rate: null, unit: null, etaMin: 0, size: '61.0 GB', release: 'Probed · renaming into library' },
  { id: 'j-bat-fail', work: 'the-batman', item: 'Standard', target: '2160p HDR', stage: 'Import', node: 'atlas', status: 'failed', progress: 1, cause: 'Probe found SDR video; the release said HDR10.', recovery: 'Blocklisted the release and searching again. The file is kept for 1080p SDR, which it meets.', release: 'The.Batman.2022.2160p.WEB-DL.HDR10.DDP5.1.Atmos-PLAYWEB' },
  { id: 'j-eeaao-search', work: 'eeaao', item: 'Standard', target: '2160p HDR', stage: 'Searching', node: 'seedbox-ams', status: 'running', progress: 0.4, rate: null, unit: null, etaMin: null, release: '14 indexers · 31 releases so far', requestedBy: 'sam' },
];

export const health = [
  { level: 'error', stage: 'Import', title: 'The Batman · 2160p HDR release was misdescribed', body: 'Probe found SDR video; the release said HDR10. Blocklisted it and searching again.', work: 'the-batman', at: '2026-10-08T21:31:00' },
  { level: 'warn', stage: 'Relaying', title: 'Plex remote access unconfirmed on relay-fra', body: 'Plex has not confirmed an inbound connection since 19:20. Remote players may fall back to Plex Relay.', node: 'relay-fra', at: '2026-10-08T19:20:00' },
];

export const today = { imported: 14, conversions: 3, grabbed: 19, failed: 1, downloadRate: 32.7, uploadRate: 38.2 };

export const storage = { root: '/Volumes/Vault/Media', usedTB: 41.2, totalTB: 56 };

export const library = { movies: 812, series: 96, episodes: 7341 };

export const upNext = [
  { work: 'severance', item: 'S02E08 · Sweet Vitriol', at: '2026-10-10T03:00:00', label: 'Fri' },
  { work: 'tlou', item: 'S02E04 · Day One', at: '2026-10-12T21:00:00', label: 'Sun' },
  { work: 'the-boys', item: 'S05E01', at: '2026-10-15T03:00:00', label: 'Wed' },
  { work: 'hotd', item: 'S03E01', at: '2026-10-19T21:00:00', label: 'Sun' },
];

export const recentlyAdded = ['interstellar', 'top-gun-maverick', 'dune-2', 'bcs', 'the-prestige', 'narcos'];

export const requests = [
  { id: 'r1', work: 'eeaao', by: 'sam', state: 'approved', at: '2026-10-08T20:58:00', note: 'Auto-approved · searching' },
  { id: 'r2', work: 'hotd', by: 'priya', state: 'pending', at: '2026-10-08T18:12:00', note: 'Season 2' },
  { id: 'r3', work: 'black-panther', by: 'maya', state: 'pending', at: '2026-10-08T11:40:00', note: '' },
  { id: 'r4', work: 'severance', by: 'maya', state: 'approved', at: '2026-10-01T09:00:00', note: 'Seasons 1–2 · monitoring new episodes' },
];

// What a Member sees. Outcomes only: requested, coming, ready. No Nodes, Targets, stages or releases.
export const portal = {
  member: { name: 'Maya', initials: 'MC', email: 'maya@example.com' },
  mine: [
    { work: 'severance', state: 'coming', line: 'Episode 7 is almost here', detail: '6 of 10 episodes ready · new episode Friday', progress: 0.71, job: 'j-sev-dl' },
    { work: 'black-panther', state: 'requested', line: 'Waiting for approval', detail: 'Requested this morning' },
    { work: 'tlou', state: 'ready', line: 'Season 2 episode 2 is ready', detail: 'Watch on Plex' },
    { work: 'interstellar', state: 'ready', line: 'Ready to watch', detail: 'Added tonight · Watch on Plex' },
  ],
  newInLibrary: ['interstellar', 'top-gun-maverick', 'dune-2', 'bcs', 'the-prestige', 'narcos'],
  popular: ['the-batman', 'oppenheimer', 'the-matrix', 'the-dark-knight', 'got', 'the-boys'],
  inbox: [
    { at: '2026-10-08T21:10:00', text: 'Interstellar is ready to watch.' },
    { at: '2026-10-07T22:02:00', text: 'The Last of Us S02E02 is ready to watch.' },
  ],
};

// One Work page for each surface. Admin sees Dune: Part Two in full; the portal shows the same Work as a Member does.
export const portalWork = {
  work: 'dune-2', state: 'ready', line: 'Ready to watch on Plex', detail: 'In 4K HDR · English and Spanish subtitles',
};
