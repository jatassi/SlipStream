// PROTOTYPE fixture for the Tracker detail page. Extends the torrent client fixture with Indexers, Tracker routing per Node,
// hit-and-run history and a few more torrents. Trackers, sites, people and numbers are invented.

import * as base from '../torrent-client/data.js';

export * from '../torrent-client/data.js';

const T = Object.fromEntries(base.trackers.map((t) => [t.id, t]));

// Per-tracker extras live on the same objects, so the base helpers (goalFor, hnr) keep working.
Object.assign(T.aurora, {
  site: 'aurora.example', seedingTB: 2.1, grabs30: 26, since: '2026-10-02T10:00:00',
  allowList: ['qBittorrent 4.3–5.0', 'Transmission 3.00–4.0.6', 'Deluge 2.x', 'rTorrent 0.9.8'],
  allowRequest: 'Asked Aurora staff to add -SS2000- on Oct 2. Pending.',
  routes: { 'seedbox-ams': 'qbit-ams' },
  hnrModel: 'ratio',
});
Object.assign(T.beacon, {
  site: 'beacon.example', seedingTB: 18.4, grabs30: 212,
  routes: { 'seedbox-ams': 'builtin-ams', atlas: 'builtin-ams' },
  hnrModel: 'window',
});
Object.assign(T.cinder, {
  site: 'cinder.example', seedingTB: 9.7, grabs30: 64,
  routes: { 'seedbox-ams': 'builtin-ams', atlas: 'builtin-atlas' },
  hnrModel: 'window',
});
Object.assign(T.public, {
  site: '', seedingTB: 0.9, grabs30: 31,
  routes: { atlas: 'builtin-atlas' },
  hnrModel: 'none',
});

// Default Tracker routing: the built-in client on the Indexer's home Node.
export const defaultRoute = (nodeId) => base.clients.find((c) => c.node === nodeId && c.kind === 'built-in').id;

// ───────────────────────────── Indexers: one configured way to search Trackers
// kind: catalog (Cardigann definition) | native | torznab (generic, can reach many Trackers) | rss
// cred: apikey | passkey | session | none. API key and passkey Indexers fail over; login sessions wait unless allowed.
export const indexers = [
  {
    id: 'ix-beacon', name: 'Beacon', kind: 'catalog', def: 'beacon (catalog definition v14)', trackers: ['beacon'],
    cred: 'apikey', credText: 'API key ····7f2a', home: 'seedbox-ams', failoverTo: 'atlas', graceMin: 10,
    health: 'ok', lastOk: '2026-10-08T21:38:00', ms: 340,
    limits: { queries: { used: 412, max: 1000, per: 'day' }, grabs: { used: 38, max: 100, per: 'day' } },
    rss: { on: true, every: 15 }, auto: { beacon: true }, interactive: true, priority: 10,
  },
  {
    id: 'ix-prowlarr', name: 'Prowlarr', kind: 'torznab', def: 'http://prowlarr.lan:9696 · Torznab', trackers: ['beacon', 'cinder', 'public'],
    cred: 'apikey', credText: 'API key ····c19e', home: 'atlas', failoverTo: 'seedbox-ams', graceMin: 10,
    health: 'ok', lastOk: '2026-10-08T21:35:00', ms: 820,
    limits: { queries: { used: 96, max: null, per: 'day' }, grabs: { used: 4, max: null, per: 'day' } },
    rss: { on: true, every: 30 }, auto: { beacon: false, cinder: true, public: true }, interactive: true, priority: 25,
    note: 'Reaches Beacon, Cinder and 6 public sites through one feed',
  },
  {
    id: 'ix-aurora', name: 'Aurora', kind: 'catalog', def: 'aurora (catalog definition v9)', trackers: ['aurora'],
    cred: 'session', credText: 'Login session · cookie earned on seedbox-ams Oct 1, expires Nov 1', cookieNode: 'seedbox-ams', home: 'seedbox-ams', allowFailover: false,
    solver: { node: 'seedbox-ams', name: 'FlareSolverr', status: 'ok' },
    health: 'ok', lastOk: '2026-10-08T21:31:00', ms: 1260,
    limits: { queries: { used: 188, max: 500, per: 'day' }, grabs: { used: 9, max: 50, per: 'day' } },
    rss: { on: true, every: 20 }, auto: { aurora: true }, interactive: true, priority: 10,
  },
  {
    id: 'ix-cinder', name: 'Cinder', kind: 'native', def: 'UNIT3D API · cinder.example/api', trackers: ['cinder'],
    cred: 'apikey', credText: 'API token ····a03d', home: 'seedbox-ams', failoverTo: 'atlas', graceMin: 10,
    health: 'warn', healthMsg: 'Rate limited: 3 requests answered 429 in the last hour. Backing off until 21:55.', lastOk: '2026-10-08T21:20:00', ms: 410,
    limits: { queries: { used: 57, max: 60, per: 'hour' }, grabs: { used: 6, max: 25, per: 'day' } },
    rss: { on: true, every: 15 }, auto: { cinder: true }, interactive: true, priority: 10,
  },
  {
    id: 'ix-lantern', name: 'Lantern', kind: 'catalog', def: 'lantern (catalog definition v31)', trackers: ['public'],
    cred: 'none', credText: 'No account', home: 'atlas', failoverTo: 'seedbox-ams', graceMin: 10,
    solver: { node: 'atlas', name: 'FlareSolverr', status: 'ok' },
    health: 'ok', lastOk: '2026-10-08T21:36:00', ms: 2100,
    limits: { queries: { used: 140, max: null, per: 'day' }, grabs: { used: 3, max: null, per: 'day' } },
    rss: { on: false, every: 60 }, auto: { public: true }, interactive: true, priority: 40,
  },
];
export const indexersFor = (tid) => indexers.filter((i) => i.trackers.includes(tid));
export const KIND_LABEL = { catalog: 'Catalog definition', native: 'Native API', torznab: 'Torznab feed', rss: 'RSS feed' };
export const CRED_LABEL = { apikey: 'API key', passkey: 'Passkey', session: 'Login session', none: 'No credentials' };

// ───────────────────────────── Hits and runs: what the tracker recorded. `days` is the last 30 days, oldest first: [cleared, hits].
const days = (seed, hitDays = []) => Array.from({ length: 30 }, (_, i) => [Math.max(0, Math.round(((Math.sin(i * 1.7 + seed) + 1) * seed) % 4)), hitDays.includes(i) ? 1 : 0]);
export const hnrHistory = {
  beacon: {
    cleared30: 41, hits30: 0, hitsAll: 1, days: days(3),
    events: [
      { at: '2026-10-08T09:10:00', kind: 'cleared', title: 'Severance S02E06', detail: 'Seeded 72 h on day 4' },
      { at: '2026-10-07T22:40:00', kind: 'cleared', title: 'Better Call Saul Season 6', detail: 'Seeded 72 h on day 3' },
      { at: '2026-10-05T14:00:00', kind: 'cleared', title: 'The Last of Us S02E02', detail: 'Seeded 72 h on day 3' },
      { at: '2026-08-14T18:20:00', kind: 'hit', title: 'Narcos Season 2', detail: 'Removed by hand at 51 h of 72 h. Beacon recorded a hit and run; it expires Nov 14.' },
    ],
  },
  cinder: {
    cleared30: 18, hits30: 1, hitsAll: 1, days: days(2, [17]),
    events: [
      { at: '2026-10-06T08:00:00', kind: 'cleared', title: 'Narcos Season 1', detail: 'Seeded 7 days' },
      { at: '2026-10-02T11:30:00', kind: 'cleared', title: 'Game of Thrones Season 1', detail: 'Reached ratio 1.0' },
      { at: '2026-09-25T19:05:00', kind: 'hit', title: 'The Prestige 1080p', detail: 'Removed by hand at 4 d of 7 d, ratio 0.3. Cinder recorded a hit and run.' },
    ],
  },
  aurora: { account: { ratio: 1.7, min: 0.6, buffer: '1.4 TB' } },
  public: null,
};

// ───────────────────────────── more torrents, so each tracker has something in every obligation group
const ep = (season, ...eps) => eps.map((episode) => ({ season, episode }));
const ok = (last) => ({ status: 'ok', last, next: '2026-10-08T22:10:00' });
const timeout = { status: 'error', msg: 'Timed out after 30 s · retrying', last: '2026-10-08T21:12:40', next: '2026-10-08T21:42:40' };
const extra = [
  { id: 't-sev-205', work: 'severance', coverage: { items: ep(2, 5) }, target: '1080p', client: 'builtin-ams', tracker: 'beacon', state: 'seeding', progress: 1, down: 0, up: 1.2, sizeGB: 3.8, downloadedGB: 3.8, uploadedGB: 4.4, ratio: 1.16, seededH: 141, completedAt: '2026-10-02T23:00:00', seeds: 288, leech: 14, peers: 4, addedAt: '2026-10-02T22:10:00', release: 'Severance.S02E05.1080p.ATVP.WEB-DL.DDP5.1.Atmos.H.264-FLUX', announce: ok('2026-10-08T21:32:00') },
  { id: 't-tlou-202', work: 'tlou', coverage: { items: ep(2, 2) }, target: '1080p', client: 'builtin-ams', tracker: 'beacon', state: 'seeding', progress: 1, down: 0, up: 0.6, sizeGB: 5.0, downloadedGB: 5.0, uploadedGB: 4.1, ratio: 0.82, seededH: 120, completedAt: '2026-10-03T21:00:00', seeds: 351, leech: 20, peers: 2, addedAt: '2026-10-03T20:20:00', release: 'The.Last.of.Us.S02E02.1080p.MAX.WEB-DL.DDP5.1.Atmos.H.264-FLUX', announce: ok('2026-10-08T21:28:00') },
  { id: 't-eeaao', work: 'eeaao', coverage: { items: [{ edition: 'Standard' }] }, target: '2160p HDR', client: 'builtin-ams', tracker: 'beacon', state: 'met', progress: 1, down: 0, up: 0, sizeGB: 24.6, downloadedGB: 24.6, uploadedGB: 57.0, ratio: 2.32, seededH: 302, completedAt: '2026-09-26T08:00:00', seeds: 120, leech: 1, peers: 0, addedAt: '2026-09-26T07:00:00', release: 'Everything.Everywhere.All.at.Once.2022.2160p.WEB-DL.DDP5.1.HDR.H.265-FLUX', removeAt: '2026-10-08T22:05:00', announce: { status: 'ok', last: '2026-10-08T21:30:00', next: null } },
  { id: 't-narcos-s1', work: 'narcos', coverage: { items: ep(1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10), whole: true }, target: '1080p', client: 'builtin-ams', tracker: 'cinder', state: 'seeding', progress: 1, down: 0, up: 0.4, sizeGB: 26.0, downloadedGB: 26.0, uploadedGB: 10.4, ratio: 0.4, seededH: 196, completedAt: '2026-09-28T12:00:00', seeds: 77, leech: 2, peers: 1, addedAt: '2026-09-28T10:00:00', release: 'Narcos.S01.1080p.NF.WEB-DL.DDP5.1.x264-NTb', announce: timeout },
  { id: 't-boys-s3', work: 'the-boys', coverage: { items: ep(3, 1, 2, 3, 4, 5, 6, 7, 8), whole: true }, target: '1080p', client: 'builtin-ams', tracker: 'cinder', state: 'seeding', progress: 1, down: 0, up: 0.2, sizeGB: 31.2, downloadedGB: 31.2, uploadedGB: 4.7, ratio: 0.15, seededH: 52, completedAt: '2026-10-06T13:00:00', seeds: 40, leech: 3, peers: 1, addedAt: '2026-10-06T10:00:00', release: 'The.Boys.S03.1080p.AMZN.WEB-DL.DDP5.1.H.264-NTb', announce: timeout },
  { id: 't-oppen-q', work: 'top-gun-maverick', coverage: { items: [{ edition: 'Standard' }] }, target: '1080p', client: 'qbit-ams', tracker: 'aurora', state: 'seeding', progress: 1, down: 0, up: 1.9, sizeGB: 12.4, downloadedGB: 12.4, uploadedGB: 30.1, ratio: 2.43, seededH: 410, completedAt: '2026-09-20T09:00:00', seeds: 62, leech: 4, peers: 2, addedAt: '2026-09-20T08:00:00', release: 'Top.Gun.Maverick.2022.1080p.BluRay.DDP5.1.x264-HiFi', announce: ok('2026-10-08T21:25:00') },
  { id: 't-int-pub', work: 'interstellar', coverage: { items: [{ edition: 'Standard' }] }, target: '1080p', client: 'builtin-atlas', tracker: 'public', state: 'met', progress: 1, down: 0, up: 0, sizeGB: 14.1, downloadedGB: 14.1, uploadedGB: 14.8, ratio: 1.05, seededH: 88, completedAt: '2026-10-05T01:00:00', seeds: 900, leech: 30, peers: 0, addedAt: '2026-10-04T23:00:00', release: 'Interstellar.2014.1080p.BluRay.x264.DTS-HD.MA.5.1', removeAt: '2026-10-08T21:55:00', announce: { status: 'ok', last: '2026-10-08T21:30:00', next: null } },
];
export const torrents = [...base.torrents, ...extra];
export const moreFor = { aurora: 35, beacon: 250, cinder: 143, public: 54 };
