# TRaSH Custom Formats run verbatim and stay tracked

SlipStream imports Custom Formats from TRaSH Guides and keeps tracking them, rather than importing once or overwriting on a schedule as Recyclarr does. Each imported item keeps its `trash_id`. The Controller checks upstream on a schedule and shows a diff. Untouched fields update. A field the admin edited stays local, and a field changed on both sides shows as a conflict. A format deleted upstream stays as a local format until the admin removes it. Release-group lists in the guide change weekly, so a one-shot import rots, and blind overwrite destroys local score tuning.

Tracking per field only works if an imported format is byte-identical to upstream. So the Custom Format regex contract is .NET syntax, case-insensitive, run on a backtracking engine with a per-match timeout, not the linear-time RE2 syntax of Go's and Rust's standard libraries. Decided in [Custom formats engine and TRaSH Guides import](https://github.com/jatassi/SlipStream/issues/7), Q4, Q7, Q10, Q11 and Q16.

## Considered Options

- **RE2 syntax with a generated translator.** Rejected. Measured against TRaSH at `e7c97a6`, 71 of 1,342 unique regexes fail under Go's `regexp`. 57 have exact rewrites that agree with a .NET-compatible oracle on 31,883 release names, but 30 of those need a generator and one grows twelvefold. 10 more need formats restructured or split, which breaks tracking by `trash_id`. 4 stay approximate. The translator would rerun on every upstream change.
- **One-shot import, re-run by hand.** Rejected. Release-group lists go stale within weeks.
- **Continuous overwrite sync.** Rejected. It discards the admin's score tuning.

## Consequences

- The stack must offer a .NET-compatible regex engine. Go's `dlclark/regexp2` compiles all 1,342 TRaSH regexes, and Rust's `fancy-regex` all but 4, which trip on a `[([]` class (compilation checked, matching not).
- Under offline mode, the upstream guide is a boundary whose fake is a pinned snapshot on disk. Releases bundle a snapshot (MIT, notice kept), which setup offers as a starting point.
