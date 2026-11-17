# History records every fact in the Write that makes it true

One log of what happened to the library is owned by the core `history` package, and History, the bell and every Notification connection read it. An owner records an entry through its `entry.Recorder` inside the Write that makes the fact true, as ADR 0026 had owners record notification events. `history` inserts the entry with every Work it is about, its Grab and Import, and its Health items, then hands it to its Outbox, which `notifications` implements, in the same Write. Every Grab failure is recorded, and a failure SlipStream deals with itself is quiet: History lists it and nothing is sent. A delete across Content Types records one Works deleted entry per type. The kinds are rows in one `entry.Catalog`, which owners register into while the Node wires the Controller, and which is sealed before any runner starts. Decided in the S38 (Calendar and History) design.

`notifications` keeps what it still owes as its own rows. Queue writes one pending row per entry it may send, due at once, or after "Wait before sending a new Health item" for a New Health item. A Worse inside that wait stays on the earlier entry and keeps its due time, and a `notifications_worse` row says which newer entry the key renders from. A clear inside the wait takes the key back, and the raise goes when no key is left. Release fans a due pending row out as ADR 0026 describes.

Housekeeping prunes History in batches. An entry goes once it is older than "Keep history for", unless notifications still owes it: a pending row, a due delivery, or the newer entry a pending or due message renders a key from. Deleting an entry deletes its Works, its items and notifications' rows of it. `history.changed` tells the web after a burst of commits and after a sweep that deleted entries.

## Considered Options

- **Keeping the log in notifications and reading it from History.** Rejected. History would list only what some target can be sent, and a Test, which records nothing, would sit beside the library's facts.
- **Replacing the waiting item in place on a Worse, as ADR 0026 did.** Rejected. History keeps both entries as they were captured, so the wait moves to a pending row that points at the newer entry.
- **A group or Health key column on each entry.** Rejected. A kind's group comes from its catalog row, and Health keys live on the entry's items.
- **Carrying "Keep notifications for" into "Keep history for".** Rejected. Their ranges and defaults differ.

## Consequences

- ADR 0026's A2 and A3 still hold as behaviour: a Worse inside the wait sends one message at the newer level when the first raise is due, and one evaluation is at most one New Health item and one Health item cleared entry.
- A message renders its entry's stored facts when it is sent, so the words follow the code that sends them.
- The migration keeps every event's id as its entry's, and carries the id sequence and the bell's sequence, so ids and read cursors stay right.
- A later spec adds a kind as one catalog row and one `Record` call in the Write that makes it true. History, the bell, the connections and the events API take it with no other change.
