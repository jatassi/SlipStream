# State machines

This catalog lists the state machines that cross a Write, a restart or a Node boundary, with each state, what enters it and what leaves it. Each one names the code that owns it, which is the truth when this file and the code differ.

## Backup run

One Backup at a time, owned by `internal/backups` and recorded in the Backups index, `backups.json` in the data dir. An entry's `state` is one of four.

| State | Enters when | Leaves when |
|---|---|---|
| `taking` | A run starts: the scheduled Backup, Back up now, `slipstream backup`, the Backup before a migration or a newer version, before a restore, Replace passphrase, or Wipe and join's `superseded` Backup. The entry is listed before its file exists. | The archive is written, read back and its entries hashed again, then renamed in: `verified`. Any step fails: `failed`. A crash leaves it `taking`, and the next start or run checks its file and makes it `verified`, or drops it when only a `.part` is left. |
| `verified` | The read-back check passed and the file is in the Backups folder. Only now does retention prune, and only now is a copy queued to each destination. | Retention prunes it, or an Admin deletes it: `deleting`. The newest verified Backup is never pruned and refuses a delete. |
| `failed` | A step of the run failed. The entry keeps the cause in words, and Health warns, then errs once no verified Backup is recent. | The next run removes it. It counts for nothing. |
| `deleting` | Pruned or deleted. | Its local file is gone and each destination reported its copy deleted. The entry leaves the index. |

Each verified Backup has one copy per destination, with its own state.

| Copy state | Enters when | Leaves when |
|---|---|---|
| `waiting` | The Backup passed, or the destination was added, or a check found the copy missing or changed, or the library doesn't know the copy's Transfer. | A Low Transfer starts: `copying`. A Low Transfer starts only when no Normal Transfer waits on that Node pair, and runs as Normal once it has waited "Let a copy wait for downloads for up to". While Reconciling no Transfer dispatches. |
| `copying` | Its Transfer exists. | The Transfer finishes: `copied`. It fails: `failed`. |
| `copied` | The destination renamed the file in. Every "Check each destination's newest copy every", the Controller compares the destination's keyed BLAKE3 with its own. | A mismatch: `failed`, queued again as `waiting`. |
| `failed` | The Transfer or the check failed. Health warns for that destination. | The next destination run queues it again: `waiting`. |

## Restore stages

A restore crosses a restart, so its state is `lifecycle.json` in the data dir, owned by `internal/lifecycle` and resumed by the Node before library.db opens. The record exists only from the confirm step to the end of the restore's Write, and while it exists no other lifecycle operation starts (`lifecycle_busy`).

| Stage | Enters when | Leaves when |
|---|---|---|
| none | Staging: the archive streams to a `.part`, the passphrase opens it into the `.restore` files, `Inspect` refuses a newer schema, and except for a rollback the staged library migrates so the summary reads its Controller and Nodes. Nothing live is written. | The Admin confirms, on a new machine only with the old Controller off. In place, the `before-restore` Backup is taken into the kept index. The record is written and the Node restarts: `staged`. Cancelling deletes the `.restore` files. |
| `staged` | Every `.restore` file is synced and nothing live has changed. | At the next start, each live file with a `.restore` file renames aside to `.pre-restore`, and the `.restore` file renames in. The renames aren't stages: the plan is read from the files, so a crash between two renames resumes with the renames left. Then `swapped`. |
| `swapped` | The live files are the restored ones. library.db opens with no Backup before a migration. | The restore's Write commits, stamped with the record's id so it never runs twice: `written`. An automatic rollback skips the Write. |
| `written` | Reconciling has begun, the new key is in on a new machine, credentials are carried over in place, and sessions and Health items are gone. | The `.pre-restore` files are deleted, then the record. The Node runs as the restored Controller. |

Wipe and join uses the same record with one stage, `deleting`, written after its `superseded` Backup is listed. The resume deletes library.db, library.key, `backup.key` and `backups.json`, then joins with the record's Join Code.

## Reconciling

One row in library.db, `reconciling_state`, owned by `internal/reconciling`. `reconciling.State` answers `none`, `reconciling` or `superseded`, and every gate reads it in the snapshot of the Write it guards, so it holds across a restart.

| State | Enters when | Leaves when |
|---|---|---|
| none | No `reconciling_state` row and no fence. | A restore's Write inserts the row and one `reconciling_node` row per Node: `reconciling`. |
| `reconciling` | Searches, removals, renames, Transfers, Imports, the disk queue and Holds wait. Each Node keeps every torrent. Backups run and their copies wait. | One Write, once the re-home step is saved (on a new machine), every Node row is adopted or continued without, and every root has finished a scan since the start or was continued without. Finish now continues without everything left. The Write deletes the row, decides Grabs at fetching and publishes `reconciling.changed`: none. |

While `reconciling`, each Node's row moves through three states.

| Node row | Enters when | Leaves when |
|---|---|---|
| waiting | The restore's Write. The Node's intent carries `keep_all`. | The Node's last Holdings page is adopted: adopted. The Admin presses Continue without: continued without. |
| adopted | Adoption sets `adopted_at` and drops `keep_all`. | Reconciling ends. |
| continued without | Continue without or Finish now sets `continued_without_at`. The Node keeps `keep_all` until it reports. | Its Holdings are adopted later, or Reconciling ends. |

The review list, `reconciling_torrent`, outlives Reconciling: each row leaves by Re-add, Forget, or the Node reporting the hash.

## Keeping

A Node whose Holdings wait to be adopted reads `keeping` for the Downloading Role on the Nodes page, owned by the role board in `internal/nodes`. It replaces `draining` for that Node, so nothing it holds is dropped.

| State | Enters when | Leaves when |
|---|---|---|
| `keeping` | A Node pairs, re-pairs, or is listed by a restore's Write, and its first Holdings aren't adopted yet. Its intent carries `keep_all`: its client discards, fetches and adds nothing. | Its Holdings are adopted in one Write. A Node whose Holdings listed torrents gains the Downloading Role in that Write and reads `holding`. A Node with none reads no state for Downloading. Outside a restore, a recorded torrent the Node lacks goes back into its intent. |

## Superseded

An old Controller's fence, the `nodes_fence` row, owned by `internal/nodes`. `reconciling.State` answers `superseded` for every state but `unproven`.

| State | Enters when | Leaves when |
|---|---|---|
| none | No row. | A Worker this Controller pins answers with `Superseded`. With a proof that opens under this Controller's fence secret: `proven`. Without one: `unproven`. An Admin presses Retire this Controller: `retired`. |
| `unproven` | Only a Nodes Health Error names the Worker. The Controller keeps running. | A later reply carries a proof: `proven`. The Admin presses Stop this Controller: `stopped`. |
| `proven`, `stopped`, `retired` | Every gate holds, the built-in client pauses every torrent, no new intent goes out, every page opens the Superseded page, and no History row or notification is recorded. | Resume this Controller clears the row and regenerates the fence secret, which stops every earlier proof: none. Wipe and join takes a `superseded` Backup and writes the Wipe record, and the Node starts again as a Worker of the new Controller, keeping its torrents. |
