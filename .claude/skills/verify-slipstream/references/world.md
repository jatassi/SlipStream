# The Fake world

`fakeworld start` launches the Scenario's Nodes from the run's binary copy and the fakes every boundary needs, and writes `manifest.json`: every Node's addresses, data dir, PID and state, every front, the releases with their sizes and BLAKE3 hashes, the browser block, the Admin and the API key. Read it before guessing any address. `$FW` with no arguments prints every verb.

## Isolation

A real SlipStream instance is out of reach as long as a run holds to four rules:

1. Type and dial only addresses the manifest names: `{nodes.<n>.url}`, `{nodes.<n>.browserUrl}` and the fronts. Every address an Admin enters in a recipe is a fake's front.
2. Touch only data dirs the manifest names. They sit in the run's scratch dir.
3. Drive only the fresh Chromium that `playwright cli` launches from the run's config; never your own browser, Claude in Chrome or a desktop browser pane.
4. Stop and kill only through `fakeworld stop` and `fakeworld node`, which act on the manifest's PIDs. Never kill by process name.

A production log or database is off limits. Doctor's checks 1 and 5 catch a run that slipped.

## Fronts

Each Node reaches each fake through its own front, a listener on `127.0.0.1` that the supervisor owns, so the world knows which Node dialled and a fault can hit one Node's traffic alone. The manifest's `fronts` list names each by `target` and `node` (`world` is the harness's own). Targets on `home`:

| Target | What crosses it |
|---|---|
| `tmdb` | metadata lookups (routes `search`, `movie`) |
| `omdb` | ratings and awards lookups (route `title`) |
| `fixture-site` | every Indexer site, each at its own host under `fixture-site.fake.localhost` (routes `<site>.<route>`, such as `torznab.api`, `brook.rss`, `larkspur.search`) |
| `tracker` | announces (route `announce`) |
| `smtp` | the SMTP server an Admin saves, plaintext with STARTTLS or implicit TLS on one front |
| `seeders` or `seeders/<release>` | every swarm byte from the fake seeders, TCP only |
| `link/<node>` | the Node link to the Node that listens, for example `link/home` |
| `transmission` | fake-transmission's RPC (route `rpc`), on a Scenario that gives the Node a Transmission, such as `external-clients`. The world typed its front as the client's address |
| `qbittorrent`, `deluge`, `rtorrent` | fake-qbittorrent's WebUI API (route `webui`), fake-deluge's JSON-RPC (route `json`) and fake-rtorrent's XML-RPC (route `rpc`), on a Scenario that gives the Node that kind, such as `external-kinds`. The world typed each front as the client's address |
| `dht` | KRPC queries to fake-dht, the Node's one DHT router. A UDP socket per Node, not a front, so fault rules don't play on it |

The Indexer was entered as `https://torznab.fixture-site.fake.localhost/torznab`. Each Node's `endpoints.searching.sites` sends every host under `fixture-site.fake.localhost` to that Node's own fixture-site front, so a fault rule on `fixture-site@seedbox` plays on the seedbox's searches and one on `fixture-site@home` on the Controller's. A feed entered at `https://prowlarr.fixture-site.fake.localhost/torznab` lists the same Releases with each item naming the source `Fixture Source`, as Prowlarr does, so its first Grab makes that Tracker.

## Faults

```text
fakeworld fault set <run> <id> <target>[@<node>] <fault> [--route r] [--param k=v]... [--skip n] [--times n] [--after <kind>[,k=v...]]
fakeworld fault clear <run> [<id>]
fakeworld fault list <run>
fakeworld fault wait <run> [--rule id] [--node n] [--after seq] [--timeout d]
```

The grammar of `<fault>`, one per rule:

```text
refuse | reset | stall | latency=200ms | throttle=256KiB/s | cut=64KiB | tls=untrusted|expired|wrong-name
status=<code>[,retry-after=<dur>][,reset-header=<name>] | hang | malformed | truncate | slow-body=<rate>
content-type=<media type>
```

- The first six and `tls=` are connection faults; the rest are HTTP faults and need a fake that speaks HTTP. A rule the front can't play is refused when set, naming the layer.
- `@<node>` limits a rule to the traffic of that Node; without it the rule holds every origin.
- `--skip 2 --times 1` plays the third match only. `--times 0`, the default, plays until cleared. A rule that has played its times drops out of `fault list` and doctor's active faults. A connection fault other than `reset` stays listed until `fault clear`, because the connections it took keep playing it.
- `--after <kind>,k=v` arms the rule on the first matching event-log record, never on a timer.
- Setting an id again replaces the rule. `fault clear` without an id clears every rule, the Scenario's included.
- `fault set` prints the `fault.set` record, and every firing logs `fault.fired` with its match count.

"Complete this torrent now" is `fault clear` on a throttle over `seeders`: the live connection speeds up, and the Node's libtorrent still downloads and hashes every piece. "Drop the Node link" is `fault set link-down link/home stall`, or `reset`.

## Cues

`fakeworld cue <run> <name> [<json args>]` plays a cue a fake declares; the manifest's `fakes[].cues` lists them. A cue changes only the world outside every Node. The `smtp` fake's cues each turn a play on, and `{"off": true}` turns it off: `smtp-auth-failure` (535), `smtp-defer` (451 to RCPT), `smtp-reject` (550 to RCPT), `smtp-tls-required` (530 to MAIL without TLS), `smtp-refused` (closed on accept) and `smtp-slow` (silent for `{"delay": "30s"}` before the greeting).

The `omdb` fake's `omdb-revoke-key` refuses `{"key": "…"}`, or every key it has seen, with OMDb's bad-key answer. `omdb-cap` spends the rest of today's allowance of `{"key": "…"}`, or of every key, so OMDb answers "Request limit reached!" until the next UTC midnight.

`fixture-site-list` with `{"release": "<name>"}` lists a release the Scenario marks `unlisted: true`, so a search after the cue finds it and one before does not. The `targets` Scenario holds its Bluray back this way until its WEB-DL is imported. `fixture-site-refuse` with `{"code": 101}` has the Torznab API answer every request with that Newznab error, which stops each Indexer on the site, and `{"code": 0}` serves requests again.

`fixture-site-rss-item` with `{"release": "<name>"}` lists an `unlisted: true` release in Brook's RSS feed only, at `https://brook.fixture-site.fake.localhost/rss`. Brook's `?play=` takes `no-items`, `no-link`, `repeated-guid` or `no-size`, and any `brook_session` cookie other than the Scenario's gets 403.

Five catalog sites run a definition from the fake catalog, each at its own host: Larkspur signs in with a form and an image captcha and wants thanks before a download, Quarry takes an API key, Cinder takes a pasted cookie behind a browser challenge whose `cf_clearance` holds only for the Node and User-Agent it was earned on, Marsh signs in by POST and answers XML in windows-1251, and Ferry signs in at a one-time link with its passkey. A session holds only on the Node that signed in. `fixture-site-expire` with `{"site": "<label>"}` signs every session of that site out and stops its pasted cookie, so the next search signs in again. `fixture-site-maintenance` with `{"site": "<label>", "on": true}` answers 503 to every page until a cue with `"on": false`.

Five family sites each fake one family of ported site with the protocol of the port of the same id, at `<port id>.fixture-site.fake.localhost`: PassThePopcorn takes its API user and key as headers on every request, TorrentBytes signs in with a form, AvistaZ signs in with a username, password and PID to a bearer token, IPTorrents takes a pasted `uid` and `pass` cookie only with the browser's User-Agent, and NorBits signs in with a form and a two-factor code. NorBits' account has two-factor sign-in on, so a sign-in with no code shows the page that asks for one, and it takes the RFC 6238 code of the seed's authenticator key, whether the Node makes it from the key or the Admin types it for a Test. `fixture-site-expire` and `fixture-site-maintenance` take a family site's label too, and expiring AvistaZ ends its bearer tokens. Every other port's host replays its fixture from `fakes/site/ports/<port id>/`, and from any other host its exchanges name, such as an `api.` host under the port's. With no case cued a request gets the first exchange of any case that matches it. `fixture-site-port-case` with `{"port": "<port id>", "case": "<case>"}` answers from that case's exchanges in order and starts it over after its last, and `"case": ""` goes back to any case. A request out of order gets status 599 naming the exchange it expected.

A Scenario's `library.indexers` names each Indexer's `kind` (`torznab`, `rss`, `catalog` or `native`), its `site`, for a catalog Indexer its `definition`, for a native Indexer its `port`, the same id as its site, and for either its plain `settings`. Credentials come from the seed on a catalog or family site, and from the fixture's `settings` and pasted session on a port with no family site. With several Nodes holding Searching, each names its `home`. Setup adds a catalog Indexer through the API as an Admin would, after the catalog's first refresh, typing the code its captcha shows. The `catalog-sites` Scenario is one Controller with Quarry added at setup, for adding Larkspur by hand. The `indexers` Scenario has home and seedbox both holding Searching, and den holding no Role: Quarry, Larkspur, Marsh and Cinder are homed on seedbox, which signs them in and holds their sessions, and Fixture Site and Brook on home. seedbox has a Challenge solver, which clears Cinder's browser challenge, and home and den have none. It has four movies with one release each, so each search on a new Work can grab. The `ported-sites` Scenario has one native Indexer on each family site: PassThePopcorn, Torrent Bytes and AvistaZ homed on home, which holds every Role but Relaying, and IPTorrents and NorBits on seedbox, which holds Searching. NorBits signs in with its authenticator key. It has five movies with one release each. `$FW site $RUN <label> [--node <name>]` prints what an Admin enters for one of fixture-site's sites as JSON: `origin`, a feed's `feed` and `apiKey` (`torznab`, `prowlarr`), Brook's `feed` and `cookie`, or a catalog or family site's `settings`, for Cinder and IPTorrents the `cookie` and `userAgent`, Cinder's earned on that Node (the first Node by default), and for NorBits the two-factor `code` of now, for a Test with the authenticator key left out.

## Captures and the event log

`events.jsonl` in the run dir is the one ordered log of the run: launch, Node lifecycle, `node.error` lines, front opens and closes with bytes each way, faults, cues, steps, SSE messages and captures. Readers read the file, so they work on a live run, a stopped one and a downloaded CI bundle.

```sh
$FW announces list $RUN --node seedbox
$FW announces wait $RUN --node seedbox --where event=completed --after <seq> --timeout 60s
$FW announces open $RUN <seq> --proof <f>/<s>
```

- `announces` is the tracker's capture: one `tracker.announce` record per announce, attributed to the Node by the front it came through. `data.event` is `started`, `completed` or empty for a regular announce, which libtorrent spaces five minutes apart; wait on `started` and `completed` only.
- `mail` is the `smtp` fake's capture: one `smtp.message` record per accepted message with its envelope (`from`, `to`), `username`, `security` (`none`, `starttls` or `tls`), `subject`, `text` and `html`, attributed to the sending Node. `open` copies `message.eml`, `message.html` and `message.txt`.
- `lookups` is fake-dht's capture: one `dht.query` record per KRPC query, attributed to the Node by the socket it came to. `data.method` is `ping`, `find_node`, `get_peers` or `announce_peer`, and `data.infoHash` is set for the last two. `$FW lookups list $RUN --node home --where method=get_peers --where infoHash=<hex>` lists a Node's lookups of one torrent. A private torrent's info hash never appears.
- `captchas` is fixture-site's record of each captcha image a site served, with its `site` and the `text` it shows. `sign-ins` records each sign-in a site accepted, with its `site` and `method`, attributed to the Node that signed in.
- `searches` is fixture-site's record of each Torznab search a site answered, with its `site`, `t`, `q`, `tvdbid`, `season` and `ep`, attributed to the Node that asked. `$FW searches list $RUN --where t=tvsearch --where season=0` lists the searches for specials.
- `leecher.hello` is the Scenario's leecher (`leecher: {node, release}`), attributed to its Node. After that Node's first accepted announce of the release, the leecher dials the Node's listen port through its `leecher/<node>` front and says a BitTorrent hello. `data.answered` says whether the Node answered for the release, and `data.error` says why not. Under the `loopback-peers` divergence the connection proves the listen port, so the Node's client card reads reachable. A Scenario with no leecher reads unproven. `home` has a leecher on seedbox.
- `works.added` is the world's one record for a Scenario's `library.works`, written once every Work is added: `data.contentType`, `data.count` and `data.seconds`, the time the adds took. The `scale` Scenario adds 20,000 movies that fake-tmdb generates from `tmdb.generated {count, seed}`, with ids from 50000001.
- `wait` takes the first match after `--after`, by default after the previous step's start. When the awaited record can arrive during an earlier step, pass `--after` the seq of a record printed before that step, such as a `fault.set` or `fault.fired`.
- `open` copies a record's files into the proof dir.

## fs

```sh
$FW fs stat $RUN <path>
$FW exec $RUN --proof <f>/<s> -- $FW fs match {runDir} '{mediaDir}/movies/<folder>/<file>' --release <release>
```

`stat` prints `{path, size, links, blake3}`. `match` exits 0 when the file's size and BLAKE3 equal the manifest's release, which proves a placed file from the receiving side. Both refuse a path outside the run dir and the scratch dir. Tokens expand only inside `exec`, hence the second form.

## Summary and replay

- `fakeworld summary <run>` writes `summary.md`: the build stamp, the Scenario, the last doctor result, every step with its exit and proof dir, the fault and cue timeline, every `node.error` with its step, and the proof file list. Text only, from the run dir alone.
- `fakeworld replay <run> [--to N] [--stop]` checks the bundle's `scenario.yaml` and binary against its manifest, starts a fresh world from the same Scenario, and runs the journaled steps 1 to N again, stopping at the first whose exit differs from the recorded one. A CI bundle replays its recorded test command instead; pass `--binary` with the uploaded `slipstream-<target>`.
- A bug verify finds gets a regression test built from the run's journal in the PR that fixes it.
