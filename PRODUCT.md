# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Open. A map ticket decides it with research and prototypes ([Chart SlipStream v2](https://github.com/jatassi/SlipStream/issues/2), Q4). v2 ships v1's package set plus Docker images.

## Users

Two audiences carry equal weight.

- **Admins** run SlipStream on their own hardware. The first is the chef; v2 is then packaged and documented for strangers who self-host. An Admin adds and monitors Works, tunes Profiles, Targets and Custom Formats, assigns Roles to Nodes, and needs to know at a glance whether everything is moving and what broke. They work at a desk and from a phone.
- **Members** are people an Admin invites with a Signup link. They request Works through the portal and learn when those Works are ready to watch. They never see the machinery: Nodes, Roles, Conversions, indexers or Download Clients.

## Product Purpose

SlipStream finds, downloads, processes and files media into a library that Media Servers play, across one or more machines. v2 is a rebuild of v1 that keeps every feature a v1 user can see, plus what v1's docs list as not started ([parity matrix](docs/parity-matrix.md)).

Success for an Admin: Works arrive at every wanted Edition and Target without babysitting, and failures surface with their cause and stage. Success for a Member: ask for something, then watch it.

## Positioning

- **One app replaces the *arr stack.** A built-in indexer catalog, a private-tracker-grade torrent client, subtitles with timing sync, Conversions and a requests portal replace Sonarr, Radarr, Prowlarr, Bazarr, Overseerr and Tdarr. Usenet stays external.
- **Multi-machine by design.** One Controller owns the library database and the UI. Workers join by Pairing and do the Roles assigned to them, such as a seedbox that downloads and keeps seeding while a storage Node imports, or a GPU box that converts. Transfers move files between Nodes, and Handover moves the Controller.
- **Edition × Target.** Every Item can want several cuts, each at every Target in its Profile. A Target is downloaded or derived from another Target by a Conversion, and one file meets every Target whose rules it passes.
- **TRaSH Guides native.** Custom Formats, scores, quality sizes and naming import from TRaSH Guides, run verbatim and stay tracked.

## Operating Context

- Admins operate SlipStream alongside Plex, Jellyfin or Emby, which serve the library to players. SlipStream never live-transcodes for playback.
- Setups range from a single machine (a Controller holding every Role) to several Nodes on a LAN or remote, reached through the Relaying and Serving Roles with TLS and a domain.
- Admins live in long-running state: live download progress, scheduled tasks, health alerts and events pushed over a live connection.
- Members mostly arrive on a phone, from a Signup link or a notification that something is ready.
- Developers, agents and CI run an offline mode with network egress blocked.

## Capabilities and Constraints

- Content Types are pluggable. v2 releases with movies and TV; a test-only third type proves extensibility.
- Every row of [docs/parity-matrix.md](docs/parity-matrix.md) is kept unless the chef explicitly agreed to drop it on that row. This includes the wide desk shell and the phone shell (bottom tabs, push navigation, safe areas, 44 px targets, action sheets), global search, the dashboard, and light, dark and system themes (P-012).
- Identity: one Account system. Admin is a permission on an Account. Sign-in is an email plus a 4-digit PIN or a passkey; only Admins issue Signup and Reset links; a fresh install prints a Setup code. Scripts use an API key. Email goes over generic SMTP.
- v1 data comes forward through a one-shot importer.
- Architecture is locked in [docs/adr/](docs/adr/).
- Undecided: the stack; the visual direction ([UI redesign direction](https://github.com/jatassi/SlipStream/issues/25), pick pending on the [prototype PR](https://github.com/jatassi/SlipStream/pull/54)); Role Group names other than Acquire.

## Brand Commitments

- The name is SlipStream, written with a capital second S.
- UI copy uses [GLOSSARY.md](GLOSSARY.md) terms and avoids each term's _Avoid_ list: Work, not title or show; Target, not quality version; Conversion, not transcode; Node, not instance; Member, not user.
- A logo and colour palette were in progress in v1 (P-017) and are not settled.

## Evidence on Hand

- No real library content, artwork, users, testimonials or usage numbers are in the repository. Everything in the UI prototype (titles, posters, people, Nodes, numbers) is invented.
- Future work must not present fabricated Works, people or benchmarks as real. Third-party artwork comes from metadata sources at runtime.

## Product Principles

1. **Members never see machinery.** The portal speaks in outcomes: requested, coming, ready.
2. **Answer "is it moving, and what broke" first.** Admin surfaces lead with live state and name the stage and cause of any failure.
3. **Parity is a floor.** Nothing a v1 user relied on disappears without the chef's recorded yes.
4. **One vocabulary everywhere.** The glossary is the UI's language, so docs, logs and screens agree.
5. **Distributed, not complicated.** A single-machine install should feel complete; more Nodes add capacity without adding ceremony.

## Accessibility & Inclusion

WCAG 2.2 AA across the desk and phone shells, Admin and Member alike: AA contrast, full keyboard paths, and screen-reader equivalents for any visual that carries state, such as a canvas or animated status display.
