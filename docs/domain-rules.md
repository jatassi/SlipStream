# Domain rules

This file holds the behavior rules that [GLOSSARY.md](../GLOSSARY.md)'s definitions point to and no spec or ADR records yet. GLOSSARY.md is the vocabulary.

## Cell

A final file makes it met. Otherwise it is downloading from the moment a Grab for it exists until no unfinished Grab holds it, fetching included, and failed when its latest Grab failed in a way only an Admin can resolve. A downloading or failed cell keeps the file its Grab would replace.

"Cell" is a word for engineers only. Copy never says it, in the UI, API docs, Health items or task outcomes. Per [WRITING.md](../WRITING.md) rule 20, a string says "file" or names the episode and Target, such as "S01E03 at 1080p", and the `house-words` lint keeps the word out.

## Item status

An Item reads unreleased only when nothing else applies, and a Work's status rolls up its Items the same way.
