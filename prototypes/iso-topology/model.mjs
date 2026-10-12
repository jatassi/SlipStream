// PROTOTYPE truth for the isometric Nodes topology. Every number a figure shows or moves by comes from sample().
// Hardware follows Jackson's real setup; titles, rates and counts are invented but plausible.

export const NODES = {
  home: {
    id: 'home', name: 'Media server', host: 'Jackson-Plex-PC', where: 'Home', kind: 'Controller',
    hw: 'Core i5 · Intel UHD 750', os: 'Windows 11', addr: '192.168.1.20',
    disks: [
      { id: 'd1', label: 'Movies A', size: 18, used: 15.2 },
      { id: 'd2', label: 'Movies B', size: 18, used: 11.9 },
      { id: 'd3', label: 'Shows', size: 16, used: 13.4 },
      { id: 'd4', label: 'Downloads', size: 4, used: 1.7 },
    ],
    transcoder: { engine: 'Quick Sync · UHD 750', maxSessions: 4 },
    torrent: { engine: 'libtorrent 2.0', trackers: ['TorrentLeech', 'seedpool'] },
  },
  relay: {
    id: 'relay', name: 'Relay Node', host: 'ubuntu-2gb-ash-1', where: 'Ashburn · Hetzner VPS', kind: 'Relay',
    hw: '2 vCPU · 2 GB', os: 'Ubuntu 24.04', addr: '5.161.48.12',
    fronts: ['slipstream.atassi.org', 'plex.atassi.org'],
  },
};

export const TUNNEL = { name: 'WireGuard tunnel', port: 51820, upstreamMbps: 40 };

const STREAMS = [
  { who: 'Mom', title: 'The Bear · S3E4', device: 'TV', mbps: 8, transcode: false, where: 'remote' },
  { who: 'Sam', title: 'Dune: Part Two', device: 'phone', mbps: 6, transcode: true, from: '4K HDR', to: '1080p SDR', where: 'remote' },
  { who: 'Living room', title: 'Severance · S2E1', device: 'TV', mbps: 22, transcode: false, where: 'home' },
];

const DOWNLOADS = [
  { title: 'Andor · S2 · 2160p', size: 61, tracker: 'TorrentLeech' },
  { title: 'Conclave (2024) · 1080p', size: 14, tracker: 'seedpool' },
];

export const STATES = {
  calm: {
    label: 'Calm', sentence: 'Everything is running. One person is watching from outside the house.',
    streams: [0], downloads: [], seeding: 418, home: 'ok', relay: 'ok', tunnel: 'ok',
  },
  busy: {
    label: 'Busy evening', sentence: 'Everything is running. Three people are watching and two downloads are coming in.',
    streams: [0, 1, 2], downloads: [0, 1], seeding: 421, home: 'ok', relay: 'ok', tunnel: 'ok',
  },
  warn: {
    label: 'Relay warning', sentence: 'Plex has not confirmed remote access since 19:20. Remote viewers may fall back to Plex Relay.',
    streams: [0, 1], downloads: [0], seeding: 418, home: 'ok', relay: 'warn', tunnel: 'slow',
  },
  down: {
    label: 'Media server offline', sentence: 'The Relay Node cannot reach the media server. Last heard from at 21:31.',
    streams: [], downloads: [], seeding: 0, home: 'down', relay: 'ok', tunnel: 'down',
  },
};

const wobble = (t, seed, speed = 1) =>
  0.5 + 0.28 * Math.sin(t * 0.9 * speed + seed * 1.7) + 0.14 * Math.sin(t * 2.3 * speed + seed * 4.1) + 0.08 * Math.sin(t * 5.1 * speed + seed * 9.3);

export function sample(stateId, t) {
  const s = STATES[stateId];
  const streams = s.streams.map((i) => {
    const base = STREAMS[i];
    const mbps = +(base.mbps * (0.82 + 0.36 * wobble(t, i + 1, 0.6))).toFixed(1);
    const fps = base.transcode ? Math.round(52 + 14 * wobble(t, i + 7, 0.8)) : null;
    return { ...base, mbps, fps };
  });
  const remote = streams.filter((x) => x.where === 'remote');
  const transcodes = streams.filter((x) => x.transcode);
  const downloads = s.downloads.map((i, n) => {
    const base = DOWNLOADS[i];
    const MBps = +(12 + 14 * wobble(t, i + 3, 0.5)).toFixed(1);
    const done = (((t * MBps) / (base.size * 1024) + 0.31 + n * 0.27) % 1);
    return { ...base, MBps, done, peers: Math.round(18 + 22 * wobble(t, i + 11, 0.3)) };
  });
  const downMBps = downloads.reduce((a, d) => a + d.MBps, 0);
  const seedUpMBps = s.home === 'down' ? 0 : +(1.2 + 2.6 * wobble(t, 21, 0.4) + (downloads.length ? 1.5 : 0)).toFixed(1);
  const tunnelMbps = remote.reduce((a, x) => a + x.mbps, 0) + (s.home === 'down' ? 0 : 0.3);
  const transcodeLoad = transcodes.length ? Math.min(1, 0.38 * transcodes.length + 0.18 * wobble(t, 31, 1.2)) : 0.02;
  const diskIO = NODES.home.disks.map((d, i) => {
    const reading = streams.some((x, n) => (n + i) % 4 === i % 4) ? 0.3 + 0.6 * wobble(t, 40 + i, 1.6) : 0.04;
    const writing = d.id === 'd4' ? Math.min(1, downMBps / 40) : (downloads.length && i === 0 ? 0.25 * wobble(t, 50, 0.9) : 0);
    return { ...d, reading: s.home === 'down' ? 0 : reading, writing: s.home === 'down' ? 0 : writing };
  });
  return {
    state: stateId, label: s.label, sentence: s.sentence, t,
    health: { home: s.home, relay: s.relay, tunnel: s.tunnel },
    streams, remote, transcodes, downloads,
    downMBps: +downMBps.toFixed(1), seedUpMBps, seeding: s.seeding,
    tunnel: {
      mbps: +tunnelMbps.toFixed(1), share: Math.min(1, tunnelMbps / TUNNEL.upstreamMbps),
      latencyMs: s.tunnel === 'slow' ? Math.round(140 + 60 * wobble(t, 61)) : s.tunnel === 'down' ? null : Math.round(17 + 4 * wobble(t, 62)),
    },
    transcoder: { load: transcodeLoad, sessions: transcodes.length, fps: transcodes[0]?.fps ?? null },
    disks: diskIO,
    relay: {
      cpu: s.relay === 'down' ? 0 : Math.round(4 + 10 * wobble(t, 71) + remote.length * 6),
      tls: s.home === 'down' ? 0 : remote.length * 2 + 3,
      viewers: remote.length,
    },
  };
}
