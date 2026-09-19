# Issue tracker: Local Markdown (in `docs/`)

Issues and specs for this repo live as markdown files under `docs/`, following the convention
already used by `docs/*-spec.md` and `docs/*-plan.md`. GitHub Issues are not used for this repo.

## Conventions

- The spec for a feature is `docs/<feature-slug>-spec.md`
- Implementation tickets are one file per ticket at `docs/tickets/<feature-slug>/<NN>-<slug>.md`, numbered from `01`, never a single combined tickets file
- Triage state is recorded as a `Status:` line near the top of each spec or ticket file (see `triage-labels.md` for the role strings)
- Comments and conversation history append to the bottom of the file under a `## Comments` heading
- Once a spec or ticket is fully implemented, add `Status: done` and leave the file in place as the record

## When a skill says "publish to the issue tracker"

Create the spec at `docs/<feature-slug>-spec.md`, or the ticket under `docs/tickets/<feature-slug>/`
(creating the directory if needed).

## When a skill says "fetch the relevant ticket"

Read the file at the referenced path. The user will normally pass the path directly.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a file with one **child** file per ticket.

- **Map**: `docs/tickets/<effort>/map.md` (the Notes / Decisions-so-far / Fog body).
- **Child ticket**: `docs/tickets/<effort>/<NN>-<slug>.md`, numbered from `01`, with the question in the body. A `Type:` line records the ticket type (`research`/`prototype`/`grilling`/`task`); a `Status:` line records `claimed`/`resolved`.
- **Blocking**: a `Blocked by: NN, NN` line near the top. A ticket is unblocked when every file it lists is `resolved`.
- **Frontier**: scan `docs/tickets/<effort>/` for files that are open, unblocked, and unclaimed; first by number wins.
- **Claim**: set `Status: claimed` and save before any work.
- **Resolve**: append the answer under an `## Answer` heading, set `Status: resolved`, then append a context pointer (gist + link) to the map's Decisions-so-far in `map.md`.

## Switching to GitHub Issues

If the repo starts using GitHub Issues, replace this file with the GitHub template from
`.agents/skills/setup-matt-pocock-skills/issue-tracker-github.md` and re-run `/setup-matt-pocock-skills`.
