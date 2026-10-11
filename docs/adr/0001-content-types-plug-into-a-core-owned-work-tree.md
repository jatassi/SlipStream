# Content Types plug into a core-owned Work tree

Each Content Type is a module compiled into the binary behind one interface. SlipStream loads no third-party plugins. The core owns the tree of Works, levels and Items, and every generic mechanism that acts on it: monitoring, files, Editions, Targets, quality, decisioning on Coverage, the naming engine, request lifecycle, calendar rendering and the UI. A Content Type supplies what only it knows: its metadata providers, Item keys, Coverage parsing for release names and file paths, search groups and queries, naming tokens and templates, requestable nodes, calendar events and availability, its own tables and migrations, and UI descriptors with layout hints. No core data shape carries a per-type field, and no Content Type ships UI code.

v1 had a module interface too, yet grep found 149 places outside its modules that branch on movie or TV (measured). Most came from core shapes that mixed movie and TV fields. So v2 enforces the boundary with a build-time check that fails when core code names any concrete Content Type. The offline end-to-end suite also runs the whole flow against a music Content Type that exists only in tests. In that test type, the track is the Item, the album is a level, and deluxe or remaster releases are Editions of each track.

## Considered Options

- **Out-of-process plugins.** Rejected. They add a versioned ABI and a security surface that no user needs.
- **Declarative Content Types in data.** Rejected. Data can't express Coverage parsing or import matching.
- **Per-type UI pages.** Rejected in favor of descriptors, so that adding a Content Type never touches the UI.

## Consequences

- The music type has the same shape as movies and TV, so a track's file fills a Target exactly as a movie's or an episode's does (ADR 0011). Decided in the [coherence pass](COHERENCE_PR), Q3.
