# Kitchen config

This folder is the kitchen's config. Before a skill, playbook or routine acts on one of these topics, it opens that topic's document and follows it; the document overrides any default the skill carries.

| Topic | Document | What it holds |
| ----- | -------- | ------------- |
| Issue tracker | [issue-tracker.md](issue-tracker.md) | GitHub Issues through `gh`, including wayfinder operations and the kitchen labels |
| Triage labels | [triage-labels.md](triage-labels.md) | The label string for each of the five triage roles |
| Domain | [domain.md](domain.md) | Where `GLOSSARY.md` and ADRs live, and how to use them |
| Models | [models.md](models.md) | Role → model tier and effort, detected families, and the fallback rule |
| Verification | [verification.md](verification.md) | Which verify skill proves changes, who verifies, and the live lane as the floor of every verdict |
| Autonomy | [autonomy.md](autonomy.md) | Areas as path globs, the rung of each, clean-merge and promotion rules, and the one-way doors |
| Garden | [garden.md](garden.md) | Banned patterns, the reflect / sweep / cluster cadences, and how `correct` fixes them |

Edit any document by hand. Re-running `setup-milliways` adds a missing document and its row, and leaves everything else as it is.
