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
8. **Push to remote** - `git push origin main && git push origin vX.Y.Z`
9. **Create GitHub release** - `gh release create vX.Y.Z --title "SlipStream vX.Y.Z" --notes-file <generated>`

## Version Scheme

- Format: `vMAJOR.MINOR.PATCH` (e.g., v0.2.12)
- Patch: Bug fixes, small features
- Minor: Significant new features, breaking changes within pre-1.0
- Major: Breaking changes (post-1.0)

## Release Notes Format

Group commits by feature area using the directory-to-category mapping below. Each category gets a `### Category` heading with bullet points underneath. Omit empty categories. End with a "Full Changelog" comparison link.

Example output:
```markdown
## What's Changed

### Auto Search & Upgrades
- Fix auto search grabbing same-quality releases due to stale file records
- Skip redundant season pack fallback for episodes that already have files

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

- The release pipeline is triggered automatically when a new tag is pushed
- Version is injected at build time via ldflags (no version file to update)
- Do not watch the pipeline - it will complete asynchronously
- The GitHub release is created by this skill with notes; the CI pipeline uploads build artifacts to the existing release
