# grabs

grabs owns the search queue, every automatic and Manual decision, Grabs and their failures. [ADR 0025](../../docs/adr/0025-every-search-joins-one-keyed-queue-that-drains-within-each-indexers-limit.md) is why the queue has this shape. Read it before changing the queue, dispatch, a run's decision or failures.

## One writer each

Each of these has one home. Change it there, and never write a second path to the same rows.

- **A due row's queued entry** (lane, trigger, gone cells, scope, order) is written by `land` in `queue.go` through `pace.Merge`. The one other writer is Backlog search's `QueueBacklogDue` in `queries/backlog.sql`, which queues only idle rows, so it never meets a queued entry to merge with. `startRun` and `finishRun` own `search_id`, `due_at` and the gap. `CancelQueued` clears the `bulk` lane.
- **A trigger's lane, gap reset, gone cells, Holds and paced cells** are its row in `laneRules` (`triggers.go`). A caller passes what happened, never a lane.
- **A failed Grab's next step, count and refusal** are its Code's row in `afterFailure` (`failures.go`), read only through `ruleOf`. `move` reaches it through `failed`.
- **Every Grab, join and wait** comes from `walk` (`walk.go`) through `decideIn` and `commitClaims`. Build a `decisionIn` only with `runInput`, `rssInput`, `waitInput` or `rerunInput`, which set the scope, the open cells and Holds.
- **Room for a query** is indexers' answer from `Reserve`, in the dispatch Write. grabs never counts in-flight queries or limit slots. It only switches on reserve, `*indexers.WaitError` (settle the line) and `*indexers.RoomError` (the line waits).
- **What the queue is doing** is read through `readStates` (`searchstate.go`), which serves `/searching`, `/wanted`, the cells' search facts and the status sentence.

## Entry points

- Owners below grabs ask through `searches.Queue` inside their own Write. grabs' `Queue` (`seam.go`) writes nudge rows and nothing else. Read no cell and call no Facet there, because the caller's Write is still open.
- grabs' own Writes (the press, Search all, Backlog search) call `enqueue` directly.
- `fence` refuses with `ErrReconciling` while a restored Controller is Reconciling. Dispatch, the deciding Write, wait ends, Grab now, the press, Search all and the RSS unit's Write are fenced. Failure rows, nudges, the drain and Backlog search's reconcile are not, so their work waits for Reconciling to end. Fence any new Write that reserves a query or makes a Grab.

## Adding a trigger, lane or Code

Each is pinned by a test that fails until every part lands.

- **Trigger.** A `searches.Trigger` in `searches.Triggers()` order, a `laneRules` row (`TestEveryTriggerHasARule`), the trigger CHECKs in a new grabs migration (`TestTriggersMatchTheCheck`), and its words in `web/src/grabs/search-words.ts`.
- **Lane.** A rank in `pace.Lanes`, the lane CHECKs of `grabs_nudge`, `grabs_due` and `grabs_search` rebuilt in a new grabs migration (`TestLanesMatchTheCheck`), and its claim in `claimOf`.
- **Code.** An `afterFailure` row (`TestEveryCodeHasAFailureRule`). A Code that searches again at once must change what the next search can grab, or it grabs the same Release again.

## Gotchas

- A merged migration is frozen. A schema change is the next grabs migration number in its own PR.
- Storage panics on a Read inside a Write. Read Backlog search's interval and other settings before you open the Write, as `enqueue` callers do.
- The drain always opens its Write. A Read first misses nudges from a caller's Write that hasn't committed.
- Finished runs prune earlier runs of their group, so tests never count runs. Wait for a new run id with `worldtest.API.Searched`, or read `grabstest.World.Queue(t)`.
- Under `synctest`, `time.Now` starts in 2000. Code that compares with today's date reads `Config.Today`, and test configs set `Today: time.Now`.
- A test that wants one failed Grab without a search again sets `search.stopCount` to 1.
