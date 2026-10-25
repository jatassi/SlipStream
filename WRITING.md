# Writing

How SlipStream writes the words people read. [GLOSSARY.md](GLOSSARY.md) sets the vocabulary, [PRODUCT.md](PRODUCT.md) the principles and [DESIGN.md](DESIGN.md) where text sits on screen. This file sets how a sentence reads.

## Scope

Every string a person can read follows this file:

- UI copy: labels, buttons, dialogs, toasts, empty states, field errors and the error page.
- API problem titles and details, and OpenAPI operation summaries.
- Health items, task outcomes, log messages and CLI output.
- Go error strings, because they wrap into logs, Health items and problem details.
- Email.

Panic and invariant messages that only a developer sees follow the Go conventions below and nothing else.

## Sources

These rules come from established guides. When this file is silent, follow them in this order.

- Google, [Writing helpful error messages](https://developers.google.com/tech-writing/error-messages/set-tone).
- Microsoft, [Error message guidelines](https://learn.microsoft.com/en-us/windows/win32/debug/error-message-guidelines).
- GOV.UK Design System, [Error message](https://design-system.service.gov.uk/components/error-message).
- Nielsen Norman Group, [Error-message guidelines](https://www.nngroup.com/articles/error-message-guidelines/).
- Go Wiki, [Code Review Comments: error strings](https://go.dev/wiki/CodeReviewComments#error-strings).
- Wikipedia, [Signs of AI writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing), for the patterns that make text read as machine-written.

## Messages

1. **Say what happened, then what to do.** Give each its own sentence. If the person can do nothing, say what SlipStream does next. (Microsoft, GOV.UK, NN/g)
2. **Be specific.** Name the thing and the cause. Never write "Something went wrong", "An error occurred" or "Invalid input". (GOV.UK, Microsoft)
3. **Describe the problem, not the person.** "That PIN is wrong." is fine. "You entered a wrong PIN." blames. (Google)
4. **No apologies, courtesy words, jokes or cheer.** Leave out "sorry", "please", "oops", "whoops" and exclamation marks. (Google, GOV.UK, Microsoft)
5. **No harsh or vague verdicts.** Leave out "invalid", "illegal", "forbidden", "bad" and "fatal". Say what is wrong and what would be right: "Use 100 characters or fewer." (GOV.UK, Microsoft)

## Plain, literal words

6. **Don't give software human verbs.** A program, Node, disk or server does not answer, tell, think, know better or act on its own. Microsoft's guide forbids this outright. Glossary verbs such as a Node *holds* a Role or an Item *wants* a Target are defined terms, so use them as the glossary does.
7. **Use the standard word for a standard thing.** A server *responds*, a request gets a *response*, a retry happens *automatically*. A plain word that no other software uses is harder to read than the common one.
8. **No figures of speech.** Say the literal thing. "Its rules go with it" becomes "Deleting it also deletes its rules."
9. **Use contractions.** "can't", "isn't" and "doesn't" read as speech. Read the message aloud. If you wouldn't say it, rewrite it. (GOV.UK)

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

## Mechanics

- Sentence case everywhere. Glossary terms keep their capitals: Work, Node, Target, Profile, Grab.
- A full sentence ends with a period. Labels, buttons, headings and menu items don't.
- Straight quotes (`"` and `'`) in source. The ellipsis character `…` marks only an in-progress state or a menu item that opens a dialog, as DESIGN.md sets.
- Go error strings start lowercase, end without punctuation and add context as they wrap: `fmt.Errorf("read the volumes: %w", err)`. Their words still follow every rule above.
- An API problem's `title` and `detail` are sentences a person reads, so they follow the message rules.

## Examples from this codebase

| Before | After | Rules |
|---|---|---|
| The server did not answer, so check your connection and try again. | SlipStream isn't responding. Check your connection and try again. | 6, 7, 9, 10 |
| The folder isn't answering. | The folder isn't responding. Check the disk or share it is on. | 1, 6 |
| It tries again on its own; check the network of each Node that can't reach it. | SlipStream retries automatically. Check the network of each Node that can't reach it. | 6, 7, 11 |
| Accept them only on purpose; they rank below every other quality. | They rank below every other quality. Accept them only if you want them. | 11 |
| You're all set, Sam. | Your Account is ready, Sam. | 14 |
| Another cut already goes by this name or one of these phrases. | Another cut already uses this name or one of these phrases. | 8 |

## Reviewing

Every review of a diff that adds or changes a string in scope checks it against this file, including poteto-mode's interrogate reviewers, `/deslop` and verification swarms. Quote the string, name the rule number and give the rewrite. The `copy-words` lint fails on the fixed words and dashes from rules 2, 4, 11, 14 and 15 in Go and web source. Everything else needs a reviewer.
