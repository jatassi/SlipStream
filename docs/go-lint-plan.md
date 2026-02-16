# Go Backend: Zero Lint Errors Plan

## Context

The Go backend has **2,324 lint issues** (uncapped) across 21 enabled linters. But the number is misleading: **~660 are noise from misconfigured linter rules** (fieldalignment, err-shadowing), **~1,000 are mechanical transforms** (pass-by-pointer, import formatting), and the remaining **~660 are genuine code quality issues** spanning error handling, complexity, dead code, and security.

The real problems are architectural:
- `api/server.go` (3,360 lines): god struct with 69 injected dependencies
- `librarymanager/service.go` (2,635 lines): orchestrates 5 domains with **zero test coverage**
- `import/pipeline.go`: 273-line mega-function with 6 inlined phases
- Errors are swallowed across the codebase (43 nilerr, 68 errcheck)
- 0 existing `//nolint` directives (clean slate — good)

The goal is not to make linters happy. The goal is idiomatic, maintainable Go code that happens to pass all linters.

---

## Agent Execution Rules

Each phase of this plan will be executed by an AI agent (Claude) in a separate conversation. These rules apply universally.

### Universal Rules

1. **One phase per conversation.** Start fresh. Read this plan for context.
2. **Use Sonnet for subagents** (`model: "sonnet"` on Task tool) unless the task requires deep architectural reasoning (Phase 7D). Use Opus for complex refactoring proposals only.
3. **Never go more than ~50 file edits without running `make test`.** Batch your work.
4. **Line numbers in this plan WILL drift** as earlier phases modify files. Always search by function/variable name, not line number.
5. **Update the progress table** at the end of each phase.
6. **Do not commit** unless the user explicitly asks.
7. **No new packages** unless there's a real encapsulation boundary. File splits within the same package are zero-risk.

### Phase Lifecycle (follow for EVERY phase)

```
1. SNAPSHOT     ./scripts/lint/go-snapshot.sh save phaseN-pre
2. SIGNATURES   ./scripts/lint/go-check-signatures.sh save phaseN-pre
3. EXECUTE      (phase-specific work below)
4. TEST         make test
5. VERIFY LINT  ./scripts/lint/go-snapshot.sh compare phaseN-pre
6. VERIFY API   ./scripts/lint/go-check-signatures.sh compare phaseN-pre
7. FINAL SNAP   ./scripts/lint/go-snapshot.sh save phaseN-post
8. UPDATE PLAN  Edit this file's progress table
```

For step 6: API changes are EXPECTED in Phase 4A (Broadcast signature) and Phase 2 (removing dead code). For all other phases, exported signatures should not change.

### Script Reference

| Script | Purpose | When to Use |
|--------|---------|-------------|
| `./scripts/lint/go-summary.sh` | Total + top linters/files | Start of phase, quick status |
| `./scripts/lint/go-breakdown.sh` | Full per-linter counts | Issue distribution |
| `./scripts/lint/go-breakdown.sh --linter X` | All issues for linter X | Processing a specific linter |
| `./scripts/lint/go-file.sh <path>` | Lint a single file | After editing, verify fixes |
| `./scripts/lint/go-count-linter.sh <name>` | Count for one linter by file | Track progress within a phase |
| `./scripts/lint/go-test-affected.sh` | Test only modified packages | Quick verification after edits |
| `./scripts/lint/go-check-signatures.sh save/compare` | API signature regression | Before/after refactors |
| `./scripts/lint/go-snapshot.sh save/compare` | Lint count regression | Phase boundaries |
| `./scripts/lint/go-verify.sh` | Full vet+build+test+lint | End-of-phase final check |

### Context Management

| Phase | Subagents? | Model | Purpose |
|-------|-----------|-------|---------|
| 2 | No | — | Simple deletions |
| 3 | No | — | Auto-fix + mechanical |
| 4A | Sonnet | — | Find all Broadcaster interfaces + callers |
| 4C | Sonnet | — | Classify nilerr issues (BUG/SOFT_FAIL/RESTRUCTURE) |
| 5 | Sonnet | — | List all functions per struct type for batch processing |
| 6E | Sonnet | — | Group goconst suggestions for constant extraction |
| 7A-E | Sonnet | — | Read large files and map functions → target files |
| 7D | Opus | — | Complex function restructuring proposals |
| 8 | No | — | Localized fixes |

**Avoiding context overflow**:
- Phase 5 (60 files): Process in batches of ~15 files by struct type. Run `go-test-affected.sh` between batches.
- Phase 7 large files (1,500-3,400 lines): Use subagents to read and return only the function-to-file mapping. Don't load entire files in main context.
- Phase 4C (43 nilerr sites): Use a subagent to classify all 43, then process fixes in main context with the classifications only.

### Regression Test Strategy

**Deterministic checks (run after EVERY phase)**:
```bash
go build ./...                                     # Compilation
go vet ./...                                       # Static analysis
make test                                          # Behavioral regressions
./scripts/lint/go-check-signatures.sh compare X    # API surface
./scripts/lint/go-snapshot.sh compare X            # Lint regression
```

**Tests to write BEFORE risky phases**:

- **Before Phase 4C** (nilerr): For each nilerr classified as BUG, check if a test exercises that path. If not, add one that triggers the error and asserts it's returned.
- **Before Phase 7B** (librarymanager split): Package has zero tests. File split is safe, but if reducing complexity (7D), write baseline tests for `ScanRootFolder`, `AddMovie`, `AddSeries`, `RefreshMovieMetadata` first.
- **Before Phase 7D** (complexity reduction): For each function to refactor, check if tests exist. If no → write a minimal test for the main path before refactoring.

### nolint Philosophy

A `//nolint` directive is acceptable ONLY when:
1. The linter is demonstrably wrong (false positive)
2. The lint-clean version would be worse code
3. The justification is documented inline

**Budget: fewer than 15** across the entire codebase. If you're adding more than 3 per phase, stop and reconsider.

Expected candidates:
- `defer tx.Rollback()` — errcheck (rollback after commit is idiomatic)
- Prowlarr TLS skip — gosec G402 (admin-configured endpoint)
- Config key variable names — gosec G101 (names, not credentials)
- Interface-mandated unused params — unparam (required by contract)

---

## Phase 1: Calibrate the Instrument ✅

**What:** Tune `.golangci.yml` so it reports real issues, not noise.
**Issues eliminated:** ~660 (from 2,324 → ~1,664)
**Risk:** None — zero code changes.
**Status:** Complete. Actual count: **1,653**.

### Changes:
1. **Disable `fieldalignment`** in govet (564 issues). This is a desktop app, not an HFT system. Struct padding savings of 8-32 bytes per struct are irrelevant. The churn makes diffs unreadable for zero correctness benefit. Change `govet.enable-all: true` to an explicit list of valuable analyzers (assign, atomic, bools, buildtag, composites, copylocks, errorsas, httpresponse, ifaceassert, loopclosure, lostcancel, nilfunc, printf, shadow, sortslice, stdmethods, stringintconv, structtag, tests, unmarshal, unreachable, unsafeptr).
2. **Keep shadow but exclude `err`-only shadowing** (~95 issues). Redeclaring `err` inside sequential `if` blocks is idiomatic Go. Add shadow exclusion settings or switch to govet shadow with `strict: false`. Keep importShadow (14 issues) — those are real code smells.
3. **Update `go-summary.sh`** to pass `--max-issues-per-linter 0 --max-same-issues 0` so it reports true counts instead of capped-at-50 numbers.

### Files modified:
- `.golangci.yml`
- `scripts/lint/go-summary.sh`

---

## Phase 2: Remove Dead Weight

**What:** Delete unused code, remove unused parameters.
**Issues fixed:** ~35 (unused: 13, unparam: 22)
**Risk:** Low — dead code by definition has no callers.

### Approach

- **unused (13):** Delete dead functions entirely. Key targets:
  - `cmd/fetchmockdata/main.go:361` — unused `prettyJSON`
  - `cmd/slipstream/main.go:444` — unused `waitForPortFree`
  - `internal/filesystem/drives_unix.go:89` — unused `getDarwinVolumeLabels`
  - `internal/import/matching.go:194,197` — unused `dailyPattern`, `animePattern`
  - `internal/import/pipeline.go:857` — unused `handleCleanup`
  - `internal/indexer/search/aggregator.go:445` — unused `matchesMovieCriteria`
  - `internal/prowlarr/client.go:136,240,261,268` — 4 unused methods (`doXML`, `convertCapabilities`, `convertSearchType`, `convertCategories`)
  - `internal/prowlarr/service.go:540` — unused `buildSearchRequest`
  - `internal/library/slots/naming_test.go:10` — unused test helper
- **unparam (22):** For each, either remove the unused parameter or use it. Key targets:
  - `internal/rsssync/settings.go:98` — `loadSettings` always returns nil error → remove error return
  - `internal/update/service.go:545,806` — unused `ctx` parameter → remove or use
  - `internal/prowlarr/client.go:110` — `method` always receives "GET" → inline it

### Agent Execution

1. Run `./scripts/lint/go-breakdown.sh --linter unused` to get all 13 issues with file locations.
2. For each: delete the dead function/variable. If removing causes unused imports, remove those too.
3. Run `./scripts/lint/go-breakdown.sh --linter unparam` to get all 22 issues.
4. For each unparam:

| unparam message | Action |
|----------------|--------|
| "X always receives Y" | Inline Y at the call site, remove the parameter |
| "result N is always nil" | Remove that return value, update ALL callers |
| Function implements an interface | `//nolint:unparam // implements InterfaceName` |

5. After all edits, verify:
```bash
go build ./...
make test
./scripts/lint/go-count-linter.sh unused    # Should be 0
./scripts/lint/go-count-linter.sh unparam   # Should be 0
```

### Files modified: ~15 files across internal/

---

## Phase 3: Auto-fix Mechanical Issues

**What:** Run formatters and apply deterministic transforms.
**Issues fixed:** ~170
**Risk:** Minimal — tooling-applied, semantically equivalent.

### Approach

1. Run `golangci-lint run --fix` for goimports (101), regexpSimplify (65)
2. Manually fix: usestdlibvars (6), misspell (1), octalLiteral (20) — use `0o644` not `0644`
3. Review the diff — auto-fix should only touch imports, regex simplification, and literal formatting

### Agent Execution

1. **Auto-fix** (handles goimports + regexpSimplify):
   ```bash
   make lint-fix
   ```
   Review the git diff. Should only touch import grouping and regex simplification. If anything else changed, investigate.

2. **Octal literals** (20 issues): Get the list:
   ```bash
   ./scripts/lint/go-breakdown.sh --linter gocritic 2>&1 | grep 'octalLiteral'
   ```
   Replace `0644` → `0o644`, `0755` → `0o755`, `0600` → `0o600`, etc. Use Edit with `replace_all` per file.

3. **usestdlibvars** (6): Run `./scripts/lint/go-count-linter.sh usestdlibvars`, fix each (e.g., `200` → `http.StatusOK`).

4. **misspell** (1): Run `./scripts/lint/go-count-linter.sh misspell`, fix the typo.

5. Verify:
```bash
make test
./scripts/lint/go-count-linter.sh goimports          # Should be 0
./scripts/lint/go-count-linter.sh usestdlibvars       # Should be 0
./scripts/lint/go-count-linter.sh misspell             # Should be 0
# regexpSimplify and octalLiteral are under gocritic:
./scripts/lint/go-breakdown.sh --linter gocritic 2>&1 | grep -cE 'regexpSimplify|octalLiteral'  # Should be 0
```

---

## Phase 4: Error Handling Discipline

**What:** Fix error swallowing, wrapping, and comparison. This is correctness work.
**Issues fixed:** ~210 (errcheck: 68, nilerr: 43, errorlint: 78, nilnil: 21)
**Risk:** Medium — changes control flow. Must review each case.

**Sub-phase execution order: 4A → 4B → 4C → 4D → 4E. Run `make test` between each.**

### 4A: Change Hub.Broadcast signature (kills 11+ errcheck violations)

The `Hub.Broadcast()` in `internal/websocket/hub.go` returns an error only on JSON marshal failure. Every caller ignores it with `_ =`. The right fix: make Broadcast not return error; log the marshal failure internally.

```go
// Before:
func (h *Hub) Broadcast(msgType string, payload interface{}) error { ... }
// After:
func (h *Hub) Broadcast(msgType string, payload interface{}) { ... }
```

This eliminates errcheck violations in: `import/pipeline.go`, `import/handlers.go`, `indexer/grab/service.go`, `indexer/search/service.go`, `autosearch/scheduled.go`, `logger/broadcaster.go`, and all Broadcaster interface implementations.

#### Agent Execution (4A)

This is a cross-cutting change: 1 implementation, 10 interface definitions, ~55 call sites.

1. Save pre-state: `./scripts/lint/go-check-signatures.sh save phase4a-pre`
2. Modify `internal/websocket/hub.go`: Remove error return from `Broadcast`. Log marshal errors internally via the Hub's logger.
3. Update ALL Broadcaster interface definitions. There are 10 across the codebase — use a Sonnet subagent to find them:
   ```
   Search for 'Broadcast(msgType string' or 'Broadcast(eventType string' in interface definitions
   ```
   Known locations:
   - `internal/logger/broadcaster.go`
   - `internal/autosearch/service.go`
   - `internal/indexer/grab/service.go`
   - `internal/indexer/search/service.go`
   - `internal/history/service.go`
   - `internal/downloader/broadcaster.go`
   - `internal/portal/requests/events.go`
   - `internal/health/service.go`
   - `internal/notification/mock/notifier.go` (already has no error return — verify)

   **Do NOT touch** `internal/update/service.go`'s `Broadcaster` — it has a different method (`BroadcastUpdateStatus`).

4. Update all ~55 call sites: Remove `_ =`, `err =`, or error handling around Broadcast calls.
5. Verify:
   ```bash
   go build ./...    # Catches any missed callers — won't compile until ALL are fixed
   make test
   ```

### 4B: errcheck (remaining ~57)

Triage each unchecked error:
- **Type assertions** (`internal/auth/passkey.go:97`): Add comma-ok pattern
- **Deferred rollbacks** (`internal/library/rootfolder/service.go:255`): `defer tx.Rollback()` is fine — add `//nolint:errcheck // rollback after commit is a no-op`
- **Notification status updates** (`internal/notification/service.go:441,459,469`): These are fire-and-forget DB writes. Log the error instead of ignoring.
- **os.Chmod** (`internal/library/organizer/organizer.go:137`): Check and log the error
- **json.Unmarshal** (`internal/notification/service.go:474`): This can actually fail — check it

#### Agent Execution (4B)

Run `./scripts/lint/go-breakdown.sh --linter errcheck` to get the full list after 4A.

**Decision rules**:

| Pattern | Action |
|---------|--------|
| `defer tx.Rollback()` | `//nolint:errcheck // rollback after commit is no-op` |
| Type assertion `val.(*Type)` without comma-ok | `val, ok := x.(*Type); if !ok { return ... }` |
| `json.Unmarshal(...)` unchecked | Check and return/log the error |
| `os.Chmod(...)` unchecked | Check and log |
| `go func() { ... }` ignoring error | Log inside the goroutine |
| Notification DB writes | Log: `if err := ...; err != nil { s.logger.Error().Err(err)...}` |
| `go artwork.Download(...)` in goroutine | Log inside goroutine if not already |

### 4C: nilerr (43 issues)

These are functions where `err != nil` is detected but `nil` is returned. Each must be triaged:
- **Bug** (return the error): Most cases — the function found an error and silently swallowed it
- **Intentional soft failure** (log and continue): For non-critical operations like artwork downloads
- For intentional cases, restructure so nilerr doesn't fire (return early before the error check, or use a different control flow)

#### Agent Execution (4C)

**This is the riskiest sub-phase.** Use a Sonnet subagent to read each nilerr site and classify it:

```
For each nilerr issue, read the surrounding function and classify:
- BUG: The error should be returned to the caller
- SOFT_FAIL: The operation is non-critical (artwork, notifications) — log and continue
- RESTRUCTURE: The control flow needs reworking to satisfy the linter
```

**Decision rules**:

| Context | Classification | Fix |
|---------|---------------|-----|
| Error during DB query in CRUD | BUG | Return the error |
| Error downloading artwork/poster | SOFT_FAIL | Log warning, continue |
| Error in cleanup/defer path | SOFT_FAIL | Log warning |
| Error parsing optional field | SOFT_FAIL | Log debug, use zero value |
| `if err != nil { return nil, nil }` | RESTRUCTURE | Return `nil, err` or sentinel |

For SOFT_FAIL cases, restructure so the linter doesn't fire — typically by extracting the try-and-log pattern into its own block or using early return before the error-nil return.

**Regression test requirement**: For nilerr cases classified as BUG in packages with existing tests (`import`, `autosearch`, `movies`, `tv`), add a test that triggers the error and asserts it's now returned.

### 4D: errorlint (78 issues)

Three sub-categories:
1. `fmt.Errorf("...: %s", err)` → `fmt.Errorf("...: %w", err)` — enables proper unwrapping
2. `err == sql.ErrNoRows` → `errors.Is(err, sql.ErrNoRows)` — wrapping-safe comparison
3. Type assertions in error chains → `errors.As()` where appropriate

#### Agent Execution (4D)

Mechanical transforms. Process all `%w` fixes first, then all `errors.Is`, then all `errors.As`.

Exception: If a function intentionally strips the error chain (creating a new error), keep `%s` and add a comment explaining why.

### 4E: nilnil (21 issues)

Functions returning `(nil, nil)`. Introduce sentinel errors or restructure so callers can distinguish "not found" from "success."

#### Agent Execution (4E)

For each:
- "Not found" case → return `sql.ErrNoRows` or a package-level `ErrNotFound` sentinel
- "Nothing to do" case → return typed zero value (empty slice, zero struct) instead of nil
- If callers already check `result == nil` as "not found" → introduce sentinel and update callers

### Phase 4 Verification

```bash
make test
./scripts/lint/go-count-linter.sh errcheck     # Should be ≤3 (nolint'd rollbacks)
./scripts/lint/go-count-linter.sh nilerr        # Should be 0
./scripts/lint/go-count-linter.sh errorlint     # Should be 0
./scripts/lint/go-count-linter.sh nilnil         # Should be 0
```

### Files modified: ~40 files across internal/

---

## Phase 5: Pointer & Value Semantics

**What:** Pass large structs by pointer, fix range copies.
**Issues fixed:** ~476 (hugeParam: 369, rangeValCopy: 107)
**Risk:** Low but widespread — pointer semantics change mutability guarantees.

### Approach

- **hugeParam:** Change function signatures from value to pointer receivers/params. The dominant struct is `zerolog.Logger` (112 bytes) — many functions pass it by value. Other large structs: `ParsedMedia`, `sqlc` generated params.
- **rangeValCopy:** Change `for _, item := range items` to `for i := range items` with `item := &items[i]` for large structs.

### Watch-outs
- Changing params to pointers means callers can now mutate the original. Review each call site.
- For `zerolog.Logger`: it's safe to pass by pointer — the logger is designed for it.
- For sqlc-generated structs: these are typically created locally and passed once — pointer conversion is safe.

### Agent Execution

**Strategy: Process by STRUCT TYPE, not by file.** This groups related changes and makes mutation analysis manageable.

The gocritic sub-check breakdown is:
- `zerolog.Logger` (~88 issues) — safe to pointer, designed for it
- `zerolog.Event` (~84 issues) — method chains, fine by pointer
- Various `criteria` structs (~33 issues) — check for mutation
- `SearchableItem`, `ParsedMedia`, sqlc params, config structs, etc.

**Step-by-step**:

1. **zerolog.Logger first** (biggest batch). Use a Sonnet subagent:
   ```
   Find all functions taking zerolog.Logger by value (not pointer).
   Search: 'func.*zerolog\.Logger[^*]' in *.go excluding _test.go
   ```
   Change each to `*zerolog.Logger`. Update callers.

2. **rangeValCopy** (107 issues). For each:
   ```go
   // Before
   for _, item := range items { use(item) }
   // After
   for i := range items { item := &items[i]; use(item) }
   ```
   **CAUTION**: If the loop body appends to `items` or modifies the slice, taking a pointer is unsafe. Check for slice mutation inside loops.

3. **Other hugeParam structs**. For each remaining struct type:
   - If the function only reads → change to pointer (safe)
   - If the function modifies → it SHOULD already be a pointer, verify callers

**Batch testing**: After each struct type batch:
```bash
./scripts/lint/go-test-affected.sh    # tests only modified packages
```

Full suite at the end:
```bash
make test
./scripts/lint/go-breakdown.sh --linter gocritic 2>&1 | grep -cE 'hugeParam|rangeValCopy'  # Should be 0
```

### Files modified: ~60 files (widespread but mechanical)

---

## Phase 6: Style & Idioms ✅

**What:** Apply remaining style fixes.
**Issues fixed:** 254 (732 → 478). Key results: gocritic 179→3, revive 29→1, errorlint 14→0, staticcheck 9→0, goimports 5→0, usestdlibvars 2→0, goconst 57→37.
**Risk:** Low — localized transforms.
**Status:** Complete. All tests pass, build clean, 95 API signature changes (all paramTypeCombine/hugeParam/unnamedResult improvements).

### Approach

- **paramTypeCombine (54):** `func(a string, b string)` → `func(a, b string)`
- **httpNoBody (44):** `http.NewRequestWithContext(ctx, method, url, nil)` → `..., http.NoBody)`
- **ifElseChain (26):** Convert to switch statements
- **importShadow (14):** Rename local variables that shadow package imports (e.g., `status` → `currentStatus` when `status` package is imported)
- **goconst (56):** Extract repeated string literals to package-level constants. Key targets: status strings (`"failed"`, `"missing"`, `"available"`, `"upgradable"`), media types (`"movie"`, `"episode"`)
- **revive (31):** Follow Go conventions per configured rules
- **staticcheck (31):** Fix each individually — deprecated APIs, unnecessary conversions, unreachable code
- **Remaining:** unnamedResult, filepathJoin, redundantSprint, equalFold, emptyStringTest, etc.

### Agent Execution

Process by sub-category in this order:

#### 6A: paramTypeCombine (54)
Purely syntactic: `func(a string, b string)` → `func(a, b string)`. No behavioral change.

#### 6B: httpNoBody (44)
`http.NewRequestWithContext(ctx, method, url, nil)` → `..., http.NoBody)`. Mechanical.

#### 6C: ifElseChain (26)
Convert `if/else if/else if` to `switch`. Read each one — watch for side effects in conditions.

#### 6D: importShadow (14)
Rename local variables: `status` → `currentStatus`, `history` → `historyEntry`, etc.

#### 6E: goconst (56)
Use a Sonnet subagent to run `./scripts/lint/go-breakdown.sh --linter goconst` and group the suggestions. **Be selective**:
- Status strings (`"failed"`, `"missing"`, `"available"`) → YES
- Media types (`"movie"`, `"episode"`) → YES
- Error messages → NO (unique context, leave inline)

#### 6F: revive (31) + staticcheck (31)
Fix individually per linter message.

#### 6G: Remaining
`unnamedResult` (11), `filepathJoin` (8), `redundantSprint` (6), `equalFold` (5), `emptyStringTest` (3), etc.

### Verification

```bash
make test
./scripts/lint/go-summary.sh  # Should show ~593 remaining (mostly gocognit/gocyclo/nestif)
```

---

## Phase 7: Architectural Simplification

**What:** Break apart god functions and god files. This is the highest-value phase.
**Issues fixed:** ~250+ (gocognit: 181, gocyclo: 69, nestif: 70, funlen: 4, minus those fixed by earlier phases)
**Risk:** Medium-high. Mitigated by tests and the file-level (not package-level) split approach.

### Philosophy

Rob Pike: "A package is an API boundary." We split *files*, not *packages*, unless there's a genuine encapsulation boundary. Moving code between files within the same package is a zero-risk refactor — no import changes, no interface changes, no API changes.

### 7A: Split api/server.go (3,360 lines → 6 files, same package)

The file mixes routing, handlers, dev mode, and adapters. But the real problem isn't file length — it's that the Server struct has 69 dependencies. We address both:

| New File | What Moves | Est. Lines |
|----------|-----------|------------|
| `server.go` | Server struct, NewServer, Start, Shutdown, Set* methods | ~300 |
| `server_init.go` | InitializeNetworkServices and all service wiring (currently lines 230-740) | ~500 |
| `routes.go` | setupRoutes, setupMiddleware, setupPortalRoutes | ~400 |
| `handlers_system.go` | Status, settings, restart, firewall, API key handlers | ~300 |
| `handlers_downloads.go` | Download client, queue, indexer history handlers | ~350 |
| `adapters.go` | All adapter structs bridging between packages (~15 types) | ~500 |
| `devmode.go` | All dev mode switch/copy/populate/mock methods (~17 functions) | ~750 |

No new packages. No interface changes. Just file organization.

#### Agent Execution (7A)

1. `./scripts/lint/go-check-signatures.sh save phase7a-pre`
2. Use a Sonnet subagent to read `internal/api/server.go` and produce a function → target file mapping. The subagent should return a list like: `FuncName → server_init.go`.
3. Create each new file. Write the `package api` declaration + required imports + the moved functions.
4. After EACH file extraction, verify: `go build ./internal/api/...`
5. After all splits:
   ```bash
   make test
   ./scripts/lint/go-check-signatures.sh compare phase7a-pre  # No changes expected
   ```

### 7B: Split librarymanager/service.go (2,635 lines → 6 files, same package)

**IMPORTANT:** This file has zero test coverage. Before splitting, we need to either (a) add baseline tests for the public methods, or (b) accept the risk given that the split is purely organizational (same package, same types, no logic changes).

| New File | What Moves |
|----------|-----------|
| `service.go` | Service struct, NewService, Set* methods, getDefaultQualityProfile |
| `scanning.go` | ScanRootFolder, ScanSingleFile, scanMovieFolder, scanSeriesFolder, scan state management |
| `matching.go` | matchOrCreateMovie, matchOrCreateSeries, matchUnmatchedMovies, matchUnmatchedSeries |
| `creation.go` | createMovieFromParsed, addMovieFile, createSeriesFromParsed, addEpisodeFile, getOrCreateEpisode, ensureSeasonExists |
| `metadata_refresh.go` | RefreshMovieMetadata, RefreshSeriesMetadata, RefreshAll*, downloadPendingArtwork |
| `add.go` | AddMovie, AddSeries, applyMonitoringOnAdd, triggerSeriesSearchOnAdd |

#### Agent Execution (7B)

Same approach as 7A. Use a Sonnet subagent to read the file and map functions. Build-test after each extraction.

### 7C: Extract phases from processImport (273 lines → ~30 lines orchestrator + 6 phase methods)

The `processImport` function in `internal/import/pipeline.go` has 6 clear phases inlined:

```go
func (s *Service) processImport(ctx context.Context, job ImportJob) (*ImportResult, error) {
    result := &ImportResult{SourcePath: job.SourcePath}
    settings, err := s.loadSettings(ctx)
    if err != nil { return result, err }

    if err := s.validateImportFile(ctx, job, settings); err != nil { ... }
    match, err := s.resolveMatch(ctx, job, settings)
    if err != nil { ... }
    if err := s.evaluateQualityAndUpgrade(ctx, job, match, result); err != nil { ... }
    destPath, err := s.planDestination(ctx, match, job, settings, result)
    if err != nil { ... }
    return s.executeAndFinalize(ctx, job, match, destPath, result)
}
```

Each extracted method stays in pipeline.go (or a new `pipeline_phases.go`). Same package, same struct. The 273-line function becomes a 30-line orchestrator calling 6 focused 30-50 line methods.

#### Agent Execution (7C)

The example above is illustrative. Read the actual `processImport` function, identify real phase boundaries, name methods accordingly. Keep extracted methods on the same `*Service` receiver.

### 7D: Flatten remaining complex functions ✅

**Status:** Complete. 478 → 22 (-456 issues across 7D + 7A-E file splits + Phase 8 mechanical fixes). Key results: gocognit 178→2, gocyclo 73→0, nestif 68→0, funlen 4→0, goconst 37→4, gosec 7→0, noctx 5→0, thelper 4→0, unparam 9→0.

**Files refactored:**
- `slots/migration.go` (16→0): Extracted matchFileToSlot, resolveOneAssignment, evaluateAndResolveFiles, groupEpisodeFiles, buildEpisodePreview, buildSeasonPreview, tallyFileSummary, migrateMovieFiles, migrateEpisodeFiles, reevaluateSlotFiles, loadReviewFileInfo, populateDetectedAttributes, buildSlotOptions
- `slots/debug.go` (15→0): Extracted resolutionScore, sourceScore, attributeMode, collectProfileValues, mismatchReason, buildProfileMatchOutput, evaluateSlotForImport, determineImportAction, previewMovie, previewTVShow, previewSeason, previewEpisode, accumulateFileSummary
- `slots/assignment.go` (3→0): Extracted evaluateSlotForRelease, sortAssignmentsByPriority, countMatchingSlots, buildSlotWithProfile
- `slots/status.go` (6→0): Extracted buildSlotStatus, buildEpisodeSlotStatus, initializeMovieSlot, initializeEpisodeSlot, priority map
- `slots/service.go` (2→0): Extracted getMovieSlotFileID, getEpisodeSlotFileID, validateSlotProfiles, checkSlotExclusivity
- `autosearch/service.go` (3→0): Extracted buildGrabRequest, findBestRelease, grabAndReport
- Various other files: removed dead code, fixed nilerr, unparam, rangeValCopy across filesystem, import, prowlarr packages

**Note:** gocritic increased 3→21 as refactoring surfaced new paramTypeCombine/hugeParam/importShadow issues in extracted helpers. These are Phase 6-type mechanical fixes.

For each remaining gocognit/gocyclo/nestif violation after the splits:
1. **Early returns** — invert conditions, return early
2. **Extract helpers** — named functions for repeated patterns
3. **Table-driven logic** — for parser.go pattern matching and notification embed building
4. **Strategy pattern** — only where there's a real polymorphic boundary (not just to reduce line count)

Key files needing attention:
- `internal/library/scanner/parser.go` (71 lint issues, 553 lines) — regex-heavy parser with many branches
- `internal/indexer/cardigann/search.go` (818 lines) — complex search logic
- `internal/autosearch/scheduled.go` (992 lines) — TV upgrade collection logic
- `internal/notification/email/notifier.go` (20 lint issues) — HTML template building
- `internal/rsssync/matcher.go` (14 lint issues) — release matching logic

#### Agent Execution (7D)

**This is where regressions happen.** For each function:
1. Read the function
2. Identify the minimal simplification: early returns? extract helper? table-driven?
3. Apply the smallest change to satisfy the linter
4. Test immediately: `./scripts/lint/go-test-affected.sh`

**For parser.go**: Table-driven approach — define `[]struct{pattern, handler}` pairs. Use a Sonnet subagent to read the file and propose the table structure.

**For files with zero test coverage**: Only apply SAFE transforms (early returns, extract methods in same file). Do NOT restructure logic.

**Use Opus subagent** for complex restructuring proposals where understanding multiple interacting code paths is needed.

### 7E: Split tv/service.go (1,579 lines → 3 files, same package)

| New File | What Moves |
|----------|-----------|
| `service.go` | Service struct, series CRUD, Count, GetStatus |
| `episodes.go` | Episode CRUD, BulkMonitorEpisodes, file management methods |
| `seasons.go` | ListSeasons, UpdateSeasonMonitored, monitoring stats |

#### Agent Execution (7E)

Same approach as 7A/7B.

### Phase 7 Verification

```bash
make test
./scripts/lint/go-verify.sh
./scripts/lint/go-check-signatures.sh compare phase7a-pre  # No API changes
./scripts/lint/go-summary.sh  # Should show ~73 remaining
```

---

## Phase 8: Security & Reliability ✅

**What:** Fix security findings and add missing contexts.
**Issues fixed:** Folded into Phase 7D+ work. gosec 7→0, noctx 5→0, thelper 4→0, unparam 9→0.
**Risk:** Low-medium — each fix is localized.
**Status:** Complete. All gosec, noctx, thelper, and unparam issues resolved during extended complexity reduction pass.

### Approach

- **noctx (24):** Replace `http.NewRequest` with `http.NewRequestWithContext` in notification clients and download clients. Thread context from the caller.
- **gosec (34):**
  - File permissions (G301/G302): Tighten `0777` → `0750`, `0666` → `0644`
  - Integer overflow (G115): Add bounds checking in `auth/passkey.go`
  - Weak TLS (G402): Fix in email notifier; Prowlarr intentional skip gets `//nolint:gosec // admin-configured endpoint`
  - Hardcoded credential names (G101): These are config key names, not credentials → `//nolint:gosec // variable name, not a credential`
- **thelper (8):** Add `t.Helper()` to test helper functions

### Agent Execution

#### 8A: noctx (24)
Run `./scripts/lint/go-breakdown.sh --linter noctx`. For each: replace `http.NewRequest(...)` with `http.NewRequestWithContext(ctx, ...)`. Thread `ctx` from the caller — every handler has `r.Context()`, every service method should accept `ctx context.Context`.

#### 8B: gosec (34)
Run `./scripts/lint/go-breakdown.sh --linter gosec`. Process by rule:

| Rule | Fix |
|------|-----|
| G301/G302 | `0777` → `0o750`, `0666` → `0o644` |
| G115 | Bounds check before int conversion |
| G402 | Fix TLS config; Prowlarr: `//nolint:gosec // admin-configured endpoint` |
| G101 | `//nolint:gosec // variable name, not a credential` |

#### 8C: thelper (8)
Add `t.Helper()` as first line of test helper functions. Mechanical.

### Verification

```bash
make test
./scripts/lint/go-count-linter.sh gosec    # Should be ≤3 (nolint'd)
./scripts/lint/go-count-linter.sh noctx    # Should be 0
./scripts/lint/go-count-linter.sh thelper  # Should be 0
```

---

## Progress Tracking

| Phase | Description | Est. Issues Fixed | Running Total | Risk | Status |
|-------|-------------|------------------|---------------|------|--------|
| 1 | Calibrate linter config | ✅ 671 | **1,653** | None | ✅ Done |
| 2 | Remove dead code | ✅ 30 | **1,623** | Low | ✅ Done |
| 3 | Auto-fix mechanical | ✅ 353 | **1,270** | Minimal | ✅ Done |
| 4 | Error handling discipline | ✅ 136 | **1,134** | Medium | ✅ Done |
| 5 | Pointer & value semantics | ✅ 402 | **732** | Low | ✅ Done |
| 6 | Style & idioms | ✅ 254 | **478** | Low | ✅ Done |
| 7D | Flatten complex functions (initial) | ✅ 57 | **421** | Medium-High | ✅ Done |
| 7A-C,E | File splits (server, libmgr, pipeline, tv) | ✅ ~100 | ~321 | Medium | ✅ Done |
| 7D+ | Extended complexity reduction + Phase 8 fixes | ✅ 299 | **22** | Medium-High | ✅ Done |
| Final | Remaining stragglers | ✅ 22 | **0** | Low | ✅ Done |

**Zero lint issues achieved.** Next step: add `golangci-lint run` as a blocking CI check.

### Estimated Linter Counts After Each Phase

Use these to validate progress. Significant deviation means missed fixes or incidental fixes from earlier phases.

| After Phase | Expected Total | Key Linters at Zero |
|-------------|---------------|---------------------|
| 2 | ~1,618 | unused, unparam |
| 3 | ~1,448 | + goimports, regexpSimplify, octalLiteral, usestdlibvars, misspell |
| 4 | ~1,238 | + errcheck (≤3), nilerr, errorlint, nilnil |
| 5 | ~762 | + hugeParam, rangeValCopy |
| 6 | ~582 | + goconst, revive, staticcheck, paramTypeCombine, httpNoBody, importShadow |
| 7 | ~62 | gocognit ≤30, gocyclo ≤20, nestif ≤10 remain |
| 8 | ~0 | + gosec (≤3), noctx, thelper |

---

## Execution Notes

- **Each phase = separate conversation.** Don't accumulate drift.
- **Run `make test` after every sub-phase.** Never go more than ~50 file changes without testing.
- **Phase 7 is the hard one.** Everything else is mechanical. Budget the most time here.
- **librarymanager has zero tests.** The Phase 7B file split is safe (same package, no logic changes), but any complexity reduction in those functions must be done with extra care.
- **Don't create new packages unless there's a real encapsulation boundary.** File splits within the same package are zero-risk. New packages create import cycles, API surfaces, and coupling.
