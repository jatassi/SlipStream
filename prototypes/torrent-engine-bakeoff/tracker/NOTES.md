# tracker

Build: `cd tracker && /usr/local/go/bin/go build -o tracker .` (stdlib only).

Run: `tracker/tracker --log FILE --interval 1800 --min-interval 900 --listen 127.0.0.1:7070,127.0.0.1:7071 [--fail-infohash HEX,...] [--enforce]`

- Path `/<passkey>/announce`; anything else 404. Raw-query parsing keeps binary info_hash/peer_id intact. IPv4 compact peers only, ip from the connection (or `ip=` param); the announcer is excluded from its own list; `stopped` removes the peer. Peers expire after 2x interval (lazily, on the next announce for that infohash).
- Rules mirror UNIT3D `checkMinInterval` at 8b88f4c (keyed passkey|infohash|peer_id). `duplicate_event_30s`: a second announce with the same event (empty counts as `empty`) within 30 s of the first one in the window. `min_interval_violation`: any announce other than completed/stopped inside 0.9 x min interval of the announce that opened the window; the window is opened by whichever announce finds no live window, a stopped more than 5 min into a window clears it, and a rejected-as-duplicate announce touches neither window.
- With `--enforce` a violating announce gets `failure reason` listing the kinds and is logged `rejected:true`. `--fail-infohash` always rejects with `fake failure` and records no violations. Malformed requests (info_hash or peer_id not 20 bytes) are logged rejected with "bad request".
- Log: JSONL per the contract; `peer_id` is printable-escaped (`\xNN` for non-printables), buffered 1 MiB, flushed every 1 s and on SIGTERM/SIGINT (listeners drained first).
- Load: 14k announces/s on loopback with a 64-goroutine Go client on the same box (client likely the limit, not the tracker).
- `--delay-ms D` sleeps D ms before handling each announce, to model a remote tracker's latency.
