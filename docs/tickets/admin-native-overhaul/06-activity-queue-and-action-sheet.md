# 06: Activity queue rows, segmented filter and the action presenter

Spec: `docs/admin-native-overhaul-spec.md` (stories 49–57, 84–85)

**What to build:** The Activity page (`/downloads`) opens with a large title and a summary line (n downloading, total speed, n in queue). Each queue item is a row with poster thumbnail, title, episode label, release name in small monospace, a media-coloured progress line, and percent, speed and time remaining in tabular figures. A 44 px pause or resume control at the trailing edge toggles the item without opening anything; importing and queued items show their state text in that slot instead of a control. Tapping a row opens an action sheet with Pause or Resume, Remove from queue (with the existing delete-files choice), and Cancel; destructive actions are red and separated from the safe ones. The All, Movies and Series filter is a segmented control with a sliding thumb. The download-client-unreachable banner remains as an amber row at the top. Progress advances smoothly between updates.

This ticket introduces two shared primitives that later tickets reuse:

- `Segmented`: sliding-thumb control for two to five options, keyboard operable, replacing the Tabs primitive here.
- `ActionSheet` and the action presenter: a bottom-anchored list of actions with title, optional monospace subtitle, destructive tone and Cancel on phones; the same action list renders as a dropdown menu (for non-destructive lists) or alert dialog (for confirmations) on wide screens through one presenter component, so callers describe actions once.

Blocklist: the spec's action sheet lists "Remove and blocklist release", but the queue API has no blocklist operation and the spec forbids backend changes and new features. The action sheet ships without it; if a blocklist endpoint is added later, the action slots in.

**Blocked by:** 05 (Dashboard as grouped sections with Group and Row)

**Status:** done

- [x] Both projects: Activity shows the summary line and one row per developer-mode queue item with thumbnail, title, release name, progress line and tabular numbers
- [x] Both projects: activating the trailing control on a downloading item changes its state text to paused and back to downloading on a second activation, without opening anything
- [x] Both projects: importing and queued items show state text where the control would be and expose no pause control
- [x] Phone project: activating a row opens the action sheet with Pause/Resume, Remove from queue and Cancel; Remove is styled destructive and separated; choosing it removes the row after confirmation
- [x] Wide project: activating the row's action opens the same actions as a menu; Remove confirms in a dialog
- [x] Both projects: the segmented filter narrows the list to movies or series and back to all
- [x] Both projects: toggling the developer-mode download client off shows the amber unreachable row at the top; toggling it on removes it
- [x] Progress values between two consecutive updates render intermediate widths (no single-frame jump); numbers use tabular figures
- [x] `Segmented` and the action presenter are shared components documented for reuse

## Comments

The action sheet keeps the existing "Fast forward" action for mock download clients; it is the developer-mode shortcut the old row actions had and it has no place else to live. The old page header's "View History" button is gone — History is a first-class destination in both shells.

`Segmented` renders a `radiogroup` rather than a tablist: it switches a filter, it does not reveal tab panels. The thumb animates `transform` only; the label colour still crossfades over 150 ms, as in the Native prototype.

`DropdownMenuContent` gained an `anchor` passthrough so the presenter can position a menu it opens itself, without a trigger element. `tab-panes.tsx` and `push-routes.ts` list the activity pane as screen-filling now that the page owns its own scroll container through `Screen`; the back-label logic in `push-routes.ts` is untouched.

Blocklist is still absent, as the ticket states: the queue API has no blocklist operation.

E2E: the queue checks run against real developer-mode data — the mock client grabs, pauses, resumes and removes for real, and the unreachable row comes from a genuinely offline qBittorrent client created and deleted through `/downloadclients`. Three things the mock cannot hold still are served from `e2e/helpers/queue-stub.ts`: a `queued` item (mock downloads are queued for two seconds), an importing one (they complete after five minutes), two progress values a frame apart, and a queue holding exactly one movie and one series for the filter check. The stub intercepts the WebSocket as well, because `queue:state` frames write straight into the query cache.

Every real grab in the spec is an episode of a series of the project's own: series always have missing episodes, whereas the mock library runs out of grabbable movies once the client has imported them. `activity.spec.ts` runs `mode: 'serial'` and removes its grab only in the last test, so it adds one download per project — the dashboard spec reads the same queue and its Downloading group shows only the first three items of an unordered list.

Two helpers in `e2e/helpers/dev-data.ts` needed the same tolerance: `ensureRecentHistory` now returns the newest history entry that carries a media title (episode grabs are recorded without one), and `dashboard.spec.ts` takes the first progress line in the Downloading group rather than assuming there is only one.
