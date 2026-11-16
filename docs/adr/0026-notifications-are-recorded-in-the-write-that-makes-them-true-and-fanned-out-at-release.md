# Notifications are recorded in the Write that makes them true and fanned out at release

An owner of a fact records a notification event inside the Write that makes the fact true, as ADR 0024 does for a Media Server change. Grabs, Imports and the Library each hold a `notice.Recorder` in their Config, and the only way to record is a typed catalog row's `Record`, which inserts one `notifications_event` row in the owner's `tx` and rings the dispatcher's doorbell. Health hands its transitions to notifications' sink inside the Write that records them. An event commits with its fact or not at all, so no crash loses one and a rolled-back Write leaves none. The doorbell rings before the commit. SQLite runs one Write at a time, so the dispatcher's next Write reads the event. The facts are stored as JSON, and each row renders them into words when the message is sent, so a copy change reaches messages still waiting. Decided in [Notifications](https://github.com/jatassi/SlipStream/issues/162) and the S37 (Notifications) build design.

An event waits unreleased until its `due_at`. A New Health item waits "Wait before sending a new Health item", and every other kind is due when it happens. The dispatcher's pass releases due events, and only then fans each one out: one `due` delivery for each connection that is on, was turned on before the event, has the kind on and passes its filters, and one row in the bell's list when the bell has the kind on. Every filter lives in one pure function. A Health item cleared goes only to the targets that were told the item, which `notifications_told` records when the message carrying it is sent, or when the bell shows it.

Delivery is at least once, and order is a promise about queued messages. Each connection sends its due deliveries oldest first with one in flight, and connections run in parallel. A crash after a provider accepted a message and before the Write that settles it sends that one message again, and nothing else repeats. A delivery row is deleted in the Write that settles it as sent, or drops it. A given-up delivery keeps its reason and the provider's response until its event is pruned, and the connection's line counts it.

## Considered Options

- **Watching the live hub for each owner's change event.** Rejected. The hub is an in-memory ring that a restart empties, and its events carry ids, not the facts a message needs after a delete.
- **Making deliveries when the event is recorded.** Rejected. A Health item waits before it is sent, and one that clears inside the wait sends nothing to anyone. Fanning out at release decides the targets once, with the item's final level.
- **An event log read by a cursor per connection.** Rejected. It loses the per-message outcome of a given-up message, splits the writers of a connection's row, and repeats the fan-out filter in SQL to count what waits.
- **Keeping every sent delivery as history.** Rejected. At 1,000 Imports an hour, three connections and a 30-day Keep, that is about 2 to 4 million rows that nothing shows an Admin.
- **An exported `Record` on the Recorder beside the typed row.** Rejected. An owner could record a kind no catalog row declares. `notifications` registers an owner's rows when it hands out that owner's Recorder, so recording an unregistered kind panics in every owner's tests.

## Consequences

- **A1.** A Test of edits that passes doesn't unpause a paused connection. The Save that follows bumps the revision, and that unpauses. A Test of the saved values with no edits that passes bumps the revision itself.
- **A2.** A Worse inside the wait replaces the waiting item at the newer level. A Worse of an item already released starts a new New Health item event, which waits again.
- **A3.** One evaluation makes at most one New Health item event and one Health item cleared event. A connection can want one kind and not the other, so a raise and a clear from one evaluation are two messages.
- **A4.** A Test may be in flight beside a queued message. A Test isn't queued, never waits behind a backlog and never shows in the bell.
- **A5.** An unpaused connection sends its backlog in order, one at a time, at the pace the provider allows. "Give up on a message after" keeps applying while it drains, so a message that passes the limit mid-drain is given up and counted.
- A connection's Health subset is fixed at release. Turning its warnings off later doesn't drop a Warning already due. Turning the connection off, deleting it or turning a kind off drops its due deliveries.
- The dispatcher keeps the results of sends in memory until a pass commits them, and claims nothing for a connection whose result isn't yet written. A pass whose Write fails changes nothing and is retried on the ladder, so a failed settle never sends a message twice.
- Every bulk step of a pass is bounded, so a large backlog never becomes one long Write.
- A Housekeeping step deletes events older than "Keep notifications for", giving up any delivery still due first. `notifications_told` rows outlive the prune, because a clear needs their words, and go when the item clears.
- A later spec adds an event as one catalog row in its own package and one `Record` call in the Write that makes it true. Connections, the bell, the providers and the events API take it with no other change.
