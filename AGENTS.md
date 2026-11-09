## Agent skills

### Issue tracker

Issues live in GitHub Issues for `jatassi/SlipStream`, managed with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

The five default triage roles, each label named after its role. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Writing

Every string a person reads follows [WRITING.md](WRITING.md). Its Scope section lists those strings, and its Reviewing section sets how a review checks them.

## Settings

A feature that adds a Settings section places it in a group by the rules in [docs/settings-groups.md](docs/settings-groups.md), and updates that doc's table in the same PR.

## Third-party code

Adopt a library for standards, file formats and accessibility, and build the durable core and external clients ([ADR 0021](docs/adr/0021-slipstream-builds-its-durable-core-and-clients-and-adopts-libraries-for-standards.md)). Never port code from Jackett, which is GPL-2.0-only. Port Cardigann behaviour from Prowlarr instead.

## Checks

Run the fast checks locally and leave the full tiers to CI. On a 4-core cloud session the full tiers take more than 80 minutes (the Go tier about 21, e2e 18, component 8, browser 30 for Chromium alone), and a PR's CI runs all of them on every target.

Before each push, run:

1. `make lint`. Between commits, `go run ./cmd/release lint -part target` checks a Go-only change in seconds.
2. The Go packages you changed: `go run ./cmd/release test -tier go ./internal/<pkg>/... -run <Test>`. The release command sets the native toolchain that the cgo packages need.
3. For a web change, `cd web && bun run lint` and `bun run test <changed test files>`.
4. For a browser spec you added or changed, that file alone: `cd web && bun run e2e --project=chromium-desktop e2e/<name>.spec.ts`, with `TEST_SLIPSTREAM_BINARY` and `TEST_FAKEWORLD_BINARY` set to the binaries from `make build`.
5. For an e2e test you added or changed, `go run ./cmd/release test -tier e2e -filter <Test>`.

Don't run locally the whole Go, e2e, component or browser tier, the Diagram check (unless you changed the Diagram roots) or another OS or engine. CI runs them. To run one lane again, dispatch the `ci` workflow with `tier` and `filter`. The full CI run on a feature's final PR into main is its all-tiers run, after you merge main and run the checks above.

## Verification

Every PR that changes behaviour a user or a script can observe (UI, portal, API, host CLI, between Nodes) runs the verify-slipstream skill on the PR's head and pastes the run's `summary.md` into the PR's Verify section. Screenshots and videos go to the maintainer inline in chat, or on one private claude.ai Artifact when there are more than a few, and the PR carries only that link. Never attach media to a PR or commit anything under `.verify/`. The same PR updates or adds the feature files under `.claude/skills/verify-slipstream/features/` for every feature it touches and runs `bin/check-map.ts`. A PR that marks a parity row shipped cites it from a sub-feature. A PR that fixes a product bug verify found adds a regression test built from the run's journal. A PR with no observable change says "No observable change" and why. The rule is per PR, not per spec.
