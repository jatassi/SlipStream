# Autonomy

Who merges what in this kitchen. Trust is earned per area: each area sits on a rung of the trust ladder, and the trust-ladder scorer reads this document to decide when an area's rung changes.

## Areas

| Area | Paths | Rung |
| ---- | ----- | ---- |
| everything | `**` | chef |

- **Area** is a unique kebab-case name.
- **Paths** are one or more path globs, each in backticks, separated by commas, matched against repo-relative paths. A file belongs to the first area, top to bottom, with a glob that matches it, so put narrow areas above broad ones and keep a catch-all `**` area last.
- **Rung** is `chef` or `gated`:
  - `chef`: the chef merges every pull request in the area.
  - `gated`: a thread merges its own pull request once the gate holds: CI is green, a fresh verifier's verdict passes at the head SHA with live evidence, and the door is two-way.
- A pull request that touches files in more than one area takes the lowest rung among them.

## Rules

| Rule | Value |
| ---- | ----- |
| clean-window-days | 7 |
| promotion-streak | 10 |

- **Clean merge**: a merge that no revert, fix-forward pull request or `garden` issue links back to within `clean-window-days` of the merge.
- **Promotion**: when a `chef` area's last `promotion-streak` settled merges (those whose clean window has closed) are all clean, a promotion is due. The daily risk-digest routine proposes it as a pull request that changes the area's rung to `gated`, and the chef merges it. The ladder never climbs itself.
- **Demotion**: any unclean merge in a `gated` area sets its rung back to `chef` at once (the daily risk-digest routine commits the change), and its streak restarts at zero.

## One-way doors

A one-way door can't be walked back cheaply. Its pull request carries the `door:one-way` label and waits for the chef in every area, at every rung.

### Paths

A pull request that touches any of these paths is a one-way door.

- `**/migrations/**`
- `.github/workflows/**`

### Changes

A pull request that makes any of these changes is a one-way door, wherever it lands.

- Deletes or rewrites stored user data, or changes a stored data format without a migration back.
- Changes a public API, CLI contract or file format that other code or people depend on.
- Changes authentication, authorization, secrets handling or billing.
- Publishes something outside the repo: a release, a package, a message to users.
- Upgrades or swaps a framework, runtime or database.
