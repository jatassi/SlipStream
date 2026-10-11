# SlipStream ships under GPL-3.0 and pulls indexer definitions at runtime

SlipStream v2 is free, self-hosted software with no commercial plans, licensed GPL-3.0. That lets v2 port Prowlarr's GPL-3.0 site-specific C# indexers instead of writing them from scratch, and v2 ports every active one (ADR 0013). It also keeps forks open, as across the rest of the *arr ecosystem. The Cardigann YAML definitions declare no license, so v2 never bundles them. The Controller downloads the catalog at runtime from a URL the admin can override, and offline mode uses a fake catalog of its own. Tests may fetch the live catalog into a cache that never enters the repo or a release (ADR 0014). A third-party notices file lists every bundled component, and CI checks each dependency's license against an allow-list. Decided in [SlipStream v2's license](https://github.com/jatassi/SlipStream/issues/31). Research in [Prowlarr catalog: vendor or reimplement](https://github.com/jatassi/SlipStream/issues/9).

## Considered Options

- **MIT or Apache-2.0.** Rejected. With no commercial plans, a permissive license buys nothing. It would also forbid porting Prowlarr code and allow closed forks.
- **AGPL-3.0.** Rejected. Its extra clause covers running SlipStream as a hosted service, which no one plans to do.
- **Bundle a snapshot of the YAML definitions.** Rejected by the maintainer after first choosing it. The files carry no license grant, and shipping them in a release redistributes them.

## Consequences

- Nothing in v2 may be MIT-only or proprietary-only in a way that conflicts with GPL-3.0. The CI allow-list enforces this.
- A fresh install with no network has no real indexer catalog until the Controller can fetch one.
- Bundled components keep their own terms and appear in the notices file: jellyfin-ffmpeg (GPL-3.0, run as a separate process, with a link to its source), `dovi_tool` (MIT), libfvad (BSD), libtorrent (BSD), and the Timing sync code ported from ffsubsync (MIT).
