# Release Command

Create a new release by committing changes, bumping version, and triggering the release pipeline.

## Steps

1. **Check git status** - Review staged/unstaged changes and untracked files
2. **Review recent commits** - Check commit history for context on commit message style
3. **Stage files** - Add relevant changed files (do not exclude settings.local.json and other local files)
4. **Create commit** - Write a descriptive commit message summarizing the changes with Co-Authored-By trailer
5. **Determine next version** - Check existing tags (`git tag --sort=-v:refname | head -1`) and increment appropriately (patch for fixes/features, minor for significant changes)
6. **Create tag** - `git tag vX.Y.Z`
7. **Push to remote** - `git push origin main && git push origin vX.Y.Z`

## Version Scheme

- Format: `vMAJOR.MINOR.PATCH` (e.g., v0.2.12)
- Patch: Bug fixes, small features
- Minor: Significant new features, breaking changes within pre-1.0
- Major: Breaking changes (post-1.0)

## Notes

- The release pipeline is triggered automatically when a new tag is pushed
- Version is injected at build time via ldflags (no version file to update)
- Do not watch the pipeline - it will complete asynchronously