# Speech Maps stay on the Node that holds the file and the Controller carries and checks every subtitle byte

A Speech Map is built and kept on the Storage Node that holds its file, because it is read from the file's audio and only that Node can read it without moving the file. The Subtitles Role's holder searches and downloads subtitles and runs Timing sync, and it keeps nothing between calls: each call carries the file's map, the Profile's rules and Timing sync's constants, and no holder reaches a Controller store. The Controller relays every byte between them. It fetches the map from the Storage Node, sends it with the call, and checks the answer before any byte reaches a library folder. Paired Workers are not trusted (#558), so the Controller parses both byte sets itself, refits the holder's claimed segments, and recomputes the timing match against the file's own map. Prominence needs the full correlation, so it stays the holder's claim. The recomputed match still bounds a forged pass. Decided in the S46 (Subtitles) design.

Every Sidecar byte effect is a row first. A Write that decides a Sidecar should exist, move, be retimed or go writes an intent (`subtitles_write`), a parked row, a removal, a `timing_due` flag or a `survey_due` flag, and nothing else. One Controller component, the Keeper, settles those rows one file at a time through a queue keyed by file id. It is the only caller of `SidecarService`'s List, Read, Write and Remove, and the `sidecar-writer` lint keeps it so. A doomed file's parked and removal rows carry that file's id, so they settle behind any call still in flight for it. Each settle reads the rows again, so a wake for nothing costs one Read and a settle that stops anywhere is finished by the next. The Keeper writes only a `Checked`, which three constructors make: the Controller's check of a holder's answer, its own parse of an upload, and a survey's companion. A removal names a hash, and the Node removes only bytes with that hash, so a file the Admin put at that name is kept and surveyed.

The Storage Node keeps an append-only ledger of the hashes it wrote under each root. A survey after a restore reads a companion the ledger knows as SlipStream's own, timed again, and every other one as found. Nothing decrements the ledger, so there is no refcount and no crash window. `internal/calls` is the call layer the metadata providers and the Subtitle Providers share: lines, buckets, gates, tries, classify and the allowance an answer reports. A provider's pause and allowance persist on its `subtitles_provider` row, never in `calls`.

## Considered Options

- **Building maps on the Subtitles Node.** Rejected. It would read every library file over the network, and a Worker would see the library's audio.
- **Writing Sidecars from the holder or the Storage Node directly.** Rejected. The Controller couldn't check a Worker's bytes, and two writers of one name would race.
- **A refcounted ledger.** Rejected. A crash between the count and the rename leaves the count wrong for good.
- **Each Write calling SidecarService itself.** Rejected. A call inside a Write holds the library's lock across the network, and a Write that rolls back leaves bytes no row names.

## Consequences

- A Storage Node that is offline or not ready leaves the rows for the next wake, and nothing is lost.
- While a restored Controller is Reconciling (ADR 0019), the Keeper writes, carries and removes nothing.
- Found and uploaded Sidecars are never overwritten. They go only through Mark bad or with their file.
- A later PR adds a Keeper step as one more read of the key's rows and one more guarded finish. No other caller of `SidecarService` appears.
