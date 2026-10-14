## Agent skills

### Issue tracker

Issues live in GitHub Issues for `jatassi/SlipStream`, managed with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

The five default triage roles, each label named after its role. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Third-party code

Adopt a library for standards, file formats and accessibility, and build the durable core and external clients ([ADR 0021](docs/adr/0021-slipstream-builds-its-durable-core-and-clients-and-adopts-libraries-for-standards.md)). Never port code from Jackett, which is GPL-2.0-only. Port Cardigann behaviour from Prowlarr instead.

## Verification

Every PR that changes behaviour a user or a script can observe (UI, portal, API, host CLI, between Nodes) runs the verify-slipstream skill on the PR's head and pastes the run's `summary.md` into the PR's Verify section. Screenshots and videos go to the maintainer inline in chat, or on one private claude.ai Artifact when there are more than a few, and the PR carries only that link. Never attach media to a PR or commit anything under `.verify/`. The same PR updates or adds the feature files under `.claude/skills/verify-slipstream/features/` for every feature it touches and runs `bin/check-map.ts`. A PR that marks a parity row shipped cites it from a sub-feature. A PR that fixes a product bug verify found adds a regression test built from the run's journal. A PR with no observable change says "No observable change" and why. The rule is per PR, not per spec.
