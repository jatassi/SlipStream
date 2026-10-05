# Verification

How a verifier proves a change works in this kitchen. This is the one full statement of the verification ledger. Playbooks, skills and the Projects coordinator brief point here rather than restate it.

Verify skill: `none`

## The live lane is the floor

Every verdict includes the live lane: drive the running app the way a user does, on the surface the change touches (a UI, a CLI, a simulator), and capture the evidence (screenshots, a screencast, CLI transcripts). A verdict without live evidence is not a pass. CI passing is an input to the verdict, never the verdict.

When the verify skill above is `none`, prove the behaviour live by hand with whatever drives the surface (a browser, the CLI, a simulator), and propose creating one with `create-verification-skill`. Once a verify skill exists, `maintain-verification-skill` keeps it and its feature map current.

## Who verifies

The verifier is a fresh worktree subagent, and the merge gate counts its verdict.

- **Who spawns it.** The PR's worker may spawn it with `isolation: "worktree"` on the `verifier` role in `docs/agents/models.md`, its swarm lanes included.
- **Who it is.** A fresh agent, never the agent that wrote the change, and never resumed from an earlier round.
- **What it gets.** The pull request, its ticket, the commands that verify it (a brief's VERIFY field), and its own output directory for drafts such as the ledger comment (make-it-so's Subagents section). Nothing else from the worker: no plan, transcript, self-report or summary of the change.

The verifier records its own verdict on the pull request (The ledger), and the worker cites that verdict to reach merge-ready. The Projects coordinator may still start a separate verifier thread when it wants more independence, and its verdict counts the same way. The gate doesn't require one. A playbook may ask for more, as Autopilot-full's root swarm does.

## The ledger

The ledger lives in GitHub, never in memory or a transcript. It holds one verdict per pull request number and head SHA. A verdict is two things on the pull request:

- a comment that starts `ledger: <tier> at <head SHA>` and names the patch-id, the lanes run, the evidence and the verifier that produced it (its agent ID or thread);
- the tier as the pull request's only tier label. Replace the label when a new verdict lands.

A `ledger: merged on the chef's request` comment records a merge, not a verdict (Reading the ledger).

The verdict is bound to the head SHA and the patch-id, so a rebase can't carry a stale pass. "Was this verified?" is answered by the ledger and by nothing else.

## Tiers

Each verdict is at exactly one of these five tiers:

| Tier | Meaning |
| ---- | ------- |
| `live-ui-verified` | The live lane drove the running surface and passed. The only tier that passes the merge gate. |
| `unit-test-verified` | Tests passed, but no live lane ran. Not a pass. |
| `type-check-only` | Only static checks ran. Not a pass. |
| `verifier-blocked` | The verifier couldn't run its lanes (environment, auth, a dead tool). Not a pass. |
| `verifier-failed` | A lane proved a defect. |

## From a lane's report to a tier

Verifier lanes report `PASS`, `PASS+NOTES` or `FAIL` (a swarm worker's `ISSUES` is a `FAIL`, and its `BLOCKED` is blocked). The aggregate maps to one tier:

- Every lane `PASS` or `PASS+NOTES`, the live lane among them: `live-ui-verified`. Put the notes in the comment. A note that names a defect is a `FAIL`.
- Every lane passed, but no live lane ran: `unit-test-verified`, or `type-check-only` when only static checks ran.
- Any lane `FAIL`: `verifier-failed`.
- A lane that couldn't run, or whose evidence settles nothing (inconclusive, the wrong surface), and no `FAIL`: `verifier-blocked`.

## Reading the ledger

- **Check the head before you trust a verdict.** Compare it with `gh pr view <n> --json headRefOid`. A new head SHA voids the verdict unless its patch-id is unchanged.
- **`verifier-blocked` is not a pass.** Run a fresh verifier once the environment heals.
- **`verifier-failed` gets a fix**, not a re-verify of the same head.
- **A verifier overrides the worker.** A worker may self-report a tier. A verifier's verdict on the same pull request and head SHA replaces it.
- **The chef's explicit merge is the chef's own.** When the chef names a pull request and says to merge, land or ship it, in the thread or relayed verbatim by the Projects coordinator or an Autopilot-full root, the thread merges it whatever its tier. A coordinator or root that receives the request copies it verbatim into the brief or follow-up of the thread that merges, and the exemption carries with it. The merge gate and Shipping's verified-only rule don't apply to it. The thread records the merge in a ledger comment that starts `ledger: merged on the chef's request at <head SHA>` and names the pull request's actual tier, the tier of its current verdict or `no verdict`. That comment is not a verdict, so the tier label stays as it is. Every other merge goes through the merge gate in make-it-so's `playbooks/autopilot-full.md` step 5, and a closed gate stops the pull request at merge-ready for the chef.

## Cloud environments

The kitchen's vendored scripts (`watch-pr`, `check-plan`, `worktree-audit`) run on Bun. Anthropic-hosted cloud environments come with Bun preinstalled, so the setup script doesn't need to install it ([Installed tools](https://code.claude.com/docs/en/cloud-environments#installed-tools)). One caveat: the docs say Bun's package fetching doesn't work correctly through the session's security proxy, and a script's first run fetches its dependency (`commander`) with `bun install`, so that first run may fail in a cloud thread. `watch-pr` doesn't run in cloud threads anyway: GitHub's proxy blocks its GraphQL query, and Babysit reads PR state through REST there.
