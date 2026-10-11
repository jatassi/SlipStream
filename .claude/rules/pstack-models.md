---
description: pstack per-role model choices (overrides skill defaults)
---
# pstack model configuration. One line per role. Delete a line to fall back to the skill default.
# `inherit-parent` as a value: the role runs on the parent session's model and effort (omit Agent `model` and `effort`). Alias entries in a panel list still count toward its fan-out.
# budget: defaults (high for judgment, medium for code)
feature, refactoring: opus medium
bug-fix: opus medium
perf-issue: opus medium
hillclimb: opus medium
judgment and prose: opus high
hardest tasks: opus high
how explorer: opus medium
how explainer: opus high
why investigators: opus medium
why synthesizer: opus high
reflect tooling: opus medium
reflect judgment, divergent, synthesizer: opus high
arena runners: opus high, opus high
arena cross-judge pool: opus high, opus high
swarm workers: opus medium
architect runners: opus high, opus high
interrogate reviewers: opus high, opus high
