# A Request stores only Admin decisions and derives its stage from the Work tree

A Request names a Work, the levels or Items a Member picked, and whether future Items are included. It stores only what people decide: who asked and who watches, the note, the Admin's approve or deny with its reason, whether approval was automatic, when each decision and watch was made, and the moment it first became Available. Every stage a Member sees after approval (Searching, Downloading, Processing, Available, plus the Delayed flag) is computed from the requested Items' cells in the Work tree when asked for, never moved by events. An Item counts for a Request once it is Playable, meaning it has any library file a Media Server can play. A pending Request adds nothing to the library. Approval, by an Admin or automatically, is one server-side step that adds the Work or monitors the picked nodes, then searches. Decided in [Requests portal under Editions and Targets](https://github.com/jatassi/SlipStream/issues/88#issuecomment-6110902226), Q1, Q5, Q9, Q12, Q13 and Q16.

## Considered Options

- **A stored status moved by library events**, as v1 did. Rejected. v1 set a Request to failed on a download failure and then skipped failed Requests when later events arrived, so a retry that worked never reached Available. Every new pipeline step (Transfer, Conversion, import) would add another event to wire and another way to drift.
- **Available when every wanted Edition and Target is filled.** Rejected. The Media Server picks the version (ADR 0009), so the first Playable file is what the Member waits for. Waiting on Conversion queues and rare cuts would hold Requests open for weeks.
- **Available when the Profile's first Target is filled.** Rejected. It makes a Member wait on a Target they never see.
- **Adding the Work when the Request is made, unmonitored.** Rejected. Denied and cancelled Requests would leave orphaned Works in the Admin's library.
- **A per-Member Profile picking the new Work's Profile**, as v1 did. Rejected. One Work serves every player through several Targets (ADR 0002), and the result would depend on who asked first. The Content Type's default Profile applies, and the Admin can change it at approval.

## Consequences

- A Request never reverts. Once Available it keeps its date, and later upgrades, derived Targets or extra Editions never notify again.
- There is no terminal Failed stage. Delayed is computed from how long requested released Items have gone unplayable since their first search (3 days) or from a failed requested cell, while Backlog search keeps working underneath. Backlog search's longest gap for a requested Want stays under 3 days. Decided in [Search, RSS sync and grab across Nodes](https://github.com/jatassi/SlipStream/issues/87#issuecomment-6110599137), Q16.
- Progress and stage come from the Grab's Work and Items, which [Import pipeline under Editions and Targets](https://github.com/jatassi/SlipStream/issues/42) records, and travel over SSE (ADR 0013). No client matches downloads by title.
- Series report per level from the Content Type's level descriptors (ADR 0001), so no Content Type needs Request code beyond its requestable nodes.
- Cancelled is never stored. Cancelling hands the Request to its earliest Watcher, or deletes it when none watch.
- Request limits are counted from existing auto-approved Requests in the current week, so a delete refunds without a counter or reset job. A Member cancels only while a Request is Pending, and an auto-approved Request never is, so only an Admin's delete refunds its unit. Decided in the [coherence pass](https://github.com/jatassi/SlipStream/pull/112), Q8.
