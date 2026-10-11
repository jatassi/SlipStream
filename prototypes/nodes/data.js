// PROTOTYPE fixture for the Nodes screens. Builds on prototypes/ui-direction/data.js (same Works, jobs and four Nodes).
// Addresses, fingerprints, versions, probes and numbers are invented.

import * as base from '../ui-direction/data.js';

export const { works, now, admin } = base;
export const jobs = base.jobs;

export const CONTROLLER_VERSION = '2.4.0';

export const GROUPS = [
  { id: 'Acquire', roles: ['Searching', 'Downloading'] },
  { id: 'Store', roles: ['Storage'] },
  { id: 'Process', roles: ['Converting', 'Subtitles'] },
  { id: 'Connect', roles: ['Metadata', 'Media Server link', 'Notifying'] },
  { id: 'Reach', roles: ['Relaying', 'Serving'] },
];

// Guardrails are Role properties. always: never without a holder. reach: tested from the Node before saving.
// portable: a dead Node's jobs requeue to another holder after the grace period; otherwise the work is tied to the Node.
export const ROLES = {
  Searching: { always: true, portable: true, gated: false, blurb: 'Queries indexers for releases' },
  Downloading: { always: true, portable: false, gated: true, blurb: 'Runs torrents and seeds them to their goals' },
  Storage: { always: true, portable: false, gated: true, blurb: 'Holds library roots and imports into them' },
  Converting: { always: false, portable: true, gated: true, blurb: 'Runs Conversions it has proven by trial run' },
  Subtitles: { always: false, portable: true, gated: false, blurb: 'Finds and syncs subtitles' },
  Metadata: { always: true, portable: true, gated: false, blurb: 'Fetches from TMDB and TVDB' },
  'Media Server link': { always: false, portable: true, gated: false, reach: 'Plex at 192.168.1.20:32400', blurb: 'Tells the Media Server about new files' },
  Notifying: { always: false, portable: true, gated: false, reach: 'smtp.fastmail.com:465', blurb: 'Sends email and push notices' },
  Relaying: { always: false, portable: false, gated: true, blurb: 'Carries links for Nodes that cannot reach each other' },
  Serving: { always: false, portable: false, gated: false, blurb: 'Adds an entry point to the UI and portal' },
};
export const ROLE_NAMES = GROUPS.flatMap((g) => g.roles);
export const groupOf = (role) => GROUPS.find((g) => g.roles.includes(role)).id;

export const CONTROLLER_ONLY = 'library, decisions, schedules, API, sign-in';

// Converting is proven per codec, bit depth and dynamic range by a short trial run (ADR 0007).
const TRIALS = {
  atlas: { engine: 'VideoToolbox · M2 Pro', runs: [['HEVC', '10-bit', 'HDR10 to SDR', true, '96 fps'], ['HEVC', '10-bit', 'Dolby Vision P5 to SDR', true, '61 fps'], ['H.264', '8-bit', 'SDR', true, '220 fps']] },
  forge: { engine: 'NVENC · RTX 4070', runs: [['HEVC', '10-bit', 'HDR10 to SDR', true, '212 fps'], ['HEVC', '10-bit', 'Dolby Vision P8 to SDR', true, '148 fps'], ['AV1', '10-bit', 'HDR10', true, '190 fps'], ['H.264', '8-bit', 'SDR', true, '340 fps']] },
  'seedbox-ams': { engine: 'Software · 2 vCPU', runs: [['H.264', '8-bit', 'SDR', false, 'no GPU, 2 vCPU, 4 fps']] },
  'relay-fra': { engine: 'Software · 1 vCPU', runs: [['H.264', '8-bit', 'SDR', false, 'no GPU, 1 vCPU, 2 fps']] },
  cellar: { engine: 'Software · Ryzen R1600', runs: [['H.264', '8-bit', 'SDR', true, '31 fps'], ['HEVC', '10-bit', 'HDR10 to SDR', false, 'no OpenCL device for tone mapping']] },
};
export const trialsFor = (id) => TRIALS[id] || null;

// Probe verdicts that are not plain ok. ok notes fall back to OK_NOTE.
const PROBES = {
  atlas: { Relaying: ['blocked', 'No inbound port. atlas is behind CGNAT.'] },
  forge: { Relaying: ['blocked', 'No inbound port. forge is behind CGNAT.'], Downloading: ['weak', 'Home upload is 40 Mbit/s'] },
  'seedbox-ams': { Converting: ['blocked', 'Trial run failed: no GPU, 2 vCPU'], Storage: ['weak', '1.1 TB free, 940 GB of it seeding'] },
  'relay-fra': { Converting: ['blocked', 'Trial run failed: 1 vCPU, no GPU'], Downloading: ['weak', '18 GB free'], Storage: ['blocked', '18 GB free'] },
  cellar: { Relaying: ['blocked', 'No inbound port. cellar is behind CGNAT.'], Converting: ['weak', 'Proved H.264 SDR only. HDR tone mapping failed.'] },
};
const OK_NOTE = {
  Searching: 'Reached 14 of 14 indexers', Downloading: 'Ports open, disk writable', Storage: 'Disk writable', Converting: 'Trial runs passed',
  Subtitles: 'Reached OpenSubtitles', Metadata: 'Reached TMDB and TVDB', 'Media Server link': 'Tested before saving', Notifying: 'Tested before saving',
  Relaying: 'Inbound port 7878 open', Serving: 'Port 443 free',
};
export function probe(nodeId, role) {
  const p = (PROBES[nodeId] || {})[role];
  return p ? { v: p[0], note: p[1] } : { v: 'ok', note: OK_NOTE[role] };
}

// The answer a must-reach Role's test gives from each Node.
const REACH = {
  'seedbox-ams': { 'Media Server link': [false, 'No route to 192.168.1.20. Plex is on the home LAN.'] },
  'relay-fra': { 'Media Server link': [false, 'No route to 192.168.1.20. Plex is on the home LAN.'] },
};
export function reachTest(nodeId, role) {
  const r = (REACH[nodeId] || {})[role];
  const ms = { atlas: 2, forge: 3, cellar: 4, 'seedbox-ams': 31, 'relay-fra': 22 }[nodeId] || 9;
  return r ? { ok: false, note: r[1] } : { ok: true, note: `${ROLES[role].reach} answered in ${role === 'Notifying' ? ms + 18 : ms} ms` };
}

const roles = (...rs) => Object.fromEntries(rs.map((r) => [r, { state: 'active' }]));

function normalNodes() {
  return [
    {
      id: 'atlas', name: 'atlas', kind: 'Controller', where: 'Home · LAN', hw: 'Mac mini M2 Pro', os: 'macOS 15', arch: 'arm64', install: 'native',
      version: CONTROLLER_VERSION, state: 'online', cpu: 18, disk: '14.8 TB free of 56 TB', addr: '192.168.1.10:7878',
      roles: roles('Storage', 'Metadata', 'Media Server link', 'Subtitles', 'Notifying', 'Serving'),
      link: { dial: 'self', heartbeat: null, fingerprint: 'SHA256 7F3A 92C1 0B6E D4F8 51AA 3E0C 9D27 B6F1' },
      libraryRoot: '/Volumes/Vault/Media',
    },
    {
      id: 'forge', name: 'forge', kind: 'Worker', where: 'Home · LAN', hw: 'Desktop · RTX 4070', os: 'Windows 11', arch: 'x86_64', install: 'native',
      version: CONTROLLER_VERSION, state: 'online', cpu: 71, gpu: 88, disk: '1.6 TB free', addr: '192.168.1.31:7878',
      roles: roles('Converting'),
      link: { dial: 'controller', heartbeat: 2, latency: 1, fingerprint: 'SHA256 C04E 7B19 E2D3 6A80 F15C 0D94 2BE7 83A6' },
    },
    {
      id: 'seedbox-ams', name: 'seedbox-ams', kind: 'Worker', where: 'Amsterdam · remote', hw: 'Seedbox · 2 vCPU', os: 'Debian 12', arch: 'x86_64', install: 'native',
      version: CONTROLLER_VERSION, state: 'online', cpu: 9, disk: '1.1 TB free', addr: '185.107.56.24:7878',
      roles: roles('Searching', 'Downloading'), seeding: { count: 412, ratio: 2.4 },
      link: { dial: 'controller', heartbeat: 4, latency: 31, fingerprint: 'SHA256 19BD 44F0 A7C2 5E31 8C06 D9FA 70E4 2B5D' },
    },
    {
      id: 'relay-fra', name: 'relay-fra', kind: 'Worker', where: 'Frankfurt · VPS', hw: 'VPS · 1 vCPU', os: 'Ubuntu 24.04', arch: 'arm64', install: 'docker',
      version: CONTROLLER_VERSION, state: 'degraded', cpu: 3, disk: '18 GB free', addr: '91.198.22.140:7878',
      roles: roles('Relaying', 'Serving'), warn: 'Plex remote access unconfirmed', warnBody: 'Plex has not confirmed an inbound connection since 19:20. Remote players may fall back to Plex Relay.',
      link: { dial: 'controller', heartbeat: 3, latency: 22, fingerprint: 'SHA256 E6A1 0C8D 39F2 B4E7 1D05 7AC3 F928 64B0' },
    },
  ];
}

export const IMAGE = 'ghcr.io/slipstream/slipstream';

export function scenario(state) {
  let nodes = normalNodes();
  const byId = (id) => nodes.find((n) => n.id === id);
  const s = { state, nodes, notes: {} };
  if (state === 'single') {
    const a = byId('atlas');
    a.roles = roles(...ROLE_NAMES);
    a.single = true;
    a.cpu = 34;
    s.nodes = nodes = [a];
  }
  if (state === 'offline') {
    const sb = byId('seedbox-ams');
    Object.assign(sb, { state: 'offline', cpu: null, lastSeen: '2026-10-08T21:31:00', requeueAt: '2026-10-08T21:46:00' });
    sb.link.heartbeat = null;
  }
  if (state === 'skew') {
    const f = byId('forge');
    Object.assign(f, { version: '2.3.2', state: 'updating', update: { progress: 0.64, phase: 'Fetching the signed 2.4.0 release for windows/x86_64 from atlas' } });
    const r = byId('relay-fra');
    Object.assign(r, { version: '2.3.2', state: 'behind', warn: null });
  }
  if (state === 'drain') {
    byId('forge').roles.Converting = { state: 'draining', left: '2 Conversions finishing' };
    byId('seedbox-ams').roles.Downloading = { state: 'draining', left: '2 downloads finishing · 318 torrents still seeding to goals' };
    byId('atlas').roles.Downloading = { state: 'active', handedBack: true };
  }
  return s;
}

// The Node that answers the Join Code in the Pairing flow.
export function joiningNode() {
  return {
    id: 'cellar', name: 'cellar', kind: 'Worker', where: 'Home · LAN', hw: 'Synology DS923+', os: 'DSM 7.2 · Docker', arch: 'x86_64', install: 'docker',
    version: CONTROLLER_VERSION, state: 'online', cpu: 4, disk: '38 TB free', addr: '192.168.1.40:7878', roles: {},
    link: { dial: 'controller', heartbeat: 1, latency: 1, fingerprint: 'SHA256 4C2E 91A7 D06B 3F58 E7C1 0A94 B2D6 58F3' },
    suggested: ['Storage', 'Downloading'],
  };
}

export const JOIN_CODE = ['SLP1', 'K7QF2', 'M9XRT', '4HZ8N', 'PW3CJ', '6DVYA', 'QB52L', 'T8NKE', 'RF4GU', 'X9MZC', 'H7PDW', '2YQAS', 'L5VTK', 'C3RWE'];
export const CONTROLLER_ADDRS = ['192.168.1.10:7878', 'atlas.local:7878', '100.84.12.3:7878'];

// Content Type descriptors. Node pages render every Work row from these and never branch on a type.
export const contentTypes = {
  movie: { label: 'Movie', levels: [], item: '{edition}', layout: { artwork: 'poster', title: '{work}', sub: '{item} · {year}' } },
  series: { label: 'Series', levels: ['Season'], item: 'S{season:2}E{episode:2} · {name}', layout: { artwork: 'backdrop', title: '{work}', sub: '{item}' } },
};

const J = Object.fromEntries(jobs.map((j) => [j.id, j]));
const keys = {
  'j-sev-dl': { season: 2, episode: 7, name: 'Chikhai Bardo' },
  'j-dune-conv': { edition: 'Standard' },
  'j-tlou-xfer': { season: 2, episode: 3, name: 'The Path' },
  'j-opp-dl': { edition: 'Standard' },
  'j-int-imp': { edition: 'Standard' },
  'j-bat-fail': { edition: 'Standard' },
  'j-eeaao-search': { edition: 'Standard' },
};
for (const [id, k] of Object.entries(keys)) J[id].itemKey = k;
if (!J['j-tgm-conv']) {
  jobs.push({ id: 'j-tgm-conv', work: 'top-gun-maverick', itemKey: { edition: 'IMAX' }, target: '1080p SDR', stage: 'Conversion', node: 'forge', status: 'running', progress: 0.18, rate: 131, unit: 'fps', etaMin: 14, size: '≈ 8.1 GB', release: 'from 2160p HDR · NVENC HEVC · tone map HDR10 to SDR' });
}

export const torrents = [
  { work: 'dune-2', itemKey: { edition: 'Standard' }, ratio: 1.8, goal: 'ratio 2.0', tracker: 'Aurora' },
  { work: 'severance', itemKey: { season: 2, episode: 6, name: 'Attila' }, ratio: 0.9, goal: '14 days', tracker: 'Beacon' },
  { work: 'the-boys', itemKey: { season: 4, episode: 8, name: 'Assault on Democracy' }, ratio: 3.2, goal: 'ratio 2.0', tracker: 'Beacon' },
];
