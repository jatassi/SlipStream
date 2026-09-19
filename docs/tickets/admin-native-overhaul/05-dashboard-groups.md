# 05: Dashboard as grouped sections with Group and Row

Spec: `docs/admin-native-overhaul-spec.md` (stories 23–28, 99)

**What to build:** The Dashboard answers "is anything wrong, how full is disk, what is downloading, what just happened" as four inset grouped sections: Health, Storage, Downloading and Recent. Health issues are rows with an amber warning tile that open System when tapped. Storage is one bar split by media type with used/total and per-folder figures. Downloading shows up to three active items with progress, speed and remaining time and a "See all" header action that opens Activity. Recent shows history events with a coloured tile per event type (Imported, Grabbed, Upgraded, Failed) and a relative time. Tapping a Downloading or Recent row opens that title's detail. While loading, skeletons have the shape of the final rows so nothing jumps.

This ticket introduces the shared `Group`, `Row` and `IconTile` primitives (inset container with optional header and header action; rows with leading tile or thumbnail, title, subtitle, trailing value, chevron, tone default/warning/destructive, 44 px minimum height, background tint on press) and the `ProgressLine` used for media-coloured progress. On wide screens the four groups lay out in two columns within the readable maximum width. Existing data hooks (health, storage, queue, history) are reused unchanged.

**Blocked by:** 03 (Two-shell layout)

**Status:** ready-for-agent

- [ ] Both projects: the Dashboard shows Health, Storage, Downloading and Recent as grouped sections populated from developer-mode data
- [ ] Both projects: a health issue row shows the amber tile and activating it opens System
- [ ] Both projects: Storage shows a single split bar with used/total and one line per root folder
- [ ] Both projects: Downloading lists at most three items with progress, speed and remaining time; "See all" opens Activity; activating a row opens that title's detail
- [ ] Both projects: Recent rows show the event tile, title and relative time; activating a row opens the title's detail
- [ ] Wide project: the groups render in two columns; phone project: they stack with the 16 px gutter
- [ ] Skeleton state renders rows of the same height as the loaded state (no vertical shift when data arrives)
- [ ] `Group`, `Row`, `IconTile` and `ProgressLine` are shared components reusable by later tickets and match the Native prototype's look and press behaviour
