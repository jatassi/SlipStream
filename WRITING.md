# Writing

How SlipStream writes the words people read. [GLOSSARY.md](GLOSSARY.md) names the concepts and gives each one's words on screen in its `_In UI_` line, [PRODUCT.md](PRODUCT.md) the principles and [DESIGN.md](DESIGN.md) where text sits on screen. This file sets how a sentence reads.

## Scope

Every string a person can read follows this file:

- UI copy: labels, buttons, dialogs, toasts, empty states, field errors and the error page.
- API problem titles and details, OpenAPI operation summaries and field descriptions.
- Health items, task outcomes, log messages and CLI output.
- Go error strings, because they wrap into logs, Health items and problem details.
- Email.

Panic and invariant messages that only a developer sees follow the Go conventions below and nothing else.

## Sources

These rules come from established guides. For a case this file doesn't cover, follow them in this order.

- Google, [Writing helpful error messages](https://developers.google.com/tech-writing/error-messages/set-tone).
- Microsoft, [Error message guidelines](https://learn.microsoft.com/en-us/windows/win32/debug/error-message-guidelines).
- Microsoft Writing Style Guide, [Use contractions](https://learn.microsoft.com/en-us/style-guide/word-choice/use-contractions).
- GOV.UK Design System, [Error message](https://design-system.service.gov.uk/components/error-message).
- Nielsen Norman Group, [Error-message guidelines](https://www.nngroup.com/articles/error-message-guidelines/).
- Go Wiki, [Code Review Comments: error strings](https://go.dev/wiki/CodeReviewComments#error-strings).
- Wikipedia, [Signs of AI writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing), for the patterns that make text read as machine-written.

## Messages

1. **Say what happened, then what to do.** Give each its own sentence. If the person can do nothing, say what SlipStream does next. (Microsoft, GOV.UK, NN/g)
2. **Be specific.** Name the thing and the cause. Never write "Something went wrong", "An error occurred" or "Invalid input". (GOV.UK, Microsoft)
3. **Describe the problem, not the person.** "That PIN is wrong." is fine. "You entered a wrong PIN." blames. (Google)
4. **No apologies, courtesy words, jokes or cheer.** Leave out "sorry", "please", "oops", "whoops" and exclamation marks. (Google, GOV.UK, Microsoft)
5. **No harsh or vague verdicts.** Leave out "invalid", "illegal", "forbidden", "bad" and "fatal". Say what is wrong and what would be right: "Use 100 characters or fewer." A UI term from GLOSSARY.md, such as Mark bad, is exempt. (GOV.UK, Microsoft)

## Plain, literal words

6. **Don't give software human verbs.** A program, Node, disk or server does not answer, tell, think, know better or act on its own. Microsoft's guide forbids this outright. This holds for glossary verbs too (rule 21).
7. **Use the standard word for a standard thing.** A server *responds*, a request gets a *response*, a retry happens *automatically*. A plain word that no other software uses is harder to read than the common one.
8. **No figures of speech.** Say the literal thing. "Its rules go with it" becomes "Deleting it also deletes its rules."
9. **Use contractions.** "can't", "isn't" and "doesn't" read as speech. (Microsoft) Read the message aloud. If you wouldn't say it, rewrite it. (GOV.UK)

## Patterns that read as machine-written

Large language models write in recognisable patterns ([Wikipedia](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing)). Each one below has turned up in this repository.

10. **Cause and instruction spliced with ", so".** Split them: "The email failed. Copy the link and send it yourself." A short consequence inside one sentence is fine: "Node A is the Controller, so it can't be removed."
11. **Semicolons, colons and dashes joining clauses.** Use a period. A colon before a list or a quoted value is fine. Em dashes and en dashes never appear in prose.
12. **Personified status lines.** "The folder isn't answering." "SlipStream can't tell how this is now, so it reads as it last did."
13. **Negative parallelism for effect.** "Not X, but Y", "Y rather than X", "no X, no Y, just Z". State Y. Keep "not X" only when it names the mistake the reader is likely to make: "Passkeys need SlipStream's name, not an IP address."
14. **Chatbot register.** "You're all set", "and you're in", "All good", "Great", "Done!", "Let's".
15. **AI vocabulary.** additionally, align with, boasts, crucial, delve, enhance, ensure (in prose), foster, garner, highlight (as a verb for meaning), intricate, key (as an adjective), landscape, leverage, meticulous, multifaceted, pivotal, quietly, robust, seamless, showcase, steady, testament, underscore, utilize, vibrant.
16. **Fancy "is".** "serves as", "stands as", "acts as", "features", "boasts" become "is" or "has".
17. **Trailing "-ing" clauses that claim significance.** "..., ensuring your library stays in sync." Cut the clause, or say what happens in its own sentence.
18. **Over-compression.** Messages are whole sentences with their articles and verbs. Labels, buttons and headings may be fragments. "Down. Nothing is getting through." becomes "The link is down."
19. **Uniform clipped rhythm.** A paragraph of four-word sentences reads as generated just as a wall of clauses does. Join two clauses with "and" or "because" when they are one idea.

## House words in copy

GLOSSARY.md is written for engineers. The person at a screen hasn't read it.

20. **A glossary term reaches the screen only through its `_In UI_` line.** A term whose line is none stays out of copy, and the string says what the concept means for the reader instead. "3 cells become wanted." becomes "SlipStream will look for 3 more files."
21. **Glossary verbs aren't sentence verbs.** Don't make holds, wants, fills, owes, reaches, dials, covers or writes off the verb of a sentence. Say the action. "Den doesn't hold Downloading." becomes "Den doesn't have the Downloading Role." "2 torrents owe seeding." becomes "2 torrents need more seeding."
22. **Use the Content Type's noun.** Where the type is known, say movie, series, season or episode, not Work or Item. "The Work has no Item S01E02." becomes "This series has no episode S01E02."
23. **Write for the screen, not from the definition.** Don't paste a glossary entry into copy. The string says what the person sees and what they can do. "Their cells read missing until they return." becomes "They show as missing until the files are back."
24. **No code words.** API fields, enum values, state names and task internals stay out of copy: `writeOff`, "reads checking", "unit", "shown", "in step", "guardrail". Name the thing the person sees.
25. **Standard term, short definition on hover and tap.** A technical state uses the term the rest of the industry uses, such as Direct play, Direct stream, Transcode, Hit and run or Remote path mapping, with a one-sentence definition on hover and tap. Don't swap an industry term for a house one. When rule 7 and a glossary _Avoid_ list disagree, the `_In UI_` line settles it, and it takes the standard word.
26. **Point only at what exists.** An instruction names a button, page or setting that this build has. If the person can do nothing, say what SlipStream does next (rule 1). "The Controller can't be removed. Hand it over instead." is wrong while Handover isn't built.
27. **A Go error never reaches a person on its own.** A problem detail, toast or Health item is a sentence written for it. `err.Error()` goes to the log. "TMDB: didn't respond: dial tcp: ..." becomes "TMDB isn't responding. SlipStream retries automatically."

## Patterns from longer prose

Help text, settings descriptions and OpenAPI docs run long enough to pick up these patterns too. Rules 8, 13 and 18 already cover three of them, and the table numbers the other six as rules 28 to 33.

| Pattern | Rule | Fix | Before | After |
|---|---|---|---|---|
| Negation-first reveal | 13 | State the positive claim. Drop the contrast setup. | It is not a wrong answer. It is a non-answer. | The response restates the question and gives no answer. |
| Significance designation | 28 | Say what the thing does, not that it matters. | It is the leg that answers the actual question. | Leg 4 measures accuracy on held-out questions. Use its result to decide. |
| Abstraction agency | 29 | Make a person, system or number the subject of the verb. | Median hides it. The table shows it. | The median latency is 1.2 s, but 3 of 40 runs took over 10 s. Rows 3 to 5 list those runs. |
| Deferred noun | 30 | Name the items in the first sentence. | Five of those six rows are one cluster. The sixth is not. | Rows 1 to 5 used the default sampler. Row 6 used top-k sampling. |
| Coy or thesis-shaped header | 31 | Write a header that labels the topic. Put the claim in the body. | What "exhausted" means | Definition of "exhausted" |
| Aphoristic closer | 32 | State the specific defect, or delete the sentence. | It is the kind of number that looks like evidence and is not. | The 12% gain comes from one run with no variance estimate. |
| Straw-man knockdown | 33 | State the correct reading and its evidence. Mention the wrong reading only if the reader holds it. | "It thinks twice as long" is the obvious reading, and it is wrong. | Output tokens doubled, but reasoning tokens stayed flat. |
| Fragment cadence | 18 | Use full sentences with a subject and a verb. | Six legs. One GPU, one server build. | The test has six legs. All six use the same GPU and server build. |
| Dressed metaphor | 8 | Name the literal action and its effect. | That is information loss wearing the costume of a style fix. | The edit removed the confidence intervals. The commit called it a style fix, but it changed the content. |

## Mechanics

- Sentence case everywhere. A term whose `_In UI_` line capitalizes it keeps its capitals: Node, Target, Profile, Grab. Other house words don't appear (rule 20).
- A full sentence ends with a period. Labels, buttons, headings and menu items don't.
- A time inside a sentence names its day with its preposition, as a person says it: "at 2:02 PM" today, "yesterday at 2:02 PM", "tonight at 4:00 AM", "on Mon at 9:12 AM" within the week and "on Nov 19 at 9:12 AM" further off. Write it through `whenWords`, which also follows the person's hour cycle, and never build a date or time by hand.
- Straight quotes (`"` and `'`) in source. The ellipsis character `…` marks only an in-progress state or a menu item that opens a dialog, as DESIGN.md sets.
- Go error strings start lowercase, end without punctuation and add context as they wrap: `fmt.Errorf("read the volumes: %w", err)`. Their words still follow every rule above.
- A Go error states the cause and nothing else. The layer that shows it to a person, such as a problem's title or detail, a Health item or a HostRefusal, adds what to do as its own sentence. An error that ends a CLI command and is printed after a prefix may carry its instruction as a second sentence after a period.
- An API problem's `title` and `detail` are sentences a person reads, so they follow the message rules.
- A problem title of one sentence has no final period. A title of two sentences ends each with a period.
- An OpenAPI field description may use the reference shorthand of a fact, a semicolon and its edge case: "Seconds; null resets it to its default". Everywhere else, rule 11 holds.

## Examples from this codebase

| Before | After | Rules |
|---|---|---|
| The server did not answer, so check your connection and try again. | SlipStream isn't responding. Check your connection and try again. | 6, 7, 9, 10 |
| The folder isn't answering. | The folder isn't responding. Check the disk or share it is on. | 1, 6 |
| It tries again on its own; check the network of each Node that can't reach it. | SlipStream retries automatically. Check the network of each Node that can't reach it. | 6, 7, 11 |
| Accept them only on purpose; they rank below every other quality. | They rank below every other quality. Accept them only if you want cinema recordings. | 11 |
| You're all set, Sam. | Your Account is ready, Sam. | 14 |
| Another cut already goes by this name or one of these phrases. | Another cut already uses this name or one of these phrases. | 8 |
| 3 cells become wanted. | SlipStream will look for 3 more files. | 20 |
| That Node doesn't hold Downloading. | That Node doesn't have the Downloading Role. | 21 |
| 2 torrents owe seeding. | 2 torrents need more seeding. | 21 |
| Announce error | Tracker error | 25 |
| The Controller can't be removed. Hand it over instead. | The Controller can't be removed. | 26 |
| Couldn't refresh the metadata of Heat. TMDB: didn't respond: dial tcp: lookup api.themoviedb.org: no such host | Couldn't refresh the metadata of Heat. TMDB isn't responding. SlipStream retries automatically. | 27 |

## Reviewing

Every review of a diff that adds or changes a string in scope checks it against this file, including poteto-mode's interrogate reviewers, `/deslop` and verification swarms. Quote the string, name the rule number and give the rewrite. The `copy-words` lint fails on the fixed phrases and words from rules 2, 4, 14 and 15 and on any em or en dash, except a string that is only a dash, such as an empty-cell marker. It checks Go string literals and `doc:` tags in product code, and web string literals, template text and JSX text outside tests, `e2e/` and `harness/`. The Go and web checks read one list, `web/lint/copy-words.json`. The `house-words` lint does the same for the glossary words of rules 20 to 24, listed in `web/lint/house-words.json`. It allows only the matches `web/lint/house-words-baseline.json` records, and that count only goes down: after a fix, run `bun lint/house-words-baseline.js` from `web/` to lower it. The `problem-detail-from-error` lint enforces rule 27, and `glossary-format` requires an `_In UI_` line on every glossary entry. Everything else needs a reviewer.
