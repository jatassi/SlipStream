# One file fills every Rendition it meets

Renditions are library-wide named definitions, and a Rendition profile picks which of them a Work wants. A Rendition is a target that a file meets or doesn't. So one file fills every Rendition in the profile whose rules it meets, and each Rendition judges cutoff and upgrades on its own. An upgrade for one Rendition deletes the old file only when that file no longer fills any wanted Rendition of any wanted Edition. The profile editor flags a Rendition that another fully contains, but overlap is never an error.

v1 went the other way. Its up to three slots had to carry mutually exclusive quality profiles, checked on save with conflict and naming modals, so one file only ever filled one slot. The cap of three had no recorded rationale. Decided in [The Edition and Rendition model](https://github.com/jatassi/SlipStream/issues/6#issuecomment-5988738426), Q2, Q3, Q10 and Q17.

## Considered Options

- **Mutually exclusive Renditions, as v1's slots.** Rejected. Users had to resolve conflicts that described no real problem, and a file that met two targets was stored twice.
- **Overlap allowed, one file fills one best-fit Rendition.** Rejected. It needs a tiebreak rule and still duplicates files on disk.

## Consequences

- Import, upgrade and deletion work on the set of Renditions a file fills, never on one slot.
- Naming must keep files that fill different Rendition sets apart. The import pipeline decides how.
