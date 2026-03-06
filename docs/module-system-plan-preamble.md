# Module System Implementation Plan

> Companion to `docs/module-system-spec.md` and `docs/module-system-plan-structure.md`.
> This plan is designed for agent execution. Each task includes precise file paths, code-level guidance, and validation criteria.

---

## Agent Context Management Instructions

**Primary goal:** Keep your main context window lean. Use subagents for all file-touching work; reserve the main context for orchestration and verification.

### Delegation Strategy

1. **Subagent policy:** Launch Sonnet subagents for all code changes. Each subagent gets:
   - The specific task description from this plan (copy verbatim)
   - The relevant file paths listed in the task
   - Any "depends on" references so it can read prior work
   - Instruction: "Do NOT use git stash or any commands that affect the entire worktree"

2. **Parallelism:** Tasks within a phase that have no dependency arrows (`→`) can run in parallel. The plan marks these explicitly.

3. **Validation after each task:** After a subagent completes, do a lightweight check:
   - Read the key output files (not all files — just the ones listed under "Verify")
   - Run the specified lint/compile command
   - If the check fails, resume the same subagent with the error

4. **Validation after each phase:** Run `make build && make test && make lint` after completing all tasks in a phase. Fix any failures before proceeding.

5. **Context hygiene:** Do NOT read large files (service.go, wire_gen.go) into main context. Delegate reads to subagents. Only read small files (<100 lines) for verification.

---

## Current Codebase Reference

### Key Paths

| Area | Path |
|---|---|
| Wire DI | `internal/api/wire.go`, `internal/api/wire_gen.go` (generated) |
| Service groups | `internal/api/service_groups.go` |
| Switchable services | `internal/api/switchable.go` |
| Circular deps | `internal/api/setters.go` |
| Contracts | `internal/domain/contracts/contracts.go` |
| Movie service | `internal/library/movies/service.go` |
| Movie model | `internal/library/movies/movie.go` |
| TV service | `internal/library/tv/service.go` |
| Status constants | `internal/library/status/status.go` |
| Root folder service | `internal/library/rootfolder/service.go` |
| Quality service | `internal/library/quality/` |
| Slots service | `internal/library/slots/` |
| Database manager | `internal/database/manager.go` |
| Database connection | `internal/database/database.go` |
| Migrations | `internal/database/migrations/*.sql` |
| sqlc config | `sqlc.yaml` |
| SQL queries | `internal/database/queries/*.sql` |
| Scheduler tasks | `internal/scheduler/tasks/*.go` |
| History types | `internal/history/types.go` |
| Decisioning types | `internal/decisioning/types.go` |
| MediaType defs | `internal/decisioning/types.go`, `internal/history/types.go`, `internal/defaults/service.go`, `internal/metadata/artwork.go` |

### Current Schema (Final State After All 69 Migrations)

**Shared tables with media-type discrimination:**

| Table | Discrimination Pattern | Current Constraint |
|---|---|---|
| `root_folders` | `media_type` column | `CHECK (media_type IN ('movie', 'tv'))` |
| `downloads` | `media_type` + `media_id` columns | `CHECK (media_type IN ('movie', 'episode'))` |
| `history` | `media_type` + `media_id` columns | `CHECK (media_type IN ('movie', 'episode', 'season'))` |
| `autosearch_status` | `item_type` + `item_id` columns | `CHECK (item_type IN ('movie', 'episode', 'series'))` |
| `requests` | `media_type` column | `CHECK (media_type IN ('movie', 'series', 'season', 'episode'))` |
| `download_mappings` | Nullable FKs: `movie_id`, `series_id`, `episode_id` | No `media_type` column |
| `queue_media` | Nullable FKs: `movie_id`, `episode_id` | No `media_type` column |
| `import_decisions` | `media_type` + `media_id` columns | Unconstrained TEXT |
| `notifications` | Hard-coded event columns | `on_movie_added`, `on_series_added`, etc. |
| `version_slots` | Per-module FK columns | `movie_root_folder_id`, `tv_root_folder_id` |
| `movie_slot_assignments` | Dedicated table per module | FK to `movies(id)` |
| `episode_slot_assignments` | Dedicated table per module | FK to `episodes(id)` |
| `quality_profiles` | None (shared) | No `module_type` column |
| `portal_users` | Per-module FK columns | `movie_quality_profile_id`, `tv_quality_profile_id` |
| `portal_invitations` | Per-module FK columns | `movie_quality_profile_id`, `tv_quality_profile_id` |
| `user_quotas` | Per-module columns | `movies_limit`, `seasons_limit`, `episodes_limit`, etc. |

**Module-owned tables:**
- Movie: `movies`, `movie_files`
- TV: `series`, `seasons`, `episodes`, `episode_files`

### Current MediaType Definitions (Scattered)

Multiple packages define their own `MediaType` string constants:
- `internal/decisioning/types.go`: `movie`, `episode`, `season`, `series`
- `internal/history/types.go`: `movie`, `episode`, `season`
- `internal/defaults/service.go`: `movie`, `tv` (exported `MediaType` type, used for settings keys only)
- `internal/metadata/artwork.go`: `movie`, `series` (for artwork paths)
- `internal/autosearch/types.go`: type alias to `decisioning.MediaType`

---

