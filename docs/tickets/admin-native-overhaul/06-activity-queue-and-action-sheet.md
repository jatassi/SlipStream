# 06: Activity queue rows, segmented filter and the action presenter

Spec: `docs/admin-native-overhaul-spec.md` (stories 49–57, 84–85)

**What to build:** The Activity page (`/downloads`) opens with a large title and a summary line (n downloading, total speed, n in queue). Each queue item is a row with poster thumbnail, title, episode label, release name in small monospace, a media-coloured progress line, and percent, speed and time remaining in tabular figures. A 44 px pause or resume control at the trailing edge toggles the item without opening anything; importing and queued items show their state text in that slot instead of a control. Tapping a row opens an action sheet with Pause or Resume, Remove from queue (with the existing delete-files choice), and Cancel; destructive actions are red and separated from the safe ones. The All, Movies and Series filter is a segmented control with a sliding thumb. The download-client-unreachable banner remains as an amber row at the top. Progress advances smoothly between updates.

This ticket introduces two shared primitives that later tickets reuse:

- `Segmented`: sliding-thumb control for two to five options, keyboard operable, replacing the Tabs primitive here.
- `ActionSheet` and the action presenter: a bottom-anchored list of actions with title, optional monospace subtitle, destructive tone and Cancel on phones; the same action list renders as a dropdown menu (for non-destructive lists) or alert dialog (for confirmations) on wide screens through one presenter component, so callers describe actions once.

Blocklist: the spec's action sheet lists "Remove and blocklist release", but the queue API has no blocklist operation and the spec forbids backend changes and new features. The action sheet ships without it; if a blocklist endpoint is added later, the action slots in.

**Blocked by:** 05 (Dashboard as grouped sections with Group and Row)

**Status:** ready-for-agent

- [ ] Both projects: Activity shows the summary line and one row per developer-mode queue item with thumbnail, title, release name, progress line and tabular numbers
- [ ] Both projects: activating the trailing control on a downloading item changes its state text to paused and back to downloading on a second activation, without opening anything
- [ ] Both projects: importing and queued items show state text where the control would be and expose no pause control
- [ ] Phone project: activating a row opens the action sheet with Pause/Resume, Remove from queue and Cancel; Remove is styled destructive and separated; choosing it removes the row after confirmation
- [ ] Wide project: activating the row's action opens the same actions as a menu; Remove confirms in a dialog
- [ ] Both projects: the segmented filter narrows the list to movies or series and back to all
- [ ] Both projects: toggling the developer-mode download client off shows the amber unreachable row at the top; toggling it on removes it
- [ ] Progress values between two consecutive updates render intermediate widths (no single-frame jump); numbers use tabular figures
- [ ] `Segmented` and the action presenter are shared components documented for reuse
