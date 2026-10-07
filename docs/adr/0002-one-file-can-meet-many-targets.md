# One file can meet many Targets

Targets are library-wide named definitions, and a Profile picks which of them a Work wants. A file meets a Target or doesn't. So one file meets every Target in the Profile whose rules it passes, and each Target judges cutoff and upgrades on its own. An upgrade for one Target deletes the old file only when that file no longer meets any wanted Target of any wanted Edition. The Profile editor flags a Target that another fully contains, but overlap is never an error.

v1 went the other way. Its up to three slots had to carry mutually exclusive quality profiles, checked on save with conflict and naming modals, so one file only ever filled one slot. The cap of three had no recorded rationale. Decided in [The Edition and Target model](https://github.com/jatassi/SlipStream/issues/6#issuecomment-5988738426), Q2, Q3, Q10 and Q17.

## Considered Options

- **Mutually exclusive Targets, as v1's slots.** Rejected. Users had to resolve conflicts that described no real problem, and a file that met two slots was stored twice.
- **Overlap allowed, one file meets one best-fit Target.** Rejected. It needs a tiebreak rule and still duplicates files on disk.

## Consequences

- Import, upgrade and deletion work on the set of Targets a file meets, never on one slot.
- Naming must keep files that meet different Target sets apart. The import pipeline decides how.
