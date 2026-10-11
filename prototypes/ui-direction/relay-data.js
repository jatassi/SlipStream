// PROTOTYPE fixture for the Remote access screen. Every address, key and number is invented (RFC 5737 / 3849 ranges).
// A diagnostic is a set of exposures; each exposure walks the same hops, and each hop holds checks.
// Check: { id, label, status, observed, expected, meaning, fix?, because?, at }
//   status: 'pass' | 'warn' | 'fail' | 'skip' (an earlier hop failed, so the probe never got here) | 'off' (not part of this route)
//   fix: { label, action: 'apply' | 'open' | 'copy', detail }   because: id of the check that caused this one

export const HOPS = [
  { id: 'dns', label: 'DNS', icon: 'globe' },
  { id: 'relay', label: 'Relay', icon: 'radio-tower' },
  { id: 'tls', label: 'TLS', icon: 'lock' },
  { id: 'link', label: 'Node link', icon: 'cable' },
  { id: 'origin', label: 'Origin', icon: 'server' },
  { id: 'server', label: 'Media Server', icon: 'message-square-text' },
];

export const RANK = { off: 0, pass: 1, skip: 2, warn: 3, fail: 4 };
export const worst = (xs) => xs.reduce((a, s) => (RANK[s] > RANK[a] ? s : a), 'off');
export const hopStatus = (hop) => (hop.off ? 'off' : worst(hop.checks.map((c) => c.status)));
export const exposureStatus = (e) => worst(HOPS.map((h) => hopStatus(e.hops[h.id])));

// Content Types hand the UI descriptors, never code. Sections a Media Server shares render from these alone.
export const contentTypes = [
  { id: 'movie', noun: { one: 'Movie', many: 'Movies' }, icon: 'clapperboard', layout: { art: 'poster', aspect: '2 / 3', unit: { one: 'movie', many: 'movies' } } },
  { id: 'series', noun: { one: 'Series', many: 'Series' }, icon: 'tv', layout: { art: 'poster', aspect: '2 / 3', unit: { one: 'series', many: 'series' }, child: { one: 'episode', many: 'episodes' } } },
  { id: 'music', noun: { one: 'Album', many: 'Albums' }, icon: 'disc-3', layout: { art: 'cover', aspect: '1 / 1', unit: { one: 'album', many: 'albums' }, child: { one: 'track', many: 'tracks' } } },
];

const AT = '2026-10-08T21:38:00';
const ck = (id, label, status, observed, expected, meaning, extra = {}) => ({ id, label, status, observed, expected, meaning, at: AT, ...extra });
const hop = (label, sub, checks) => ({ label, sub, checks });
const off = (label, why) => ({ label, sub: '', off: why, checks: [] });

const RELAY_IP = '203.0.113.80';
const RELAY_IP6 = '2001:db8:4f::80';

function healthy() {
  return {
    id: 'healthy',
    lastRun: AT,
    probeFrom: 'seedbox-ams',
    headline: [],
    recommend: null,
    relay: {
      node: 'relay-fra', where: 'Frankfurt · VPS', ipv4: RELAY_IP, ipv6: RELAY_IP6, roles: ['Relaying', 'Serving'],
      link: { direction: 'atlas dials relay-fra', key: 'ed25519 9f3a·c41e', rtt: '18 ms', throughput: '412 Mbps', uptime: '6 d 4 h' },
      relayedToday: '38.4 GB', status: 'pass',
    },
    nodes: [
      { id: 'atlas', kind: 'Controller', where: 'Home · LAN', lan: '192.168.1.10', wan: '100.72.14.9', pub: '203.0.113.41',
        cgnat: { status: 'off', value: 'Behind CGNAT, relayed', note: 'Router WAN 100.72.14.9 is in 100.64.0.0/10 and differs from 203.0.113.41, the address seedbox-ams sees. relay-fra carries inbound traffic, so this does not matter.' },
        ipv6: { status: 'off', value: 'None from the carrier' }, portmap: { status: 'off', value: 'Not needed' },
        inbound: { status: 'off', value: 'Not needed, relay-fra takes it' }, reach: { status: 'pass', value: 'Is the Controller' }, latency: '' },
      { id: 'forge', kind: 'Worker', where: 'Home · LAN', lan: '192.168.1.24', wan: '100.72.14.9', pub: '203.0.113.41',
        cgnat: { status: 'off', value: 'Behind CGNAT, LAN only', note: 'Same router as atlas. forge only talks to atlas on the LAN.' },
        ipv6: { status: 'off', value: 'None' }, portmap: { status: 'off', value: 'Not needed' },
        inbound: { status: 'off', value: 'Not needed on the LAN' }, reach: { status: 'pass', value: 'Dials atlas on the LAN' }, latency: '0.4 ms' },
      { id: 'seedbox-ams', kind: 'Worker', where: 'Amsterdam · remote', lan: '10.8.0.12', wan: '203.0.113.57', pub: '203.0.113.57',
        cgnat: { status: 'pass', value: 'Public IP', note: 'WAN and observed address match.' },
        ipv6: { status: 'pass', value: '2001:db8:a1::57' }, portmap: { status: 'off', value: 'Not needed' },
        inbound: { status: 'pass', value: 'Accepts 443 and 51413' }, reach: { status: 'pass', value: 'atlas dials it' }, latency: '24 ms' },
      { id: 'relay-fra', kind: 'Worker', where: 'Frankfurt · VPS', lan: '10.0.0.4', wan: RELAY_IP, pub: RELAY_IP,
        cgnat: { status: 'pass', value: 'Public IP', note: 'WAN and observed address match.' },
        ipv6: { status: 'pass', value: RELAY_IP6 }, portmap: { status: 'off', value: 'Not needed' },
        inbound: { status: 'pass', value: 'Accepts 443, 32400, 13378' }, reach: { status: 'pass', value: 'atlas dials it' }, latency: '18 ms' },
    ],
    mediaServers: [
      { id: 'plex', name: 'Plex', node: 'atlas', port: 32400, remote: { status: 'pass', value: 'Fully accessible' }, lastInbound: '21:36',
        sections: [{ type: 'movie', title: 'Movies', count: 812 }, { type: 'series', title: 'Series', count: 96, child: 7341 }, { type: 'music', title: 'Music', count: 214, child: 2903 }] },
      { id: 'jellyfin', name: 'Jellyfin', node: 'atlas', port: 8096, remote: { status: 'pass', value: 'Remote connections allowed' }, lastInbound: '21:12',
        sections: [{ type: 'movie', title: 'Films', count: 812 }, { type: 'series', title: 'Shows', count: 96, child: 7341 }] },
    ],
    exposures: [
      {
        id: 'ui', name: 'SlipStream UI and portal', short: 'UI and portal', address: 'https://slip.atassi.example', entry: 'relay-fra :443', origin: 'atlas :8080', service: 'SlipStream',
        hops: {
          dns: hop('slip.atassi.example', RELAY_IP, [
            ck('ui-dns-a', 'A record points at relay-fra', 'pass', RELAY_IP, RELAY_IP, 'Browsers find relay-fra when they look up the domain.'),
            ck('ui-dns-aaaa', 'AAAA record points at relay-fra', 'pass', RELAY_IP6, RELAY_IP6, 'IPv6 players find relay-fra too.'),
          ]),
          relay: hop('relay-fra', ':443', [
            ck('ui-relay-port', 'Port 443 accepts connections from outside', 'pass', 'open from seedbox-ams', 'open', 'The relay is listening where browsers knock.'),
            ck('ui-relay-serving', 'Serving Role on relay-fra', 'pass', 'Serving', 'Serving', 'relay-fra answers web requests and passes them to atlas.'),
          ]),
          tls: hop("Let's Encrypt", 'until Dec 19', [
            ck('ui-tls-valid', 'Certificate is valid', 'pass', "Let's Encrypt R11 · expires Dec 19", 'valid for 14+ days', 'Browsers trust the connection without a warning.'),
            ck('ui-tls-san', 'Certificate covers the domain', 'pass', 'slip, jf, plex .atassi.example', 'slip.atassi.example', 'The name in the address bar is on the certificate.'),
          ]),
          link: hop('atlas dials out', 'to relay-fra · 18 ms', [
            ck('ui-link', 'Node link is up', 'pass', 'atlas dials relay-fra · 18 ms', 'up', 'Requests cross from relay-fra to atlas over the pinned-key link.'),
          ]),
          origin: hop('atlas', ':8080', [
            ck('ui-origin', 'SlipStream answers on atlas', 'pass', '200 in 41 ms', '200', 'The Controller serves the UI.'),
            ck('ui-portal', 'Portal answers', 'pass', '/portal 200', '200', 'Members can sign in from outside.'),
          ]),
          server: off('SlipStream', 'The UI is SlipStream itself, so there is no Media Server to ask.'),
        },
      },
      {
        id: 'plex', name: 'Plex', short: 'Plex', address: 'https://plex.atassi.example', entry: 'relay-fra :443 and :32400', origin: 'atlas :32400', service: 'Plex', mediaServer: 'plex',
        hops: {
          dns: hop('plex.atassi.example', RELAY_IP, [
            ck('plex-dns-a', 'A record points at relay-fra', 'pass', RELAY_IP, RELAY_IP, 'Plex players find relay-fra when they look up the domain.'),
          ]),
          relay: hop('relay-fra', ':443 :32400', [
            ck('plex-relay-port', 'Ports 443 and 32400 accept connections', 'pass', 'both open from seedbox-ams', 'open', 'Players can reach the relay by domain or by port.'),
          ]),
          tls: hop("Let's Encrypt", 'until Dec 19', [
            ck('plex-tls', 'Certificate covers plex.atassi.example', 'pass', 'valid · expires Dec 19', 'valid', 'Plex apps accept the domain without a warning.'),
          ]),
          link: hop('atlas dials out', 'to relay-fra · 18 ms', [
            ck('plex-link', 'Node link is up', 'pass', 'atlas dials relay-fra · 18 ms', 'up', 'Streams cross from relay-fra to atlas over the link.'),
          ]),
          origin: hop('atlas', ':32400', [
            ck('plex-origin', 'Plex answers on atlas', 'pass', '/identity 200 in 9 ms', '200', 'Plex is running and listening.'),
          ]),
          server: hop('Plex says', 'Fully accessible', [
            ck('plex-remote', 'Plex remote access', 'pass', 'Fully accessible', 'Fully accessible', 'Plex confirmed a connection from outside.'),
            ck('plex-url', 'Custom server access URL', 'pass', `https://plex.atassi.example:443`, 'includes the relay', 'Plex tells its apps to come through relay-fra.'),
            ck('plex-port', 'Public port Plex advertises', 'pass', '443 (manual)', '443', 'Matches the port relay-fra listens on.'),
            ck('plex-last', 'Last inbound connection', 'pass', '21:36 · 2 min ago', 'recent', 'Someone outside is watching right now.'),
          ]),
        },
      },
      {
        id: 'jellyfin', name: 'Jellyfin', short: 'Jellyfin', address: 'https://jf.atassi.example', entry: 'relay-fra :443', origin: 'atlas :8096', service: 'Jellyfin', mediaServer: 'jellyfin',
        hops: {
          dns: hop('jf.atassi.example', RELAY_IP, [
            ck('jf-dns-a', 'A record points at relay-fra', 'pass', RELAY_IP, RELAY_IP, 'Jellyfin apps find relay-fra.'),
          ]),
          relay: hop('relay-fra', ':443', [
            ck('jf-relay-port', 'Port 443 accepts connections', 'pass', 'open from seedbox-ams', 'open', 'The relay is listening.'),
          ]),
          tls: hop("Let's Encrypt", 'until Dec 19', [
            ck('jf-tls', 'Certificate covers jf.atassi.example', 'pass', 'valid · expires Dec 19', 'valid', 'Apps trust the connection.'),
          ]),
          link: hop('atlas dials out', 'to relay-fra · 18 ms', [
            ck('jf-link', 'Node link is up', 'pass', 'atlas dials relay-fra · 18 ms', 'up', 'Streams cross to atlas over the link.'),
          ]),
          origin: hop('atlas', ':8096', [
            ck('jf-origin', 'Jellyfin answers on atlas', 'pass', '/System/Ping 200', '200', 'Jellyfin is running and listening.'),
          ]),
          server: hop('Jellyfin says', 'Published URL matches', [
            ck('jf-url', 'Published server URL', 'pass', 'https://jf.atassi.example', 'https://jf.atassi.example', 'Jellyfin hands its apps the address that works from outside.'),
            ck('jf-remote', 'Remote connections', 'pass', 'Allowed', 'Allowed', 'Jellyfin accepts players that are not on the LAN.'),
          ]),
        },
      },
      {
        id: 'abs', name: 'Audiobookshelf', short: 'Audiobookshelf', named: true, address: `${RELAY_IP}:13378`, entry: 'relay-fra :13378', origin: 'atlas :13378', service: 'Audiobookshelf',
        hops: {
          dns: off('No domain', 'You named this port by hand, so players use the relay address directly.'),
          relay: hop('relay-fra', ':13378', [
            ck('abs-relay-port', 'Port 13378 accepts connections', 'pass', 'open from seedbox-ams', 'open', 'The relay forwards this port as plain TCP.'),
          ]),
          tls: off('Passed through', 'SlipStream forwards the bytes untouched. Audiobookshelf handles its own encryption, if any.'),
          link: hop('atlas dials out', 'to relay-fra · 18 ms', [
            ck('abs-link', 'Node link is up', 'pass', 'atlas dials relay-fra · 18 ms', 'up', 'Traffic crosses to atlas over the link.'),
          ]),
          origin: hop('atlas', ':13378', [
            ck('abs-origin', 'Something answers on atlas :13378', 'pass', 'TCP connect in 3 ms', 'connects', 'A service is listening on the port you named.'),
          ]),
          server: off('Not a Media Server', 'SlipStream has no link to Audiobookshelf, so there is nothing to ask it.'),
        },
      },
    ],
  };
}

const find = (d, id) => d.exposures.find((e) => e.id === id);
const check = (d, exp, hopId, checkId) => find(d, exp).hops[hopId].checks.find((c) => c.id === checkId);
const patch = (d, exp, hopId, checkId, fields) => Object.assign(check(d, exp, hopId, checkId), fields);
const skipFrom = (d, exp, hops) => hops.forEach((h) => find(d, exp).hops[h].checks.forEach((c) => {
  if (c.status === 'pass') Object.assign(c, { status: 'skip', observed: 'not reached', meaning: 'The probe stopped at an earlier hop.' });
}));

function plex() {
  const d = healthy();
  d.id = 'plex';
  d.relay.status = 'warn';
  d.headline = [{ k: 'warn', text: 'Plex unconfirmed since 19:20' }];
  const ms = d.mediaServers[0];
  ms.remote = { status: 'warn', value: 'Unconfirmed' };
  ms.lastInbound = '19:20';
  const fix = { button: 'Set custom URL', label: "Set Plex's custom URL to https://plex.atassi.example:443", action: 'apply', detail: 'SlipStream writes it through the Media Server link. Or in Plex, open Settings, Network, Custom server access URLs and paste https://plex.atassi.example:443.' };
  find(d, 'plex').hops.server.sub = 'Unconfirmed since 19:20';
  patch(d, 'plex', 'server', 'plex-remote', { status: 'warn', observed: 'Unconfirmed since 19:20', meaning: 'Plex has not seen a player reach it from outside since 19:20.', because: 'plex-url' });
  patch(d, 'plex', 'server', 'plex-url', {
    status: 'warn', observed: 'http://192.168.1.10:32400', expected: 'https://plex.atassi.example:443',
    meaning: 'Plex only tells its apps about its LAN address, so players outside never try relay-fra and fall back to Plex Relay at 2 Mbps.', fix,
  });
  patch(d, 'plex', 'server', 'plex-last', { status: 'warn', observed: '19:20 · 2 h 18 min ago', meaning: 'Remote players stopped arriving through relay-fra after 19:20.', because: 'plex-url' });
  d.recommend = {
    level: 'warn', hop: 'server', exposure: 'plex', check: 'plex-url', since: '19:20',
    chip: 'Plex degraded', title: 'Plex players outside your home fall back to Plex Relay at 2 Mbps',
    cause: "Plex's custom server access URL lists only its LAN address, so Plex apps never try relay-fra.",
    fix, alt: null,
  };
  return d;
}

function cgnat() {
  const d = healthy();
  d.id = 'cgnat';
  d.relay = null;
  d.nodes = d.nodes.filter((n) => n.id !== 'relay-fra');
  const atlas = d.nodes[0];
  atlas.cgnat = { status: 'fail', value: 'Behind CGNAT', note: 'Router WAN 100.72.14.9 is in 100.64.0.0/10, the carrier-shared range, and differs from 203.0.113.41, the address seedbox-ams sees. Your carrier shares that address with other customers.' };
  atlas.portmap = { status: 'fail', value: 'UPnP on, carrier refuses mappings' };
  atlas.inbound = { status: 'fail', value: 'Times out on 443, 32400, 8096' };
  d.headline = [{ k: 'fail', text: 'atlas is behind CGNAT' }, { k: 'fail', text: 'no Node relays for it' }];
  const turnOn = { button: 'Turn on Relaying', label: 'Turn on Relaying on seedbox-ams', action: 'apply', detail: 'seedbox-ams has a public IP and is already paired. atlas dials out to it, so CGNAT stops mattering. Relayed streams share its 1 Gbps uplink with seeding.' };
  const noRelay = (e) => ck(`${e}-relay-none`, 'A Node relays for atlas', 'fail', 'No Node holds Relaying', 'a Node with a public IP holds Relaying',
    'atlas is behind CGNAT, so nothing on the internet can dial it, and no Node relays for it.', { fix: turnOn, because: 'cgnat' });
  for (const e of d.exposures) {
    const h = e.hops;
    e.entry = 'atlas, direct';
    h.relay = hop('No relay', 'Relaying is off', [noRelay(e.id)]);
    if (!h.dns.off) {
      h.dns.sub = '203.0.113.41';
      h.dns.checks = [ck(`${e.id}-dns-a`, 'A record points at a carrier-shared address', 'warn', '203.0.113.41 (carrier-shared)', 'a public IP that reaches atlas',
        "The record names your carrier's shared address. Connections there land on the carrier, never on atlas.",
        { fix: { button: 'Copy record', label: 'Point it at seedbox-ams once Relaying is on', action: 'copy', detail: `${e.address.replace('https://', '')}  A  203.0.113.57` } })];
    }
    if (!h.tls.off) h.tls = hop('No certificate', 'not issued', [ck(`${e.id}-tls`, 'Certificate is valid', 'skip', 'none issued', 'valid', "No Node can answer Let's Encrypt from outside, so no certificate was issued.", { because: 'cgnat' })]);
    h.link = hop('No link', 'nothing to relay through', [ck(`${e.id}-link`, 'Node link to a relay', 'skip', 'none', 'up', 'There is no relay for atlas to dial.')]);
    skipFrom(d, e.id, ['origin']);
  }
  find(d, 'ui').address = 'https://slip.atassi.example';
  find(d, 'abs').address = '203.0.113.41:13378';
  const plexHop = find(d, 'plex').hops.server;
  plexHop.sub = 'Not available outside';
  plexHop.checks = [
    ck('plex-remote', 'Plex remote access', 'fail', 'Not available outside your network', 'Fully accessible', 'Plex agrees. Its own test cannot reach atlas from outside either.', { because: 'cgnat' }),
    ck('plex-url', 'Custom server access URL', 'off', 'none set', 'set once a relay exists', 'Nothing to set until a Node relays.'),
    ck('plex-last', 'Last inbound connection', 'fail', 'never', 'recent', 'No player outside has ever reached this Plex.', { because: 'cgnat' }),
  ];
  const jfHop = find(d, 'jellyfin').hops.server;
  jfHop.sub = 'URL unreachable';
  jfHop.checks = [ck('jf-url', 'Published server URL', 'warn', 'https://jf.atassi.example', 'an address that reaches atlas', 'The URL is right, but nothing at that address leads to atlas yet.', { because: 'cgnat' })];
  const ms = d.mediaServers;
  ms[0].remote = { status: 'fail', value: 'Not available outside your network' }; ms[0].lastInbound = 'never';
  ms[1].remote = { status: 'warn', value: 'URL unreachable' }; ms[1].lastInbound = 'never';
  d.cgnatCheck = ck('cgnat', 'atlas has a public address', 'fail', 'WAN 100.72.14.9 · public 203.0.113.41', 'WAN equals the public IP', atlas.cgnat.note);
  d.recommend = {
    level: 'fail', hop: 'relay', exposure: 'ui', check: 'ui-relay-none',
    chip: 'No route in', title: 'atlas is behind CGNAT, so nothing from the internet can reach it',
    cause: 'Your carrier gives atlas a shared address (WAN 100.72.14.9). Inbound tests from seedbox-ams time out on every port, and the router cannot map ports.',
    fix: turnOn,
    alt: { button: 'Pair a VPS', label: 'Pair a VPS instead', action: 'open', detail: 'A small VPS with a public IP keeps relayed streams off the seedbox. Create a Join Code and run SlipStream on it.' },
    then: 'Then point slip, plex and jf at 203.0.113.57. SlipStream gets the certificate on its own.',
  };
  return d;
}

function tls() {
  const d = healthy();
  d.id = 'tls';
  d.headline = [{ k: 'fail', text: 'DNS points at the old relay' }, { k: 'fail', text: 'certificate expired Oct 7' }];
  const OLD = '198.51.100.23';
  const records = { button: 'Copy records', label: 'Copy the 3 DNS records', action: 'copy', detail: `slip.atassi.example  A  ${RELAY_IP}\njf.atassi.example    A  ${RELAY_IP}\nplex.atassi.example  A  ${RELAY_IP}` };
  const staleDns = (e, name) => {
    const h = find(d, e).hops.dns;
    h.sub = `${OLD}`;
    patch(d, e, 'dns', `${e === 'ui' ? 'ui' : e === 'plex' ? 'plex' : 'jf'}-dns-a`, {
      status: 'fail', observed: `${OLD} (old relay)`, expected: RELAY_IP,
      meaning: `${name} still points at the relay you replaced, which no longer answers.`, fix: records,
    });
  };
  staleDns('ui', 'slip.atassi.example');
  staleDns('jellyfin', 'jf.atassi.example');
  staleDns('plex', 'plex.atassi.example');
  find(d, 'ui').hops.dns.checks = find(d, 'ui').hops.dns.checks.filter((c) => c.id === 'ui-dns-a');
  const renew = { button: 'Renew now', label: 'Renews on its own once DNS is fixed', action: 'apply', detail: 'SlipStream retries every 10 minutes. Renew now after you change the records.' };
  const expired = (id, name) => ck(id, `Certificate covers ${name}`, 'fail', 'expired Oct 7 · renewal failed', 'valid',
    `Let's Encrypt checked ${name} and reached ${OLD} instead of relay-fra, so the renewal failed. Browsers now refuse the connection.`, { fix: renew, because: 'ui-dns-a' });
  const ui = find(d, 'ui');
  ui.hops.tls = hop("Let's Encrypt", 'expired Oct 7', [expired('ui-tls-valid', 'slip.atassi.example')]);
  skipFrom(d, 'ui', ['relay', 'link', 'origin']);
  const jf = find(d, 'jellyfin');
  jf.hops.tls = hop("Let's Encrypt", 'expired Oct 7', [expired('jf-tls', 'jf.atassi.example')]);
  skipFrom(d, 'jellyfin', ['relay', 'link', 'origin']);
  jf.hops.server.sub = 'URL unreachable';
  patch(d, 'jellyfin', 'server', 'jf-url', { status: 'warn', meaning: 'Jellyfin hands out the right address, but the address does not lead to relay-fra.', because: 'ui-dns-a' });
  const p = find(d, 'plex');
  p.address = `https://plex.atassi.example, http://${RELAY_IP}:32400`;
  patch(d, 'plex', 'dns', 'plex-dns-a', { status: 'warn', meaning: 'plex.atassi.example points at the old relay. Plex apps try the next URL, the direct port, which works.' });
  p.hops.tls = hop("Let's Encrypt", 'expired Oct 7', [ck('plex-tls', 'Certificate covers plex.atassi.example', 'warn', 'expired Oct 7', 'valid',
    'The domain route is broken, but the direct port route uses Plex\'s own certificate.', { because: 'ui-dns-a' })]);
  p.hops.server.sub = `Fully accessible via :32400`;
  patch(d, 'plex', 'server', 'plex-url', { observed: `https://plex.atassi.example:443, http://${RELAY_IP}:32400`, meaning: 'Plex lists the domain first and the direct port second, so apps fall through to the port.' });
  patch(d, 'plex', 'server', 'plex-remote', { observed: `Fully accessible via ${RELAY_IP}:32400` });
  d.mediaServers[1].remote = { status: 'warn', value: 'URL unreachable' };
  d.mediaServers[1].lastInbound = 'Oct 7';
  d.recommend = {
    level: 'fail', hop: 'dns', exposure: 'ui', check: 'ui-dns-a', since: 'Oct 7',
    chip: 'Failed at DNS', title: 'slip, jf and plex still point at the relay you replaced',
    cause: `The A records name ${OLD}, the old relay. Browsers go there and find nothing, and Let's Encrypt could not renew the certificate, which expired Oct 7.`,
    fix: records,
    alt: { button: 'Check DNS again', label: 'Check DNS again', action: 'apply', detail: 'Records can take a few minutes to spread.' },
    then: 'SlipStream renews the certificate on its own once the records point at relay-fra.',
  };
  return d;
}

export const scenarios = {
  plex: { label: 'Plex URL', build: plex },
  cgnat: { label: 'CGNAT, no relay', build: cgnat },
  tls: { label: 'DNS and TLS', build: tls },
  healthy: { label: 'Healthy', build: healthy },
};
