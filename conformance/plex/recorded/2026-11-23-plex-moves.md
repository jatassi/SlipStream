# Plex Moves recorded run, 2026-11-23 (S48 (Moves), #173)

Copied from Jackson's media server (C:\Users\jacks\Git\SlipStream\recorded-runs-out\moves\moves-run.md and its .jsonl, which hold no token). Plex Media Server 1.43.5.11029 on Windows, 01:49 to 01:59 UTC, five passes.

Server prefs (read only): autoEmptyTrash true, FSEventLibraryUpdatesEnabled true, FSEventLibraryPartialScanEnabled true, ScheduledLibraryUpdatesEnabled true (900 s). Throwaway sections A (4) and B (5) with section 2's agent tv.plex.agents.movie, scanner Plex Movie, en-US. Seed: one 4 s mkv, matched plex://movie/5d776824e6d55c002040ae63.

| Case | Result |
|---|---|
| Setup | Item 91644 appeared 1.15 s after `refresh?path=`, already matched. Set viewCount 1, userRating 8, lastViewedAt, collection "Moves Test" (91645). |
| 1. Cross-section copy, refresh B's folder | New ratingKey 91646, same guid, 0.77 s after the refresh. viewCount, rating and lastViewedAt showed at once while 91644 still existed (watch state follows the guid per account, inferred). viewOffset empty on both. Collection not carried. 91644 unchanged. |
| 2. Collection PUT in B | Made a new per-section "Moves Test" (91647). Sending only `collection[0]=Moves Other` added it beside the existing one; sending both changed nothing. A collection PUT never removes (`collection[].tag.tag-=` untested). |
| 3. Delete A's copy, refresh A's path | First try failed with a Windows sharing violation 0.8 s after case 2: Media Analyzer and credits detection hold a freshly scanned file about 1 to 2 s. After waiting, the delete worked first try. `refreshing` false 0.5 s after the refresh; 91644 answers 404 and A is empty. 91646 keeps viewCount 1, rating 8, lastViewedAt and both collections. |
| History | A scrobble writes no history row, so history was empty for both ids before the delete. History for a deleted id answers 404 (treat as none). Whether history follows the guid is unmeasured. |
| 4. Same section, two locations | `PUT /library/sections/5` with only two `location` params answers 400; with name, type, agent, scanner, language and both locations, 200. A copy to section-b2 plus `refresh?path=` of b2 joined 91646 as a second Part (0.79 s), state and collections unchanged. After deleting the section-b copy (Plex held it 1.02 s), `refresh?path=` of the section-b folder destroyed 91646 ("file count 1 -> 0", "Destroying metadata item 91646") although the b2 copy was on disk. Collections stayed, empty. About 60 s later Plex's own scan of b2 re-added it as new ratingKey 91651 with watch state and rating back (guid) and no collections. |
| 4b. Same section, whole-section refresh | A copy b2 to section-b plus `refresh?path=` of section-b made 91652, which Plex found a duplicate part of by hash, merged into 91651 and destroyed: 91651 has two Parts (0.48 s). After deleting the b2 copy, `GET /library/sections/5/refresh` with no path: `refreshing` false 0.5 s later, 91651 kept with the same ratingKey, one Part at section-b, its watch state, rating and collection ("count is now 1", no Destroying). 91653 in the log is a trailer extra, not a library item. |

Single run of each case on a tiny section. The cost of a whole-section refresh on a large library is unmeasured.

## What this means for the build
- A copy in another section is a new item found by path. Watch state and rating follow the guid at once; collections never follow a new ratingKey, so the Move writes them back. A collection PUT adds and never removes.
- A copy in another folder of the item's section is a second Part of the item.
- After the source is deleted, a path refresh of the vacated folder removes the old item, which is right across sections. In the item's own section it destroys the whole item with its surviving Part, and Plex makes it again later under a new ratingKey without collections. A whole-section refresh keeps the item, its ratingKey and its collections.
- On Windows a freshly scanned file stays locked about 1 to 2 s, so the Node's Remove retries a sharing violation with backoff.
- History for a deleted id answers 404, which is no history.
- Editing a section's locations sends name, type, agent, scanner and language with them; a location-only PUT answers 400. Only tests and fakes edit sections.
- Left on Jackson's server: section 4 (empty) and section 5 (91651 and its collections).
