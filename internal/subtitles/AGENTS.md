# subtitles

subtitles owns what each library file wants, the Speech Maps and Sidecars that meet it, and the Keeper. [ADR 0028](../../docs/adr/0028-speech-maps-stay-on-the-node-and-the-controller-carries-and-checks-every-subtitle-byte.md) is why the bytes move this way. Read it before changing a Sidecar's path from decision to disk.

## One writer each

- **Only the Keeper calls SidecarService.** `keeper*.go` lists, reads, writes and removes Sidecar bytes, and `maps.go` builds and reads maps. Every other Write records its decision as a row (an intent, a parked row, a removal, `timing_due` or `survey_due`) and wakes the key with `keeper.soon`. The `sidecar-writer` lint fails a third caller.
- **`unmet` and the fact columns** (`kind`, `wantless`, `audio`, `embedded`) are written only by `records.go`. A new write path to a Sidecar or want row calls `putSidecar`, `dropSidecar` or `putWants`. `subtitlestest` checks this invariant after every step through `Audit`.
- **A want's state** is `stateOf` (`state.go`). The Work page, Subtitle search and S47 (Subtitle reports) read it through `Outcome`, never from the rows.

## Gotchas

- The schema is migration 00001, frozen once merged. A schema change is the next subtitles migration in its own PR.
- A hook runs inside files' Write. Call nothing outside the library there, and wake the Keeper with `keeper.soon`, which waits `wake_delay` for the Write to commit.
- A Keeper settle reads its rows again each time, so a test waits with `World.Settle` or `World.Sweep` under `synctest`, never on a result channel.
