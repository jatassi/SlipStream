# Fix: Queue-Triggered Imports Bypass Module System

## Problem

Queue-triggered imports (auto-search, portal requests) never use the module-aware matching/naming path. They always fall through to legacy code, which means:

- `ModuleEntity` is never set on the `LibraryMatch`
- `computeDestinationViaModule` is never reached
- Naming relies on legacy `buildTokenContext` instead of module token data
- Both TV and movie imports are affected

Production evidence: every single import in the logs shows `Orphan file claimed by module` because the module system is only invoked via the parse-based fallback, never via the queue match.

## Root Cause

`resolveLibraryMatch` in `internal/import/pipeline.go` calls `matchToLibraryWithSettings` directly, bypassing `matchToLibrary` which is the only entry point that tries `matchToLibraryViaModule`.

### Queue-triggered path (broken)
```
processCompletedEntry → ProcessCompletedDownload → queueFilesForImport
→ processJob → processImport → prepareImport → resolveLibraryMatch
→ matchToLibraryWithSettings  ← never tries module-aware matching
```

`matchToLibraryWithSettings` uses legacy `matchFromMapping` which creates a `LibraryMatch` without `ModuleEntity`. Then `computeDestination` (line ~712) checks `match.ModuleEntity != nil`, finds it nil, falls through to legacy `buildTokenContext`.

### Manual import path (works)
```
handlers.go → matchToLibrary → matchToLibraryViaModule → moduleEntityToLibraryMatch
```

`matchToLibrary` (`internal/import/matching.go:28`) correctly tries the module path first when `mapping.ModuleType` is set.

## Key Files

| File | Role |
|---|---|
| `internal/import/pipeline.go` | `resolveLibraryMatch` (line ~483), `computeDestination` (line ~703) |
| `internal/import/matching.go` | `matchToLibrary` (line 28), `matchToLibraryWithSettings` (line 40), `matchFromMapping` (line 173), `resolveMatchConflict` (line 76), `enrichQueueMatch` (line 154) |
| `internal/import/module_matching.go` | `matchToLibraryViaModule` (line 13), `moduleEntityToLibraryMatch` (line 76) |

## Fix

`resolveLibraryMatch` should use the module-aware path when the mapping has a `ModuleType`. The simplest approach: have it call `matchToLibrary` (which already handles the module dispatch + legacy fallback) instead of `matchToLibraryWithSettings` directly.

The `matchToLibraryViaModule` path produces a `LibraryMatch` with `ModuleEntity` set, which makes `computeDestination` use `computeDestinationViaModule` — the correct path that uses module token data (including `SeriesYear`).

`enrichQueueMatch` should also propagate `ModuleEntity` from the parsed match to the queue match when available, so that even when `resolveMatchConflict` prefers the queue match, the module entity data isn't lost.

## Legacy Code to Remove After Fix

Once queue imports go through the module path, the following legacy code in `pipeline.go` becomes dead:

- `buildTokenContext` (line ~929)
- `applySeriesContext` (line ~1040)
- `applyMovieContext` (line ~1030)
- `applyParsedAttributes` (line ~956)
- `applyMediaInfo` (line ~1002)
- `computeEpisodeDestination` legacy path (line ~746)
- `computeMovieDestination` legacy path (line ~725)

And in `matching.go`:
- `matchFromMapping` (line 173) — replaced by `matchToLibraryViaModule`
- `populateMovieMatch`, `populateEpisodeMatch`, `populateSeasonMatch`
- `matchTVFromParse`, `matchMovieFromParse` and all their helpers
- `searchSeries`, `searchMovie`, `cleanTitle`, `calculateTitleSimilarity`, etc.
