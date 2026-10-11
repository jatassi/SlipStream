# Offline mode runs the release binary against wire-level fakes

The shipped binary has no offline mode. Offline mode is a harness, the Fake world, that starts the release binary as a Controller and Workers and points them at fakes through ordinary configuration. Every fixed external host lives in one typed `Endpoints` config block that defaults to the real host, and URLs that arrive inside data, such as announce URLs and indexer download links, come from the fakes. Each external boundary gets exactly one fake, a wire-level server speaking the real protocol. It runs over `net.Pipe` inside `testing/synctest` simulation tests and over loopback in the end-to-end suite, so the shipped client code runs in both. A Scenario seeds and scripts every fake from one file. Nothing blocks network egress. This principle, a Fake world that is the easiest way to run anything, and fakes for every boundary keep work offline. Decided in [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26).

## Considered Options

- **A runtime developer mode in the shipped binary, as in v1.** Rejected. In v1, ten production files outside the mock packages import a mock package. The TMDB mock returned pre-parsed data, so the real parser never ran. Dev mode still loaded `image.tmdb.org`, and the toggle copied the production JWT secret into the dev database.
- **Fakes compiled in behind a build tag.** Rejected. The suites would test a binary that never ships.
- **In-process fakes behind each client interface.** Rejected. They skip the parsing, auth, retry and TLS code where integration bugs live.
- **Redirecting real hostnames through DNS or a hosts file.** Rejected. It needs root and port 443, and it is machine-wide, so parallel worktrees collide.
- **An OS-level egress block in CI** (a loopback-only network namespace on Linux, with a DNS sinkhole and a canary). Rejected by the maintainer as heavy-handed. It only worked on Linux anyway.

## Consequences

- A CI check fails if `./cmd/slipstream` depends on any package under `fakes/`. The test-only music Content Type is registered only in `cmd/slipstream-e2e`, which a check keeps identical to `cmd/slipstream` apart from that one registration.
- The Fake world refuses to start while any `Endpoints` entry still points at a real host. Because nothing blocks egress, a new boundary added without an `Endpoints` entry and a fake reaches the internet unnoticed until review catches it.
- HTTPS fakes serve TLS from a CA minted per run. The harness trusts it through `SSL_CERT_FILE`, so the product has no extra-trust-roots setting. Every setting the harness changes from production is listed in one checked file.
- Two exceptions to one wire-level fake per boundary. libtorrent can't run inside a synctest bubble, so the simulation tier uses a fake torrent engine behind the engine's Go interface. The fake capability source and fake runner from ADR 0007 also stay. The end-to-end tier runs real libtorrent, ffmpeg and `dovi_tool`.
- Fakes for vendor APIs are SlipStream's own code. Standard protocols embed a reference implementation: Pebble and challtestsrv for ACME and DNS, the go-smtp server, pion/stun, and real libtorrent sessions as peers.
- Each boundary's scenario suite runs against the fake on every PR and against the real product online, so a fake that drifts fails a test. Committed fake content is fictional.
- Recorded exchanges for ported C# indexers count as fictional once their titles and identifiers are scrubbed. The porting spec may record each site live once. Decided in the [coherence pass](COHERENCE_PR), Q21.
- The Linux end-to-end suite is required on every PR. macOS and Windows run the full end-to-end suite, not smoke tests, on main, nightly, at release, and on PRs that touch native code. Decided in [Offline mode boundary](https://github.com/jatassi/SlipStream/issues/26), Q23, and the [coherence pass](COHERENCE_PR), Q23.
- [docs/offline-boundaries.md](../offline-boundaries.md) lists every external boundary and its fake. Each spec PR that adds a boundary appends its rows there.
- Time is faked only in the simulation tier. The end-to-end tier runs on the wall clock with product durations shortened through one `Timings` config, so it never runs production timing ratios.
- Agents prove a change with the verify-slipstream skill, which launches the release binary through `fakeworld`, reads a run manifest, and shares the Scenario catalog with the end-to-end suite. Each Node gets a `<node>.localhost` name that the browser resolves to its own loopback address, because a passkey never works at an IP address. The fault proxy knows a Node by the proxy port it dials, not by its source address, and a restarted Node comes back on its first port. Decided in [verify-slipstream skill design](https://github.com/jatassi/SlipStream/issues/27), Q4, Q11, Q34 and Q35.
