# Issue tracker: GitHub

Issues and specs for `jatassi/SlipStream` live as GitHub issues. Use the `gh` CLI for all operations.

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`. Use a heredoc for multi-line bodies.
- **Read an issue**: `gh issue view <number> --comments`, filtering comments by `jq` and also fetching labels.
- **List issues**: `gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'` with appropriate `--label` and `--state` filters.
- **Comment on an issue**: `gh issue comment <number> --body "..."`
- **Apply / remove labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

Infer the repo from `git remote -v`; `gh` does this automatically when run inside a clone.

In a Claude Code cloud session these commands fail on GraphQL; use the REST forms under [Cloud threads](#cloud-threads-rest-through-gh-api) instead.

## Pull requests as a triage surface

**PRs as a request surface: no.** _(Set to `yes` if this repo treats external PRs as feature requests; `/triage` reads this flag.)_

When set to `yes`, PRs run through the same labels and states as issues, using the `gh pr` equivalents:

- **Read a PR**: `gh pr view <number> --comments` and `gh pr diff <number>` for the diff.
- **List external PRs for triage**: `gh pr list --state open --json number,title,body,labels,author,authorAssociation,comments` then keep only `authorAssociation` of `CONTRIBUTOR`, `FIRST_TIME_CONTRIBUTOR`, or `NONE` (drop `OWNER`/`MEMBER`/`COLLABORATOR`).
- **Comment / label / close**: `gh pr comment`, `gh pr edit --add-label`/`--remove-label`, `gh pr close`.

GitHub shares one number space across issues and PRs, so a bare `#42` may be either: resolve with `gh pr view 42` and fall back to `gh issue view 42`.

## Cloud threads: REST through `gh api`

In a Claude Code cloud session, GitHub traffic goes through Anthropic's GitHub proxy. It serves only a pinned set of GraphQL operations and rejects the rest with a 403 that says `This GraphQL query is not enabled for this session`. A `GH_TOKEN` you set gets the same 403, since the rule holds whatever token the environment carries. Most `gh issue` and `gh pr` commands, `gh repo view` and `gh api graphql` use GraphQL and fail there. REST through `gh api` works for the repositories attached to the session. `gh auth status` calls the token invalid because `GH_TOKEN` holds the placeholder `proxy-injected`, which the proxy swaps for the real credential on the way out, so ignore that report.

When a command fails with that 403, use these REST forms for the rest of the thread. Put each body in a file and pass it with `-F body=@<file>`, never inline in the command. The issues endpoints serve pull requests too, so comment, label and assignee calls take a PR number as well.

- **Create an issue**: `gh api repos/jatassi/SlipStream/issues --method POST -f title="..." -F body=@body.md -f 'labels[]=<label>' --jq .number`
- **Read an issue**: `gh api repos/jatassi/SlipStream/issues/<n> --jq '{title, state, body, labels: [.labels[].name]}'`, then `gh api repos/jatassi/SlipStream/issues/<n>/comments --paginate --jq '.[].body'`
- **List issues**: `gh api 'repos/jatassi/SlipStream/issues?state=open&labels=<label>&per_page=100' --paginate --jq '.[] | select(.pull_request | not) | {number, title, labels: [.labels[].name]}'`. The `select` drops pull requests.
- **Comment**: `gh api repos/jatassi/SlipStream/issues/<n>/comments --method POST -F body=@comment.md`
- **Add / remove a label**: `gh api repos/jatassi/SlipStream/issues/<n>/labels --method POST -f 'labels[]=<label>'` / `gh api repos/jatassi/SlipStream/issues/<n>/labels/<label> --method DELETE`
- **Create a missing label**: `gh api repos/jatassi/SlipStream/labels --method POST -f name=<label> -f color=<hex>`
- **Claim**: `gh api repos/jatassi/SlipStream/issues/<n>/assignees --method POST -f 'assignees[]=<login>'`, with your login from `gh api user --jq .login`
- **Close**: post the closing comment, then `gh api repos/jatassi/SlipStream/issues/<n> --method PATCH -f state=closed -f state_reason=completed`
- **Open a PR**: `gh api repos/jatassi/SlipStream/pulls --method POST -f title="..." -f head=<branch> -f base=<base> -F body=@body.md --jq .number`, with `-F draft=true` for a draft. Add its labels with the label call.
- **Edit a PR**: `gh api repos/jatassi/SlipStream/pulls/<n> --method PATCH -F body=@body.md`, or `-f title="..."`, or `-f state=closed` to close it.
- **Read a PR**: `gh api repos/jatassi/SlipStream/pulls/<n> --jq '{state, merged, draft, mergeable, mergeable_state, head: .head.sha, labels: [.labels[].name]}'`. For the diff, drop `--jq` and add `-H 'Accept: application/vnd.github.diff'`. Conversation comments come from the issue comments call, review comments from `repos/jatassi/SlipStream/pulls/<n>/comments`, reviews from `repos/jatassi/SlipStream/pulls/<n>/reviews`.
- **Checks at a commit**: `gh api repos/jatassi/SlipStream/commits/<sha>/check-runs --jq '.check_runs[] | {name, status, conclusion}'` and `gh api repos/jatassi/SlipStream/commits/<sha>/status --jq .state`

A cloud thread can't do what has no REST form: resolve a review thread, arm auto-merge, touch Projects v2, or attach media, since every command that takes `--attach` uses GraphQL. A PR opened from a cloud thread puts its evidence inline as text (see the **pr** skill).

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --comments`.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a single issue with **child** issues as tickets.

- **Map**: a single issue labelled `wayfinder:map`, holding the Notes / Decisions-so-far / Fog body. `gh issue create --label wayfinder:map`.
- **Child ticket**: an issue linked to the map as a GitHub sub-issue (`gh api` on the sub-issues endpoint). Where sub-issues aren't enabled, add the child to a task list in the map body and put `Part of #<map>` at the top of the child body. Labels: `wayfinder:<type>` (`research`/`prototype`/`grilling`/`task`). Once claimed, the ticket is assigned to the driving dev.
- **Blocking**: GitHub's **native issue dependencies**, the canonical, UI-visible representation. Add an edge with `gh api --method POST repos/<owner>/<repo>/issues/<child>/dependencies/blocked_by -F issue_id=<blocker-db-id>`, where `<blocker-db-id>` is the blocker's numeric **database id** (`gh api repos/<owner>/<repo>/issues/<n> --jq .id`, _not_ the `#number` or `node_id`). GitHub reports `issue_dependencies_summary.blocked_by` (open blockers only, the live gate). Where dependencies aren't available, fall back to a `Blocked by: #<n>, #<n>` line at the top of the child body. A ticket is unblocked when every blocker is closed.
- **Frontier query**: list the map's open children (`gh issue list --state open`, scoped to the map's sub-issues / task list), drop any with an open blocker (`issue_dependencies_summary.blocked_by > 0`, or an open issue in the `Blocked by` line) or an assignee; first in map order wins.
- **Claim**: `gh issue edit <n> --add-assignee @me`, the session's first write.
- **Resolve**: `gh issue comment <n> --body "<answer>"`, then `gh issue close <n>`, then append a context pointer (gist + link) to the map's Decisions-so-far.

## Kitchen labels

Besides the triage labels (see `triage-labels.md`) and the `wayfinder:*` labels above, the kitchen relies on these:

- **`garden`**: a workaround, banned pattern or repeated mistake to correct (see `garden.md`).
- **`door:one-way`**: a pull request that can't be walked back cheaply. It always waits for the chef (see `autonomy.md`).
- **`prototype`**: a throwaway prototype pull request, a draft titled `prototype:`, closed unmerged once the chef picks a variant. Also a prototype ticket, which waits on that pick and blocks the tickets that depend on its answer.
- **`live-ui-verified`**, **`unit-test-verified`**, **`type-check-only`**, **`verifier-blocked`**, **`verifier-failed`**: the verifier's ledger tier at the pull request's head SHA, one at a time (see `verification.md`).
