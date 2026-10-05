# Models

Which model each role runs on. Every skill, playbook and agent that dispatches a subagent reads its role's row here before it dispatches; a row here overrides any model the skill names itself.

Detected harness: claude-code
Detected families: anthropic

## Roles

| Role | Model | Effort | Covers |
| ---- | ----- | ------ | ------ |
| judgment | opus | high | Judgment, prose, plans and synthesis. pstack: judgment and prose; how explainer; why synthesizer; reflect judgment, divergent, synthesizer |
| hardest-code | opus | high | Cross-cutting design, concurrency, subtle algorithms. pstack: hardest tasks |
| code | sonnet | medium | Code delegates. pstack: feature, refactoring; bug-fix; perf-issue; hillclimb; swarm workers; reflect tooling |
| explorer | sonnet | medium | Read-only search and investigation. pstack: how explorer; why investigators |
| verifier | opus | high | A fresh verifier and its swarm lanes: gates, live lane, regression against trunk, diff audit |
| review-panel | opus, opus, opus | high | pstack: arena runners; arena cross-judge pool; architect runners |
| interrogate | opus, opus, opus | high | Adversarial review of one-way doors. pstack: interrogate reviewers |

Who verifies is stated in `docs/agents/verification.md`.

fable has no default role. To promote it for this kitchen, put `fable` in a role's Model cell.

## Model values

- **A tier**: `haiku`, `sonnet`, `opus` or `fable`, run as a subagent of this harness. Name tiers only, never versions, so the kitchen always runs the latest model of each tier.
- **Another family, through its CLI**: `<cli>:<tier>`, for example `codex:sol` (OpenAI tiers: `luna`, `sol`, `astra`), `gemini:pro`, or `claude:opus` from a harness of another family. The subagent drives that CLI non-interactively with the role's brief and returns its output.
- **`inherit`**: the role runs on its parent's model.
- **A list**, comma-separated, for a panel: one subagent per entry, so the list's length is the panel's size.

## Fallback

When a role's model or CLI isn't available in the thread (the harness has no such tier, the CLI isn't installed, or it isn't signed in), run that role on the parent's model and keep going. Say so in the thread's report, one line per role:

`fallback: <role> wanted <model>, ran on the parent's model (<reason>)`

A config written on a laptop with a second family installed still runs in a cloud thread that has only this harness; the fallback lines show what ran where.
