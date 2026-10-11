# A restored Controller takes a new key, reconciles toward what Nodes hold, and fences off the old one

A Controller restored on a new machine makes its own key, and each Worker pairs again with a Join Code. The Controller recognises a Worker's public key from the restored database, so the Worker keeps its Node record, Roles and intent. Library roots that lived on the old Controller are re-homed by the same root edit a moved disk uses (ADR 0016). The restored Controller then starts in a Reconciling state, as does every restore in place and every manual rollback. It starts nothing new and cancels in-flight jobs from the Backup. Workers report what they hold, and every library root is rescanned. Torrents the database doesn't know are adopted under the default and per-tracker seeding goals, and unmatched files go to pending imports. Torrents the Backup lists but no Node holds go to a review list with Re-add or Forget. Automation resumes once reconciliation finishes. A Worker that pairs again keeps the replaced Controller key as superseded, and answers that old Controller with "superseded by key X". The old Controller then stops its torrents and automation and offers to wipe and join as a Worker. Decided in [Config backup and restore](https://github.com/jatassi/SlipStream/issues/91), Q15 to Q18 and Q20.

Handover is not a restore. It is a planned move between two live machines, so none of a restore's distrust applies, and it reuses the restore code only for the archive. Workers follow the notice the old Controller signs with its key, with no Join Code. No library root is re-homed, and nothing in flight is cancelled. The old Controller becomes a Worker that keeps its Roles, roots and torrents, and redirects Workers that reconnect late. Decided in the [coherence pass](COHERENCE_PR), Q4.

## Considered Options

- **Re-apply the restored intent**, as a rebuilt Worker does under ADR 0003. Rejected. A Backup is older than the world. Re-applying it would download again torrents removed on purpose, and dropping torrents it doesn't know would cause hit and runs.
- **Carry the Controller's key** so Workers reconnect on their own. Rejected for ADR 0018's reasons. Workers also couldn't tell a restored Controller from a live old one.
- **Leave split brain to the admin.** Rejected. Two Controllers announcing the same private torrents from two addresses risks a tracker flag (inferred).
- **Guess library roots by scanning disks.** Rejected. A wrong guess is silent, and every file it misses looks missing and gets searched.

## Consequences

- The "superseded" reply sits in the handshake package, which ADR 0003 freezes, so it ships in the first release.
- While Reconciling, nothing removes or pauses a torrent, deletes or renames a library file, blocklists, or searches.
- A Worker that is already paired, such as a seedbox or a relay VPS, accepts a new Join Code from its local page, a CLI flag or a Docker environment variable.
- Workers removed after the Backup show as offline until the admin force-drains them. Workers paired after it pair again as new Nodes, in a restore in place too.
- A restore can start from the first-run page, gated by the Setup code, from `slipstream restore` over the local control socket that [Packaging, the updater and the update dashboard](https://github.com/jatassi/SlipStream/issues/93) uses for host commands, or from Settings on a running Controller. A restore in place takes a safety Backup first and holds the same lifecycle lock as an update and a Handover.
- A single-machine restore with no Workers relies on the restore page asking the admin to confirm the old machine is gone.
- A restore in place keeps its key, so Workers reconnect at once, but its Backup is still older than the world. So every restore and every manual rollback reconciles. The automatic rollback in ADR 0015 stays outside this, because only migrations wrote between its Backup and the rollback. Decided in the [coherence pass](COHERENCE_PR), Q5.
