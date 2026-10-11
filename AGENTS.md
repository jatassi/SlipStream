## Agent skills

### Issue tracker

Issues live in GitHub Issues for `jatassi/SlipStream`, managed with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

The five default triage roles, each label named after its role. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Third-party code

Adopt a library for standards, file formats and accessibility, and build the durable core and external clients ([ADR 0021](docs/adr/0021-slipstream-builds-its-durable-core-and-clients-and-adopts-libraries-for-standards.md)). Never port code from Jackett, which is GPL-2.0-only. Port Cardigann behaviour from Prowlarr instead.
