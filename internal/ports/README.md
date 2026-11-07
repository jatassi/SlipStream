# Ported site indexers

Each package here is one of Prowlarr's C# site indexers rewritten in Go (spec S27, Ported site indexers). `port/` is the kit and the framework every port is written against, `all/` is the registry, `internal/` holds the families (Gazelle, Avistaz, SpeedApp) and `provenance/` is the manifest check. Port from Prowlarr only, never from Jackett, which is GPL-2.0-only.

Every command below reads a clone of Prowlarr. A blobless one is enough:

```sh
git clone --filter=blob:none --no-checkout https://github.com/Prowlarr/Prowlarr ../prowlarr
```

## Porting a site

1. `go run ./cmd/prowlarr-sync new --prowlarr ../prowlarr --site HDBits`. It writes `internal/ports/hdbits/` with one Go file per C# file, each with its header and package clause, the `upstream.json` that maps them at develop's newest commit, `internal/ports/all/hdbits.go`, and `fakes/site/ports/hdbits/fixture.yaml` with the required cases empty. Pass `--id` when the port id isn't the class lowercased, and `--family gazelle` for a site on a family base.
2. Translate each C# file into its Go file, top to bottom, with the vocabulary map in `port`'s package doc. `TorrentInfo` stays `TorrentInfo`, `ParseUtil.GetBytes` is `parseutil.GetBytes`, `AddCategoryMapping` is `port.Map`. The settings class becomes a settings struct with `port` tags. Mark each line you change on purpose with `// SlipStream:` and why, so a sync keeps it.
3. Fill the fixture. Use Prowlarr's own recording from `src/NzbDrone.Core.Test/Files/Indexers/` when there is one, through the scrub rewrite, and otherwise a synthetic exchange written from the parser's shape. Every port has `search`, `empty`, `auth` and `limit`. A port that signs in or takes a pasted cookie also has `loggedout`, and a two-factor port has `code`.
4. `go test ./fakes/site/ports -run 'TestPorts/hdbits' -update`, read the `want/*.json` it wrote, and commit them when they're right.
5. `make test` and `go run ./cmd/release lint`. A wrong port fails one of them. `Define` panics at init on a field, mode or rate-limit mistake, the runner tier fails on a parsing or outcome mistake, the scrub check on a leaked value, the provenance check on a header, and `ports-confined` on an HTTP client.

A port is done when it has its code, its `upstream.json`, its fixtures, its runner-tier tests, and copy reviewed against WRITING.md.

You never write a sign-in loop, a retry, Recent, Test, a spacing call, a credential check or a stop sentence. The framework derives them from what the port declares.

## Provenance

Each unit keeps an `upstream.json` beside its Go files. Every port and family must have one, and the kit (`port/`) and each `internal/prowlarr` package keep one when they mirror Prowlarr files. It names the Prowlarr commit the unit was last synced to, that commit's committer date (ef9c1824's is 2026-11-06, a day after its author date), the family a port builds on, and the Prowlarr files each Go file was ported from:

```json
{
  "commit": "ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9",
  "date": "2026-11-06",
  "family": "",
  "files": {
    "hdbits.go": ["src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBits.cs"],
    "parser.go": ["src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsParser.cs"]
  }
}
```

A port's `Entry.Upstream` embeds the same file through `port.ReadUpstream`, so the Indexer page and the sync tool read one record. Each Go file the manifest maps opens with these three lines, then a blank line, with its files joined by `, ` in the manifest's order:

```go
// Ported from Prowlarr's src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsParser.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.
```

`go test ./internal/ports/provenance`, which `make test` runs, fails when a port or family has no `upstream.json`, when a Go file's header names other files or another commit than its entry maps to it, when a file has a header its entry doesn't map, or when an entry maps a Go file that doesn't exist. Test files carry no header and the check skips them. An entry may name any Prowlarr file, inside `Indexers/` or not, such as `src/NzbDrone.Common/Extensions/StringExtensions.cs`. CI's `prowlarr-sync check` also fails when a named file isn't in Prowlarr at its commit.

`skipped.json` lists every Prowlarr indexer SlipStream doesn't port, with why. Each is obsolete upstream, dead, a generic engine, usenet, or a base no site uses. Its `commit` is the Prowlarr commit it and Prowlarr's hidden list (`_definitionBlocklist`) were last reviewed at.

## Syncing with Prowlarr

```sh
go run ./cmd/prowlarr-sync report --prowlarr ../prowlarr          # Markdown; --json for the routine
go run ./cmd/prowlarr-sync check  --prowlarr ../prowlarr
go run ./cmd/prowlarr-sync bump   --prowlarr ../prowlarr --unit port:hdbits --to <sha>
```

`report` lists, against develop (or `--ref`), each change set the weekly routine opens one draft PR for:

- each unit whose Prowlarr files changed since its commit, with the commits, the diff and the Go files each changed file maps to
- ports whose site upstream marked obsolete, moved to a Cardigann definition, or deleted
- live C# torrent sites that no unit maps and `skipped.json` doesn't list, with the media their categories carry
- changes to the hidden list

Each change set has a key, `<subject>@<short sha>`, which stays the same until upstream changes that subject again. One Prowlarr commit and one set of manifests always give the same report, byte for byte.

To port an upstream change by hand, port each hunk into the Go files the report maps it to and keep every `// SlipStream:` line. Then run `bump` to the change set's newest commit, which rewrites the unit's `upstream.json` and every header, and re-run the port's cases with `-update`. A hunk in a C# file the unit doesn't map yet needs a Go file and an entry in `upstream.json` before `bump`. The weekly routine does the same, from [docs/agents/prowlarr-sync-routine.md](../../docs/agents/prowlarr-sync-routine.md).
