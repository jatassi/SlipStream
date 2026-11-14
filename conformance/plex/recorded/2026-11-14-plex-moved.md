# Plex moved-file recorded run, 2026-11-14 (S35 (Media Servers), #160)

Copied from Jackson's media server (C:\Users\jacks\.slipstream-recorded-runs\plex-moved\plex-moved-run.md; the full jsonl stays there, 192 lines, many bodies omitted). Plex Media Server 1.43.5.11029 on Windows, throwaway section 3 over C:\Users\jacks\.slipstream-recorded-runs\media, registry token with Jackson's approval.

Server prefs (read only): autoEmptyTrash true, FSEventLibraryUpdatesEnabled true, FSEventLibraryPartialScanEnabled true, ScheduledLibraryUpdatesEnabled true (900 s). Plex did not pick up new folders by itself within 20 s.

| Case | Result |
|---|---|
| A. Backslash partial refresh `refresh?path=C:\...\Metropolis (1927)` | Works, item on first poll (~1 s). Forward slashes also work; Plex stores backslashes either way. |
| B. Same-folder rename, one folder refresh | Same ratingKey, Part points at new name, old part gone, viewCount and collection kept. |
| C. Cross-folder move, destination refresh only | Old item 404 at once; new ratingKey holds the file with viewCount and collection carried. Nothing lingers after 60 s. |
| D. Destination then source | Same as C; source refresh changed nothing. |
| E. Source first | Destructive: source refresh deletes the item (autoEmptyTrash); new item has no viewCount, no collection (#101 confirmed). |
| F. Sidecar removed, `PUT /library/metadata/{rk}/refresh` | Subtitle stream gone within ~1 s. |
| G. Whole-section refresh | `refreshing` true then false 0.25 s later. |

Confound: C/D/E destination folders had `{edition-Test}`, which makes a separate Plex item on its own, so whether a plain cross-folder move keeps the ratingKey is open (follow-up not run).

Plex crashed 35 s after `DELETE /library/sections/3` while the recorder removed the section's folder (FS-event watcher queued section 3 just before CrashUploader). Down ~1 h 20 min until Jackson asked for a restart (06:25Z); sections unchanged.

## What this means for the build
- Plex moved change: refresh the destination first; destination then source is safe; never source first.
- Do not trust ratingKey across folders: look the item up again by file path after the destination refresh. "Kept" = the item at the new path carries the old watch state/collections, not the same ratingKey.
- Same-folder rename: one partial refresh keeps the item.
- Backslash refresh paths work on Windows Plex.
- Sidecar removed: item refresh works.
- Scan-end polling needs a short interval (0.25 s on a tiny section).
- SlipStream never deletes Plex sections; test tooling must not remove a watched folder right after deleting its section.

Example bodies: `POST /library/sections?...` → 201 `{"MediaContainer":{"size":1,"Directory":[{"key":"3","type":"movie",...,"Location":[{"id":9,"path":"C:\\Users\\jacks\\.slipstream-recorded-runs\\media"}]}]}}`; `GET /library/metadata/{rk}` → 200 JSON with `Metadata[].ratingKey, guid, librarySectionID`; after move 404 HTML `<h1>404 Not Found</h1>`; refresh, scrobble, item refresh, collection PUT → 200 empty body; `DELETE /library/sections/3` → 200 in 1.8 s.
