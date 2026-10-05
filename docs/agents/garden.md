# Garden

Weeds this kitchen pulls before agents copy them. A **garden issue** is a GitHub issue labelled `garden` that records one workaround, banned pattern or repeated mistake, with the file and line, and a link to the pull request or merge it came from.

## Banned patterns

Each pattern is a `### <name>` heading followed by `Detect:` and `Why:` lines. `Detect:` is either an extended regular expression in backticks, matched against lines added since the last sweep, or a plain-language check that needs judgment. Add a pattern by adding a heading.

### workaround-comment

Detect: `\b(HACK|XXX|WORKAROUND|KLUDGE)\b`
Why: a comment that admits a workaround marks a fix that hasn't been made yet.

### lint-suppression

Detect: `(eslint-disable|biome-ignore|oxlint-ignore|@ts-ignore|@ts-expect-error|@ts-nocheck|# noqa|# type: ignore|# pylint: disable|swiftlint:disable|//nolint)`
Why: a suppression hides the error the lint was written to name.

### god-file-growth

Detect: a source file over 500 lines that grew since the last sweep.
Why: a file that keeps growing is a module that stopped being one; split it before the next thread adds to it.

### mock-mode-gap

Detect: a new call to an external service, network API or paid resource with no mock-mode path beside it.
Why: a path that only works against the real service can't be verified live in a cloud thread or exercised in tests.

## Cadences

| Loop | When | What it does |
| ---- | ---- | ------------ |
| reflect | the last step of every thread | Files a `garden` issue for anything the thread had to work around. |
| sweep | nightly, by the garden-sweep routine | Checks the day's merges against the banned patterns and files a `garden` issue per finding not already open. |
| cluster | weekly on Monday, by the garden-cluster routine | Groups open `garden` issues by root cause and runs `correct` on each cluster. |

## Correcting

`correct` makes a mistake impossible at the highest level that works, in this order: architecture, then types, then a lint whose error names the fix, then a test, then docs. A cluster of five weeds should end as one lint rule or one architectural fix, not five patches. The pull request that lands the correction closes the cluster's issues.
