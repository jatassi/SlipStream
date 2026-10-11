// PROTOTYPE fixture for Subtitle Provider setup and subtitle screens (#69). Library state and numbers are invented.
// Provider facts (key pages, steps, limits, error replies) were looked up on 2026-10-11; see README.md for sources.

export const now = new Date('2026-10-08T21:40:00Z');

const movieArt = (id) => ({ poster: `art/movie/${id}_poster.jpg`, backdrop: `art/movie/${id}_backdrop.jpg`, logo: `art/movie/${id}_logo.png` });
const seriesArt = (id) => ({ poster: `art/series/${id}_poster.jpg`, backdrop: `art/series/${id}_backdrop.jpg`, logo: `art/series/${id}_logo.png` });

export const admin = { name: 'Jackson', initials: 'JA' };

// One entry per Subtitle Provider SlipStream ships. `check` is what pasting a key runs; `reject` is the provider's
// real reply to a bad key, captured live, so the card can quote it.
export const providers = {
  subdl: {
    id: 'subdl', name: 'SubDL', site: 'subdl.com', covers: 'Movies and TV', needsKey: true,
    signup: 'https://subdl.com/', keyPage: 'https://subdl.com/panel/api', terms: 'https://subdl.com/api-doc',
    steps: [
      'Sign in to subdl.com, or create a free account.',
      'Open the API page in your panel.',
      'Generate a key and copy it.',
      'Paste it here. SlipStream checks it right away.',
    ],
    never: 'SlipStream never asks for your SubDL password. SubDL’s terms forbid sharing it, and the key is all it needs.',
    check: 'Asks SubDL who the key belongs to (GET /api/v1/me). It costs no download.',
    reject: { status: 403, body: 'Not Authorized' },
    allowance: { unit: 'downloads', limit: 50, resets: 'midnight UTC', note: 'Free key. SubDL reports what is left after each download.' },
  },
  subsource: {
    id: 'subsource', name: 'SubSource', site: 'subsource.net', covers: 'Movies and TV', needsKey: true,
    signup: 'https://subsource.net/', keyPage: 'https://subsource.net/', terms: 'https://subsource.net/api-docs',
    steps: [
      'Sign in to subsource.net, or create a free account.',
      'Open My Profile from your account menu.',
      'Generate an API key and copy it. You can regenerate it any time.',
      'Paste it here. SlipStream checks it right away.',
    ],
    never: 'SlipStream only needs the key. It never sees your SubSource password.',
    check: 'Runs one search (60 a minute are allowed). Rejected keys answer 401.',
    reject: { status: 401, body: 'The provided API key is invalid or inactive' },
    allowance: { unit: 'requests', limit: 7200, resets: 'rolling 24 hours', perMinute: 60, note: 'Per key: 60 a minute, 1,800 an hour, 7,200 a day.' },
  },
  gestdown: {
    id: 'gestdown', name: 'Gestdown', site: 'gestdown.info', covers: 'TV only', needsKey: false,
    signup: null, keyPage: null, terms: 'https://api.gestdown.info',
    steps: [],
    never: 'No account and no key. Gestdown serves Addic7ed’s TV subtitles.',
    check: 'Nothing to check. SlipStream watches its replies for outages.',
    reject: null,
    allowance: null,
  },
};

// Scenario switch in the bar. Each names every provider's live state.
// key: none | valid | invalid · state: ready | paused | off
export const scenarios = {
  fresh: {
    label: 'Fresh install',
    order: ['subdl', 'subsource', 'gestdown'],
    status: {
      subdl: { key: null, state: 'off' },
      subsource: { key: null, state: 'off' },
      gestdown: { key: null, state: 'ready', lastOk: '2 min ago' },
    },
  },
  running: {
    label: 'Keys added',
    order: ['subdl', 'subsource', 'gestdown'],
    status: {
      subdl: { key: 'sd_8f2c…41ab', state: 'ready', account: 'jackson', plan: 'Free', used: 31, lastOk: 'just now' },
      subsource: { key: 'ss_d02e…9c77', state: 'ready', account: 'jackson', used: 1184, lastOk: '4 min ago' },
      gestdown: { key: null, state: 'ready', lastOk: '2 min ago' },
    },
  },
  strained: {
    label: 'Quota out, key revoked',
    order: ['subdl', 'subsource', 'gestdown'],
    status: {
      subdl: { key: 'sd_8f2c…41ab', state: 'paused', account: 'jackson', plan: 'Free', used: 50, resumesAt: '00:00 UTC', resumesIn: '2 h 20 min', lastOk: '38 min ago' },
      subsource: { key: 'ss_d02e…9c77', state: 'invalid', since: '06:12', error: 'The provided API key is invalid or inactive' },
      gestdown: { key: null, state: 'ready', lastOk: '2 min ago' },
    },
  },
};

// The states one Wanted Language can be in for one library file. Every view draws from this table.
export const subStates = {
  final: { label: 'Final', icon: 'badge-check', tone: 'ok', means: 'Human-made and matched by hash or release group. Not searched again.' },
  present: { label: 'Present', icon: 'captions', tone: 'ok', means: 'A synced Sidecar. Searched again on schedule for a better match.' },
  embedded: { label: 'Embedded', icon: 'film', tone: 'ok', means: 'A text track inside the file meets it. Nothing is downloaded.' },
  missing: { label: 'Missing', icon: 'circle-dashed', tone: 'warn', means: 'No subtitle yet. Searched again on schedule.' },
  nosync: { label: 'No candidate passed sync', icon: 'audio-waveform', tone: 'warn', means: 'Providers had candidates, but none lined up with the Speech Map. Searched again later.' },
  notneeded: { label: 'Not needed', icon: 'volume-2', tone: 'quiet', means: 'Wanted only when no audio track is in this language, and one is.' },
};

export const languages = ['English', 'Spanish', 'French', 'German', 'Portuguese (BR)', 'Japanese', 'Korean', 'Italian', 'Dutch', 'Arabic'];

// A Wanted Language: language, full or forced only, hearing-impaired preference, and when it applies.
export const profiles = {
  cinema: {
    id: 'cinema', name: 'Cinema', contentType: 'Movies', targets: ['2160p HDR', '1080p SDR'], works: 214,
    wanted: [
      { id: 'w-en', lang: 'English', kind: 'full', hi: 'allow', when: 'always' },
      { id: 'w-en-f', lang: 'English', kind: 'forced', hi: 'avoid', when: 'always' },
      { id: 'w-es', lang: 'Spanish', kind: 'full', hi: 'avoid', when: 'always' },
      { id: 'w-fr', lang: 'French', kind: 'full', hi: 'avoid', when: 'noAudio' },
    ],
  },
  tv: {
    id: 'tv', name: 'TV Standard', contentType: 'Series', targets: ['1080p'], works: 61,
    wanted: [
      { id: 't-en', lang: 'English', kind: 'full', hi: 'prefer', when: 'always' },
      { id: 't-es', lang: 'Spanish', kind: 'full', hi: 'avoid', when: 'noAudio' },
    ],
  },
};

export const hiLabels = { avoid: 'Avoid HI', allow: 'HI allowed', prefer: 'Prefer HI' };
export const whenLabels = { always: 'Always', noAudio: 'Only without that audio' };
export const kindLabels = { full: 'Full', forced: 'Forced only' };

// Per library file, the state of each Wanted Language. `sub` is what's on disk when there is one.
export const works = {
  'dune-2': {
    id: 'dune-2', type: 'movie', title: 'Dune: Part Two', year: 2024, art: movieArt(693134), profile: 'cinema', edition: 'Standard',
    audio: ['English', 'French'],
    files: [
      {
        id: 'f-dune-4k', target: '2160p HDR', name: 'Dune.Part.Two.2024.2160p.UHD.BluRay.REMUX.DV.HDR10.TrueHD.Atmos.7.1-FraMeSToR.mkv', size: '74.6 GB', node: 'atlas',
        embedded: [{ lang: 'English', kind: 'full', codec: 'PGS', note: 'Image track. Plays as a fallback; doesn’t meet a want.' }, { lang: 'English', kind: 'forced', codec: 'SRT' }],
        wants: {
          'w-en': { state: 'final', sub: { provider: 'subdl', release: 'Dune.Part.Two.2024.2160p.UHD.BluRay.REMUX-FraMeSToR', file: 'Dune Part Two (2024) - 2160p HDR.en.srt', reasons: ['hash'], sync: 'skipped, hash match', by: 'human', at: '2026-10-06T23:31:00Z' } },
          'w-en-f': { state: 'embedded', track: 'Track 4 · SRT · English forced' },
          'w-es': { state: 'present', sub: { provider: 'subsource', release: 'Dune.Part.Two.2024.1080p.BluRay.x264-SPARKS', file: 'Dune Part Two (2024) - 2160p HDR.es.srt', reasons: ['source'], sync: '+40 ms, framerate unchanged', by: 'human', at: '2026-10-07T02:10:00Z' }, next: 'Upgrade search tonight at 03:00' },
          'w-fr': { state: 'notneeded', note: 'Wanted only without French audio. This file has a French TrueHD track.' },
        },
      },
      {
        id: 'f-dune-1080', target: '1080p SDR', name: 'Dune Part Two (2024) - 1080p SDR.mkv', size: '9.8 GB', node: 'atlas', derived: 'Converted from the 2160p HDR file. Its Sidecars were copied over.',
        embedded: [],
        wants: {
          'w-en': { state: 'present', sub: { provider: 'subdl', release: 'copied from 2160p HDR file', file: 'Dune Part Two (2024) - 1080p SDR.en.srt', reasons: ['copied'], sync: 'copied, same timing', by: 'human', at: '2026-10-08T21:30:00Z' } },
          'w-en-f': { state: 'missing', note: 'The forced track was inside the 2160p file. Conversion keeps only the main subtitle track.', next: 'Searching SubDL, SubSource' },
          'w-es': { state: 'nosync', tried: 3, note: '3 candidates tried. Best alignment 41%, the gate is 70%.', next: 'Retry tomorrow 03:00' },
          'w-fr': { state: 'notneeded', note: 'Wanted only without French audio. The French track was kept in Conversion.' },
        },
      },
    ],
  },
  severance: {
    id: 'severance', type: 'series', title: 'Severance', year: 2022, art: seriesArt(95396), profile: 'tv', season: 2,
    audio: ['English', 'Spanish'],
    episodes: [
      { n: 1, title: 'Hello, Ms. Cobel', wants: { 't-en': { state: 'final', sub: { provider: 'gestdown', release: 'Severance.S02E01.1080p.ATVP.WEB-DL-FLUX', reasons: ['group'], sync: '+0 ms', by: 'human' } }, 't-es': { state: 'notneeded' } } },
      { n: 2, title: 'Goodbye, Mrs. Selvig', wants: { 't-en': { state: 'final', sub: { provider: 'gestdown', release: 'Severance.S02E02.1080p.ATVP.WEB-DL-FLUX', reasons: ['group'], sync: '+0 ms', by: 'human' } }, 't-es': { state: 'notneeded' } } },
      { n: 3, title: 'Who Is Alive?', wants: { 't-en': { state: 'present', sub: { provider: 'gestdown', release: 'Severance.S02E03.720p.WEB.h264-ETHEL', reasons: [], sync: '−1.2 s', by: 'human' } }, 't-es': { state: 'notneeded' } } },
      { n: 4, title: 'Woe’s Hollow', wants: { 't-en': { state: 'embedded', track: 'Track 3 · SRT · English SDH' }, 't-es': { state: 'notneeded' } } },
      { n: 5, title: 'Trojan’s Horse', wants: { 't-en': { state: 'present', sub: { provider: 'subdl', release: 'Severance.S02E05.1080p.WEB.H264-SuccessfulCrab', reasons: ['source'], sync: '+220 ms', by: 'ai' } }, 't-es': { state: 'notneeded' } } },
      { n: 6, title: 'Attila', wants: { 't-en': { state: 'nosync', tried: 2, note: '2 candidates tried. Both drifted past the framerate limit.' }, 't-es': { state: 'notneeded' } } },
      { n: 7, title: 'Chikhai Bardo', wants: { 't-en': { state: 'missing', note: 'Imported 4 min ago. Searching now.' }, 't-es': { state: 'notneeded' } }, fresh: true },
    ],
  },
};

// Manual search results for one want on one file. Ranked by the rule in #20: hash, group, source and resolution,
// provider order, then provider downloads and rating. AI- and machine-translated always rank below human-made.
export const reasonLabels = {
  hash: { label: 'Hash match', tone: 'strong', means: 'Made for this exact file. Not re-synced.' },
  group: { label: 'Same group', tone: 'strong', means: 'Made for a release by FraMeSToR, the group that made this file.' },
  source: { label: 'Same source and resolution', tone: 'mid', means: 'BluRay 1080p, like this file.' },
  copied: { label: 'Copied from source file', tone: 'mid', means: 'Carried over from the file this one was converted from.' },
  ai: { label: 'AI-translated', tone: 'weak', means: 'Made by AI translation. Ranks below every human-made subtitle.' },
  machine: { label: 'Machine-translated', tone: 'weak', means: 'Ranks below every human-made subtitle.' },
  hi: { label: 'Hearing impaired', tone: 'info', means: 'Includes sound descriptions.' },
};

export const searchFor = { work: 'dune-2', file: 'f-dune-1080', want: 'w-es' };

export const candidates = [
  { id: 'c1', provider: 'subsource', release: 'Dune.Part.Two.2024.1080p.BluRay.x264-SPARKS', lang: 'Spanish', hi: false, reasons: ['source'], downloads: 18402, rating: 9.1, uploader: 'yyyyz', status: 'failed-sync', syncNote: 'Tried at 03:00. Alignment 41%.' },
  { id: 'c2', provider: 'subdl', release: 'Dune.Part.Two.2024.1080p.WEB-DL.DDP5.1.Atmos.H.264-FLUX', lang: 'Spanish (Latin America)', hi: false, reasons: [], downloads: 30981, rating: 8.7, uploader: 'Kikerojas', status: 'new' },
  { id: 'c4', provider: 'subdl', release: 'Dune.Part.Two.2024.1080p.AMZN.WEB-DL-NTb', lang: 'Spanish', hi: false, reasons: [], downloads: 2210, rating: 7.4, uploader: 'subs_es', status: 'failed-sync', syncNote: 'Tried at 03:00. Offset past the 60 s limit.' },
  { id: 'c3', provider: 'subsource', release: 'Dune.Part.Two.2024.2160p.WEB-DL.DV.HDR.H.265-FLUX', lang: 'Spanish', hi: true, reasons: ['hi'], downloads: 6120, rating: 8.2, uploader: 'Tamaraplus', status: 'new' },
  { id: 'c5', provider: 'subdl', release: 'Dune.Part.Two.2024.1080p.BluRay.x264-SPARKS', lang: 'Spanish', hi: false, reasons: ['source', 'ai'], downloads: 840, rating: null, uploader: 'SubDL AI', status: 'new' },
  { id: 'c6', provider: 'subsource', release: 'Dune.Part.Two.2024.HDCAM.x264', lang: 'Spanish', hi: false, reasons: [], downloads: 1902, rating: 3.1, uploader: 'anon', status: 'blocked', syncNote: 'Marked bad on Oct 7 by Jackson.' },
];
