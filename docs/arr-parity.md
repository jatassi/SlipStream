# Sonarr and Radarr parity

S23 (Release parsing and Custom Formats) ports Sonarr's and Radarr's release parser and Custom Formats, so a release reads the way it reads there and a format from TRaSH Guides runs as written. This page lists what SlipStream ports, where it reads a name differently on purpose and why, and what it leaves out. The v1 features these replace are rows P-071 to P-073, P-128 to P-130, P-262, P-357 and P-358 of [the parity matrix](parity-matrix.md).

## Pins and the resync

`cmd/arr-sync/pins.json` pins each upstream. `make arr-sync` regenerates the pattern tables (the `zz_` files), the fixture JSON and the TRaSH snapshot from them, and the golden tests then show each change a new pin brings.

| Upstream | Commit | What SlipStream takes |
| --- | --- | --- |
| Sonarr | `c018578e4b79` | Parser patterns, file extensions, languages, Custom Format kinds, parser test cases |
| Radarr | `928e8871ccad` | Parser patterns, quality modifiers, languages, Custom Format kinds, parser test cases |
| TRaSH Guides | `98853590599f` | 243 Radarr and 236 Sonarr Custom Formats |
| guessit, PTT, jhin | pinned beside them | Fuzz seeds only |

Each package's `upstream.json` maps its Go files to the C# files they port.

## Fixtures

Every `[TestCase]` row of the ported test files runs through `release.Title` or `release.File` with the real series and movie readers. A row SlipStream reads differently on purpose sits in `<File>.overrides.json` with its reason. A row not yet passing sits in `<File>.pending.json`, and a pending row that starts passing fails the test, so the list only shrinks.

| Owner | Upstream | Files | Cases | Overrides | Pending |
| --- | --- | --- | --- | --- | --- |
| `internal/release` | Sonarr | 2 | 471 | 52 | 0 |
| `internal/release` | Radarr | 3 | 471 | 16 | 0 |
| `internal/contenttypes/series` | Sonarr | 13 | 1,046 | 1 | 2 |
| `internal/contenttypes/movie` | Radarr | 5 | 423 | 17 | 1 |
| Total | | 23 | 2,411 | 86 | 3 |

The three pending rows are Radarr's folder name "(1995) Movie Name" and two of Sonarr's hashed releases under a Windows path.

`internal/formats` compiles all 479 TRaSH formats at the pin and runs each against the 2,241 titles in `testdata/titles.txt` under the match limit. A format that doesn't compile or a pattern that runs past the limit fails CI.

## Ported

| Behaviour | Upstream | SlipStream |
| --- | --- | --- |
| Name preparation: reversed titles, hashed names, extensions, site prefixes and suffixes | Sonarr's `Parser.ParseTitle` and `FileExtensions`, Radarr's `Parser` | `release/prepare.go` |
| Quality: source, resolution and modifier | Radarr's `QualityParser.ParseQualityName`, with Sonarr's source and resolution patterns where Radarr's read nothing | `release/read_quality.go` |
| Revision from PROPER, REPACK, RERIP, vN and REAL | `QualityParser.ParseQualityModifiers`, with Sonarr's version pattern | `release/read_revision.go` |
| Release group, with the exception list | Radarr's `ReleaseGroupParser` | `release/read_group.go` |
| Languages | Sonarr's and Radarr's `LanguageParser` | `release/read_language.go` |
| Edition text | Radarr's `Parser` | `release/read_edition.go` |
| Plex and Jellyfin id tags | Sonarr's and Radarr's `Parser` | `release/read_ids.go` |
| Series titles and numbering: SxxEyy, multi-episode, season and multi-season packs, daily and absolute | Sonarr's `Parser` | `contenttypes/series/names.go`, `marks.go` |
| Mapping a name's numbers to stored episodes | Sonarr's `ParsingService.GetEpisodes` | `contenttypes/series/coverage.go` |
| Movie title and year | Radarr's `Parser.ParseMovieTitle` | `contenttypes/movie/names.go` |
| Custom Format kinds, fields and enum values | Each app's `CustomFormats/Specifications` | `formats/kinds.go`, `formats/zz_dialects.go` |
| The combining rule: Negate, then Required and one match per kind | `CustomFormatCalculationService`, `SpecificationMatchesGroup` | `formats/match.go` |
| A release title condition tests the name the readers read | Radarr's `SimpleReleaseTitle`, Sonarr's `ReleaseTitle` | `formats/match.go` |
| A name with no language tag counts as the Work's original language | `AggregateLanguages` | `formats/kinds.go` |
| Pasting a format as TRaSH Guides, Radarr or Sonarr export it, or as Sonarr's API returns it | `CustomFormat`, `CustomFormatInput` | `formats/paste.go` |
| .NET regex semantics, ignoring case | .NET `Regex` | `internal/regex` |
| Propers and repacks: "Prefer and upgrade", "Prefer, don't upgrade" and "Don't prefer" | Sonarr's and Radarr's "Propers and Repacks" setting | `targets` |
| A repack replaces only a file from its own release group | Sonarr's `RepackSpecification` | `targets` |
| Ranking compares Revision across the Qualities of one tier | Sonarr's `QualityModelComparer` | `grabs/chain.go` |

## Deviations

Each row is a place SlipStream reads or decides differently from upstream on purpose. The overrides files hold the row-by-row reasons.

| Deviation | Why | Rows |
| --- | --- | --- |
| A name with a source and no resolution reads as that source's lowest Quality, with the resolution unknown. Upstream fills in a default resolution. | S17 (Targets, Profiles and Editions) keeps an unstated fact unknown, so a Target's filters and Custom Formats never test a guess. | 30 |
| A name with a resolution and no source keeps the source unknown. Upstream's quality carries the TV source. | The same rule. | 18 |
| HD TV in a name with a resolution keeps the source unknown. | Upstream reads HD TV only in a name without a resolution. | 4 |
| x264 alone reads as no Quality. Upstream reads it as SDTV. | S17's quality table needs a source or a resolution. | 1 |
| Remux below 1080p keeps the Bluray source and the Remux modifier. Upstream drops the modifier. | A format that tests Remux matches it. | 2 |
| `.mkv` reads by Sonarr's extension table. Radarr reads it as WEB-DL. | One core parser serves both types and uses one extension table. | 2 |
| An HD DVD with a VC-1 stream reads as BR-DISK, as Radarr reads it. Sonarr has no BR-DISK. | Formats that refuse BR-DISK match it. | 1 |
| Release groups are read by Radarr's parser and exception list for both types. | It disagrees with Sonarr's fixtures on 10 rows and Sonarr's with Radarr's on 46, and its exception list holds the groups TRaSH's Radarr formats match. | 10 |
| Languages are read by Sonarr's tag table for both types. | One table for both types. Radarr's extra tags, such as ES, KOR and VIE, read as no language. | 15 |
| Radarr's language test cases read the whole movie name, with its group replaced. | Radarr's test reads the name with `LanguageParser` alone, where SlipStream reads it as `ParseMovieTitle` does. | 2 |
| Sonarr's languages are read only from the tokens after the episode number. | As Sonarr's `ParseTitle` does. The fixture reads the whole title. | 1 |
| A pasted format's pattern that runs past `timings.formats.match`, 100 ms by default, counts as not matching, and the format shows a warning. SlipStream's fixed parser patterns run with no limit, as upstream's do. | One slow pattern can't stall a search. | - |
| A pasted format whose kinds don't name its app is refused with `format_app_needed`. | Radarr and Sonarr number sources, resolutions and languages differently. TRaSH's release group formats name neither app. | - |
| Formats made in SlipStream hold values by name, in SlipStream's own dialect. Pasted ones keep their app's numbers. | A format reads the same in the editor whichever app it came from. | - |
| A release year may differ from the Work's by "Release year can differ by", one year by default, per Content Type. | Regional release years. | - |
| Ranking breaks ties after the score by Indexer priority, freeleech, seeders and size, and each Release keeps the step it lost at. | v1's tiebreakers (P-071), with the breakdown an Admin can read. | - |
| Upgrades by Revision compare within one Quality. Ranking compares Revision across a tier. | The spec's upgrade rule, with Sonarr's ranking. | - |

## Left out

| Behaviour | Why |
| --- | --- |
| Scene numbering from TheXEM or Sonarr's mappings | TVDB is the numbering authority, and a scene source is a new external boundary. |
| Specials found by episode title | Only `S00Exx` names a special. |
| Usenet protocol ranking | SlipStream has no Usenet. |
| Custom Formats on probe facts | Formats read only what a release says about itself. |
| Exporting formats | No parity row asks for it. S53 (TRaSH Guides import) can add it. |
| Grabbing and importing packs and multi-episode releases | S33 (Import pipeline) and S34 (Automatic search, Backlog search and RSS sync). Packs are read, shown and refused as `covers_more` for now. |
| Air-date and absolute search queries, and propers on final cells | S34 (Automatic search, Backlog search and RSS sync). |
| The Custom Formats, release group and quality naming tokens | S24 (Naming and renaming). |
| Manual search past a soft refusal, overrides and the Blocklist | S32 (Manual search and grab). |
