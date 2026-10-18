# Vendored: anatomy

Source: https://github.com/wheresryan22/anatomy
Commit: e9488296a44211c42701ff71705b7901470ec78f
Fetched: 2026-10-17
License: MIT (`LICENSE` beside this file)

`.claude/skills/anatomy/` is the upstream skill at that commit of its default branch, unmodified,
without the repository's `.github/`, `.gitattributes`, `.gitignore` and the example pages its
build scripts generate (`examples/*/*.html`). It stays out of `skills-lock.json`.

`web/vendor/anatomy/` holds this file, `LICENSE` and the kit files the web uses, each
byte-identical to its copy in the skill. The `vendor-synced` lint fails when they differ, and the
release notices read the license and this commit from there. Never edit either copy by hand. The app
imports the kit through `@anatomy/*`, typed by the declarations in `web/src/diagram/kit/`.
`kit/audit.mjs` has no declaration there: only Bun scripts import it, so it never enters the bundle.

To re-sync, run `scripts/vendor-anatomy`. It clones the latest default branch into an empty
temporary directory, rewrites both copies and this file, and regenerates the notices. Review the diff
before committing it, since the source is third-party code.
