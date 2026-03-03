# Release Command

Create a new release by committing changes, bumping version, generating release notes, and triggering the release pipeline.

## Steps

1. **Check git status** - Review staged/unstaged changes and untracked files
2. **Review recent commits** - Check commit history for context on commit message style
3. **Stage files** - Add relevant changed files (do not exclude settings.local.json and other local files)
4. **Create commit** - Write a descriptive commit message summarizing the changes with Co-Authored-By trailer
5. **Determine next version** - Check existing tags (`git tag --sort=-v:refname | head -1`) and increment appropriately (patch for fixes/features, minor for significant changes)
6. **Create tag** - `git tag vX.Y.Z`
7. **Generate release notes** - Collect commits since the previous tag, categorize them by feature area, and write concise bullet points grouped by category
8. **Push commit to remote** - `git push origin main` (triggers cache warming workflow)
9. **Await cache warming** - Poll `gh run list -w cache.yml -L 1 --json status,conclusion` until the most recent run completes (typically 10-30s with warm caches). Use `gh run watch <id> --exit-status` to wait.
10. **Push tag and create release** - `git push origin vX.Y.Z` then `gh release create vX.Y.Z --title "SlipStream vX.Y.Z" --notes-file <generated>` (tag push triggers the release pipeline with warm caches)

## Version Scheme

- Format: `vMAJOR.MINOR.PATCH` (e.g., v0.2.12)
- Patch: Bug fixes, small features
- Minor: Significant new features, breaking changes within pre-1.0
- Major: Breaking changes (post-1.0)

## Release Notes Format

Group commits by feature area using the directory-to-category mapping below. Each category gets a `### Category` heading with bullet points underneath. Omit empty categories. End with a "Full Changelog" comparison link. Focus on the functional change or result of a bugfix. Do not include technical details of what caused a bug or how we implemented a feature. Highlight new features or functionality added to existing features.

Example output:
```markdown
## What's Changed

### Auto Search & Upgrades
- Fix auto search grabbing same-quality releases
- Skip redundant season pack fallback for episodes that already have files
- **NEW:** auto search now supports trackers using the UNIT3D protocol

### UI
- Add loading skeleton to search results page

**Full Changelog**: https://github.com/owner/repo/compare/vPREVIOUS...vCURRENT
```

### Directory-to-Category Mapping

| Directory Pattern | Category |
|---|---|
| `internal/autosearch/`, `internal/decisioning/` | Auto Search & Upgrades |
| `internal/rsssync/` | RSS Sync |
| `internal/movies/` | Movies |
| `internal/tv/` | TV Shows |
| `internal/library/quality/` | Quality Profiles |
| `internal/indexer/`, `internal/prowlarr/` | Indexers |
| `internal/downloader/` | Download Clients |
| `internal/import/` | Media Import |
| `internal/library/scanner/` | Library Scanner |
| `internal/history/` | History |
| `internal/portal/` | Request Portal |
| `internal/notification/` | Notifications |
| `internal/database/` | Database |
| `internal/config/` | Configuration |
| `internal/server/`, `internal/api/` | API |
| `web/` | UI |
| `.github/`, `scripts/`, `Makefile` | Build & CI |
| `docs/` | Documentation |

If a commit touches multiple areas, place it under the most significant one. If a commit doesn't clearly fit any category, use a "General" category.

## Notes

- The commit push to `main` triggers the cache warming workflow (`cache.yml`), which must complete before pushing the tag
- The tag push triggers the release pipeline (`release.yml`), which builds and uploads artifacts to the GitHub release
- Version is injected at build time via ldflags (no version file to update)
- Do not watch the release pipeline - it will complete asynchronously
- The GitHub release is created by this skill with notes; the CI pipeline uploads build artifacts to the existing release
- Don't ask for permission to kick off the pipeline or validation on correctness of release note
