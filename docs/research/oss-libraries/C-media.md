# C. Media domain libraries: parsing, probing, subtitles, rules, naming, search

> Spec numbers follow the feature graph as renumbered on 2026-10-11, with seven foundation specs (S01 to S07) ahead of the walking skeleton (S09). "S01-S09" means the foundations plus the skeleton.


Scope: release name parsing, .torrent/magnet inspection, file probing, subtitles, Custom Format and Target rules, naming templates and path sanitizing, library and global search, reference data. Researched 2026-10-11.

**How the data was gathered.** The GitHub API was blocked in this session. I verified each module's latest tag and date through `proxy.golang.org/<module>/@latest`, its license through deps.dev, and its stars and open issues through deps.dev's project mirror. I read LICENSE files and source on raw.githubusercontent.com. "Verified" means one of those calls this session. Commit months beyond the latest tag are unverified. I ran four throwaway probes (FTS5, lingua-go binary size, anacrolix metainfo dependency graph, and the TestCase counts in the Sonarr/Radarr test fixtures) in the scratchpad. Neither repo was touched.

**Cross-cutting finding.** None of these primitives is a network boundary, so ADR 0014's fake-ability concern applies only to ffprobe (a child process). The decisive constraints are ADR 0006 (Custom Formats run verbatim), ADR 0008 (cue text is never edited), ADR 0011 (naming) and S53 (TRaSH naming and format import). Each of these pulls toward porting Sonarr/Radarr's own semantics rather than adopting a third-party library with different semantics.

---

## 1. Release name parsing (S23; consumed by S32, S33, S34, S53, S55)

**Current plan.** ADR 0001 makes Coverage parsing (SxxEyy, absolute numbers, daily dates, year) the Content Type's job. The release-facts half has no owner yet: source, resolution, codec, HDR, audio, group, language, proper/repack/real, edition and quality modifiers. Custom Formats (ADR 0006) consume these facts, so the core needs one shared release-facts parser. That parser is unspecified, so it would be custom by default.

| Candidate | License | Latest release | Signal | Verified |
|---|---|---|---|---|
| [Sonarr Parser.cs + QualityParser.cs + LanguageParser.cs](https://github.com/Sonarr/Sonarr/tree/develop/src/NzbDrone.Core/Parser) | GPL-3.0 (LICENSE.md read) | rolling `develop` | Parser.cs 1,373 lines, QualityParser.cs 725; a large team; the reference semantics for TRaSH | yes |
| [Radarr Parser.cs](https://github.com/Radarr/Radarr/tree/develop/src/NzbDrone.Core/Parser) | GPL-3.0 (inferred, same fork lineage) | rolling | Parser.cs 643 lines; movie and edition rules | sizes yes, license inferred |
| [MunifTanjim/go-ptt](https://github.com/MunifTanjim/go-ptt) | MIT | v0.14.1, 2026-02-16 | 8 stars, 1 maintainer; Go port of Python PTT; tuned for debrid and streaming names | yes |
| [dreulavelle/jhin](https://github.com/dreulavelle/jhin) | MIT | v0.8.0, 2026-09-16 | 2 stars, first released 2026; 46 fields, 1,160-title golden corpus, built-in ranking | yes |
| [moistari/rls](https://github.com/moistari/rls) | MIT | v0.6.0, 2025-06-28 | 25 stars; a lexer design; autobrr's parser (inferred) | yes |
| [middelink/go-parse-torrent-name](https://github.com/middelink/go-parse-torrent-name) | MIT | pseudo-version, 2019-03-01 | 51 stars, abandoned | yes |
| [razsteinmetz/go-ptn](https://github.com/razsteinmetz/go-ptn) | MIT | v1.0.0, 2022-09-27 | 7 stars, stale | yes |
| [f4n4t/go-release](https://github.com/f4n4t/go-release) | MIT | v0.11.2, 2026-05-09 | 1 star; scene release checking (SFV/zip) | yes |
| [guessit](https://github.com/guessit-io/guessit) (Python) | LGPL-3.0 (LICENSE read) | unverified | the mature reference; Python only | license yes |
| [anitomy](https://github.com/erengy/anitomy) (C++) / [nssteinbrenner/anitogo](https://github.com/nssteinbrenner/anitogo) | MPL-2.0 / MPL-2.0 | anitogo v1.0.0, 2023-12-22 | anitogo has 26 stars and is quiet | yes |
| [autobrr](https://github.com/autobrr/autobrr) | **GPL-2.0** (LICENSE read; whether "or later" applies is unverified) | v1.88.0, 2026-10-06 | 3,130 stars | yes |

**Verdict: `borrow`.** Port Sonarr's and Radarr's Parser, QualityParser, LanguageParser and ReleaseGroup logic to Go on `regexp2`, which ADR 0013 already includes. Adopt none of the Go libraries.

- **Why.** TRaSH Custom Formats assume Sonarr/Radarr's parse. Their Source, Resolution, QualityModifier, ReleaseType and Language conditions match the enum values *Sonarr* derives from a title, not the values PTT or guessit derive. ADR 0006 keeps formats byte-identical to upstream, which is only meaningful if the facts they test are derived the same way.
- **Why not the Go ports.** They are bus-factor-1 or abandoned, and their field vocabularies (PTT's "quality", jhin's 46 fields) do not map 1:1 to Sonarr's specification enums. jhin's `rank` package also overlaps S23's scoring, a second ranking model SlipStream would not use.
- **Porting regexes.** Sonarr's regexes use .NET constructs such as lookbehind and atomic groups, so porting them verbatim needs `regexp2`, not Go's `regexp`.
- **Cost of building.** Roughly 4-6k lines of Go across the core facts parser and the movie and TV Coverage parsers. The main risk is drift as Sonarr/Radarr keep tuning; mitigate it by tracking their test fixtures.
- **Cost of a dependency.** A parser whose output disagrees with TRaSH's expectations, and a small project to depend on for critical behavior.
- **Anime.** Sonarr's absolute-number rules already cover the TV Content Type. anitomy/anitogo are worth reading for test titles only.

**Fixture corpus (license check).** Sonarr's and Radarr's test fixtures are GPL-3.0, the same license as SlipStream (ADR 0010), so they can be copied with attribution in the notices file. TestCase counts (verified):

| Repo | Fixture | TestCases |
|---|---|---|
| Sonarr | SingleEpisode | 176 |
| Sonarr | MultiEpisode | 72 |
| Sonarr | AbsoluteEpisodeNumber | 161 |
| Sonarr | Season | 49 |
| Sonarr | Quality | 278 |
| Sonarr | Language | 168 |
| Sonarr | ReleaseGroup | 117 |
| Sonarr | Daily | 45 |
| Sonarr | AnimeMetadata | 36 |
| Radarr | Parser | 115 |
| Radarr | Quality | 273 |
| Radarr | Language | 195 |
| Radarr | ReleaseGroup | 154 |
| Radarr | Edition | 49 |

That is about 1,100 Sonarr and 790 Radarr titles. Supplementary corpora, all GPL-3-compatible:

- guessit's `movies.yml` (252 cases) and `episodes.yml` (622 cases), LGPL-3.0.
- jhin's 1,160-title golden corpus, MIT.
- PTT's tests, MIT (inferred).

Fixtures from those sources need an "expected" column rewritten to Sonarr semantics, so use them as fuzz and no-crash inputs, not as golden outputs.

- **Criticality:** critical. A wrong parse grabs the wrong release, fails to upgrade, or breaks hit-and-run ratio on private trackers.
- **ADR conflicts:** none. Adopting PTT, guessit or jhin semantics would quietly undermine ADR 0006.

## 2. Bencode, .torrent and magnet inspection on the Controller (S26, S32, S34)

**Current plan.** Unspecified. ADR 0013 links libtorrent through cgo into the single binary, so libtorrent is present on the Controller too, even when the Controller holds no Downloading Role.

| Candidate | License | Latest release | Signal | Verified |
|---|---|---|---|---|
| libtorrent via the existing shim (`load_torrent_buffer`, `parse_magnet_uri`) | BSD | pinned by ADR 0013 | heavily fuzzed bdecoder; the same infohash code as the engine | inferred (API names from memory) |
| [anacrolix/torrent](https://github.com/anacrolix/torrent) `bencode` + `metainfo` | MPL-2.0 (LICENSE read) | v1.61.0, 2025-12-17 | 6,123 stars, 93 open issues; BEP 52 v2 and hybrid supported (`MetaVersion`, `FileTree` in info.go) | yes |
| [jackpal/bencode-go](https://github.com/jackpal/bencode-go) | BSD-3-Clause | v1.2.0, 2026-08-31 | 293 stars, 0 open issues | yes |
| [zeebo/bencode](https://github.com/zeebo/bencode) | MIT | v1.0.0, 2018-08-19 | 92 stars; stale | yes |
| [cristalhq/bencode](https://github.com/cristalhq/bencode) | MIT | v0.4.0, 2023-08-30 | 11 stars | yes |

**Measured.** Importing only `anacrolix/torrent/metainfo` compiles in about 20 third-party packages: missinggo v1 and v2, go-multihash with murmur3, lukechampine blake3, cpuid, base58, huandu/xstrings and others. The module's full `go.mod` graph is much larger, which loads the CI license allow-list (ADR 0010).

**Verdict: `build` a small pure-Go decoder, with libtorrent as the oracle in tests.** What the Controller needs is narrow:

- the v1 infohash (SHA-1 over the exact raw `info` bytes) and the v2 infohash (SHA-256)
- the `private` flag, trackers and the file list with sizes (for Coverage per file before grab)
- magnet `xt`/`dn`/`tr` fields

- **Cost of building.** A strict bencode decoder that keeps raw `info` bytes and rejects non-canonical input is about 250-400 lines plus a fuzz target.
- **Testing.** Cross-check infohashes against libtorrent in a test so the two never disagree.
- **Why not libtorrent directly.** Routing untrusted indexer bytes through cgo works too, and avoids two parsers. It is the alternative if the maintainer wants exactly one parser.
- **Why not anacrolix.** It is the best Go option if a dependency is preferred, but it brings a large dependency graph for about 300 lines of value. jackpal/bencode-go is light, but it does not expose raw `info` bytes for hashing (inferred from its reflection-based API, unverified).
- **Criticality:** important. A wrong infohash breaks dedupe, grab tracking and ADR 0016's hand-off.
- **ADR conflicts:** none.

## 3. File probing (S22, S33, S44, S45)

**Current plan.** Decided. ADR 0013 and ADR 0007 make jellyfin-ffmpeg's ffprobe and `dovi_tool` child processes. Only the Go-side wrapper is open.

| Candidate | License | Latest release | Signal | Verified |
|---|---|---|---|---|
| [vansante/go-ffprobe/v2](https://github.com/vansante/go-ffprobe) | MIT | v2.3.1, 2026-09-14 | 223 stars, 1 open issue; has `SideDataList` | yes |
| [remko/go-mkvparse](https://github.com/remko/go-mkvparse) | MIT | v0.14.0, 2022-01-30 | 41 stars; stale | yes |
| [at-wat/ebml-go](https://github.com/at-wat/ebml-go) | Apache-2.0 | v0.19.4, 2026-09-30 | 92 stars; WebM-oriented | yes |
| [coding-socks/matroska](https://github.com/coding-socks/matroska) | MIT | pseudo-version, 2026-08-31 | 5 stars | yes |
| [abema/go-mp4](https://github.com/abema/go-mp4) | MIT | v1.7.3, 2026-09-08 | 548 stars | yes |
| MediaInfo (libmediainfo) | BSD-2-Clause (inferred) | unverified | C++; a new native dependency on every OS | no |

**Verdict: `build` (about 150-250 lines), borrowing go-ffprobe's struct shapes.**

- **Why not go-ffprobe.** It stores the binary path in a package global (`var binPath = "ffprobe"` plus `SetFFProbeBinPath`, read in v2's ffprobe.go). That conflicts with ADR 0007's per-Node binary and admin override, and with ADR 0014's need to inject a fake capability source. A custom wrapper takes an exec seam (`Runner` interface), so the offline suite can return canned JSON.
- **How to detect HDR formats.** All from the ffprobe JSON:
  - Dolby Vision: the stream `side_data_list` entry "DOVI configuration record" (`dv_profile`, `dv_level`, `rpu_present_flag`, `el_present_flag`, `dv_bl_signal_compatibility_id`).
  - HDR10: `color_transfer=smpte2084` plus mastering display or content light side data.
  - HLG: `arib-std-b67`.
  - HDR10+: needs `-show_frames -read_intervals "%+#1"` to read the first frame's "HDR Dynamic Metadata SMPTE2094-40" side data.
  - These side-data names are from memory and unverified. Pin them with fixtures generated from the bundled ffprobe.
  - Use `dovi_tool` (already decided) only when the RPU itself matters (profile 7 FEL vs MEL, or converting to 8.1).
- **Not needed.** MKV/EBML and MP4 parsers add nothing that ffprobe doesn't, so reject them. MediaInfo gives friendlier HDR strings but adds a C++ library per OS; reject.
- **Criticality:** important. A wrong probe mis-fills Targets, but data is not lost.
- **ADR conflicts:** MediaInfo or a second probe engine would sit beside ADR 0007's single bundled toolchain.

## 4. Subtitles: parse and write, charset, language, codes (S46, S47)

**Current plan.** ADR 0008: timing sync is SlipStream's own code (ported from ffsubsync and alass), and SlipStream changes only timing, text encoding and format, never cue text. Parsing and writing are unspecified.

| Candidate | License | Latest release | Signal | Verified |
|---|---|---|---|---|
| [asticode/go-astisub](https://github.com/asticode/go-astisub) | MIT | v0.45.0, 2026-09-17 | 715 stars, 9 open issues; SRT, SSA/ASS, VTT, TTML, STL, teletext | yes |
| [martinlindhe/subtitles](https://github.com/martinlindhe/subtitles) | MIT | pseudo-version, 2025-11-12 | 48 stars | yes |
| [saintfish/chardet](https://github.com/saintfish/chardet) (ICU port) | MIT | pseudo-version, 2023-01-01 | 383 stars; quiet | yes |
| [gogs/chardet](https://github.com/gogs/chardet) | MIT | pseudo-version, 2021-11-20 | fork of saintfish; stale | yes |
| [wlynxg/chardet](https://github.com/wlynxg/chardet) (port of Python chardet) | MIT declared; upstream Python chardet is LGPL (inferred), so provenance is muddy but GPL-3-compatible either way | v1.0.5, 2026-05-06 | 36 stars | yes |
| `golang.org/x/text/encoding/...` | BSD-3-Clause | v0.43.0, 2026-10-09 | Go team | yes |
| [pemistahl/lingua-go](https://github.com/pemistahl/lingua-go) | Apache-2.0 | v1.4.0, 2023-09-05 | 1,365 stars; **adds about 132 MB to the binary with all languages (measured: 134.9 MB vs 2.2 MB baseline)** | yes |
| [abadojack/whatlanggo](https://github.com/abadojack/whatlanggo) | MIT | v1.0.1, 2019-03-06 | 696 stars; stale; adds about 0.5 MB (measured) | yes |
| `golang.org/x/text/language` (+ `/display`) | BSD-3-Clause | v0.43.0 | BCP 47 and ISO 639-1/2/3 parsing, plus display names | yes (module); API inferred |

**Verdicts:**

- **Parse and write: `build`, borrowing astisub's fixtures.**
  - **ASS loses data.** astisub's SSA reader skips every section it doesn't know (`ssa.go`: unknown section → `continue`), so `[Fonts]`, `[Graphics]` and `[Aegisub …]` blocks are lost on a round trip.
  - **SRT text is re-tokenized.** Its SRT reader re-tokenizes cue text through `x/net/html` and rebuilds it from styled line items.
  - **Conflict with ADR 0008.** A retime through astisub can therefore change cue bytes, which ADR 0008 forbids. A purpose-built model keeps each cue's raw text bytes and every non-event line verbatim, and rewrites only timestamps.
  - **Cost.** SRT, VTT and ASS parse and write is about 600-900 lines plus tests.
  - **Format conversion.** Conversion (SRT↔VTT, ASS→SRT) is the one place that must transform markup. Write it narrowly and test it against astisub's outputs as an oracle.
- **Charset: `wrap` saintfish/chardet behind a detector that uses the known language.** The Wanted Language is known when a subtitle is fetched (Bazarr does the same).
  - Check for a BOM first, then for valid UTF-8.
  - Then try the candidate code pages for that language (for example cp1250/ISO-8859-2 for Czech, cp1256 for Arabic, GB18030/Big5 for Chinese) and score them with chardet's confidences.
  - Transcode with `x/text/encoding`.
  - saintfish is stale but small and finished. Copying it in (MIT, about 2k lines) is also fine.
- **Language detection: `adopt` whatlanggo or `borrow` its trigram tables. Do not adopt lingua-go.**
  - Subtitle files are long, so a trigram model is accurate enough to catch a mislabelled language (S47 Mark bad).
  - lingua-go's 132 MB cost is disqualifying for a single binary shipped to every Node.
- **Codes: `adopt` `x/text/language`.** It covers ISO 639-1 from TMDB, ISO 639-2/3 from TVDB and subtitle providers, and the BCP 47 tags providers send. Add a small table for release-name language tokens (VOSTFR, MULTi, ITA, etc.) ported from Sonarr's LanguageParser.
- **Criticality:** important. A wrong charset produces mojibake. A cue-text edit breaks the provider terms that ADR 0008 relies on.
- **ADR conflicts:** adopting astisub for retiming would conflict with ADR 0008.

## 5. Rules: Custom Formats and Target hard filters (S17, S23, S32, S53)

**Current plan.** ADR 0006 runs TRaSH Custom Formats verbatim, with .NET regexes on `regexp2` and a per-match timeout. A format is a list of typed specifications (ReleaseTitle, Source, Resolution, Language, ReleaseGroup, Edition, IndexerFlag, Size, QualityModifier, ReleaseType), each with `negate` and `required`. Tracking per field by `trash_id` (ADR 0006) needs the stored shape to be upstream's JSON.

| Candidate | License | Latest release | Signal | Verified |
|---|---|---|---|---|
| [expr-lang/expr](https://github.com/expr-lang/expr) | MIT | v1.17.8, 2026-02-14 | 8,038 stars, 96 open issues | yes |
| [google/cel-go](https://github.com/google/cel-go) | Apache-2.0 / BSD-3-Clause | v0.30.0, 2026-07-25 | 3,086 stars; Google-backed | yes |
| [dlclark/regexp2](https://github.com/dlclark/regexp2) | MIT | v1.12.0, 2026-04-18 | 1,193 stars; already decided (ADR 0013) | yes |

**Verdict: `build` a typed Go model mirroring Sonarr/Radarr's specification classes.**

- **Why not an expression language.** It would force a translation step from TRaSH JSON into expressions, the same "translator reruns on every upstream change" cost ADR 0006 rejected for RE2. It also breaks per-field diffing, because the stored form would no longer be upstream's.
- **Cost of building.** Evaluation is AND over required specs and OR within each kind, about 300-500 lines, and the editor and tester UI become schema-driven from the same types.
- **Target hard filters (size ranges, qualities)** are a fixed set and also belong in typed structs.
- **Criticality:** critical. The rules decide every grab.
- **ADR conflicts:** expr or cel as the *storage* form would reopen ADR 0006.

## 6. Naming templates and path sanitizing (S24, S33, S48, S53)

**Current plan.** ADR 0011 fixes the `{Target}` and Edition tokens, the `{edition-<Cut>}` placement rule and the save-time validation. S53 imports TRaSH naming schemes, which are written in Sonarr/Radarr token syntax: `{Series TitleYear}`, `{season:00}`, `{Episode Title:30}`, separator-prefixed tokens such as `{-Release Group}` and `{.Quality Full}`, bracketed `{[Custom Formats]}`, `{MediaInfo VideoDynamicRangeType}`, and Radarr's `{edition-{Edition Tags}}`. The engine itself is unspecified.

| Candidate | License | Latest release | Signal | Verified |
|---|---|---|---|---|
| `text/template` (stdlib) | BSD-3-Clause | Go 1.27 | n/a | n/a |
| [valyala/fasttemplate](https://github.com/valyala/fasttemplate) | MIT | v1.2.2, 2022-10-18 | 915 stars; tags only, no formats | yes |
| Sonarr / Radarr FileNameBuilder.cs (to port) | GPL-3.0 | rolling | 1,278 / 781 lines | sizes yes |
| [flytam/filenamify](https://github.com/flytam/filenamify) | MIT | v1.2.0, 2023-06-23 | 43 stars | yes |
| [kennygrant/sanitize](https://github.com/kennygrant/sanitize) | BSD-3-Clause | v1.2.4, 2018-08-06 | 342 stars; stale and HTML-oriented | yes |
| `golang.org/x/text/unicode/norm` | BSD-3-Clause | v0.43.0 | NFC/NFD | yes |

**Verdict: `borrow` Sonarr/Radarr's FileNameBuilder grammar into a custom tokenizer and renderer.**

- **Why not text/template.** It can't parse the syntax above, and TRaSH schemes must import verbatim.
- **Cost of building.** About 800-1,200 lines, made smaller because tokens come from the Content Type (ADR 0001) and the core only owns the grammar.
- **The grammar to port:** token, optional separator prefix, format suffix such as `:00` or `:30`, and case variants such as `{series title}` vs `{Series Title}`.
- **Path sanitizing: `build`, about 150 lines, porting Sonarr's `CleanFileName` and its colon-replacement modes (the colon style in S24). Rules:**
  - Strip `<>:"/\|?*` and control characters on every OS, not just Windows, because the library may be served to Windows clients over SMB.
  - Reserved names CON, PRN, AUX, NUL, COM1-9 and LPT1-9, with or without an extension.
  - Trailing dots and spaces.
  - Truncate to 255 bytes on ext4 and 255 UTF-16 units on NTFS. Truncate the title, never the `{Target}` label or the extension, to keep ADR 0011's uniqueness.
  - NFC-normalize names with `x/text/unicode/norm`, so a name written on macOS matches on Linux.
  - filenamify and sanitize cover only part of this list.
- **Criticality:** critical for the rename engine. A bad rename at mass-rename scale (ADR 0011 consequences) can orphan thousands of files from Media Servers.
- **ADR conflicts:** a text/template syntax would conflict with S53's TRaSH naming import and diverge from users' Sonarr/Radarr templates (S55).

## 7. Library and global search (S20, S41)

**Current plan.** SQLite on modernc (ADR 0013). Search is unspecified.

| Candidate | License | Latest release | Signal | Verified |
|---|---|---|---|---|
| SQLite FTS5 in `modernc.org/sqlite` | BSD-3-Clause | v1.60.1, 2026-09-29 (SQLite 3.53.4) | `ENABLE_FTS5` in compile options; `trigram remove_diacritics 1` and `unicode61 remove_diacritics 2` both work (measured: "pokemon" matched "Pokémon Origins", substring "wir" matched "The Wire"). spellfix1 is absent | **yes, measured** |
| [blevesearch/bleve/v2](https://github.com/blevesearch/bleve) | Apache-2.0 | v2.6.1, 2026-08-24 | 11,228 stars, 300 open issues; keeps its own index files | yes |
| [sahilm/fuzzy](https://github.com/sahilm/fuzzy) | MIT | v0.1.3, 2026-06-11 | 1,449 stars; in-memory subsequence | yes |
| [lithammer/fuzzysearch](https://github.com/lithammer/fuzzysearch) | MIT | v1.1.8, 2023-05-09 | 1,323 stars | yes |
| [hbollon/go-edlib](https://github.com/hbollon/go-edlib) | MIT | v1.7.0, 2025-08-19 | 607 stars; edit distances | yes |
| [antzucaro/matchr](https://github.com/antzucaro/matchr) | **GPL-2.0-or-later** | 2022-11-06 | 183 stars | yes |

**Verdicts:**

- **Library and global search: `adopt` FTS5** with an external-content table over Work titles, alternate titles and original titles, maintained by triggers in the core's migrations.
  - Typo tolerance is the only gap, because spellfix1 is not compiled in. Cover it with trigram matching plus `bm25()` ranking.
  - bleve would be a second store beside SQLite, outside backups (ADR 0018), with its own compaction. Reject.
- **Command palette over loaded data:** filter on the client. No Go library is needed.
- **Title matching for search results to a Work (S23):** borrow Sonarr's `CleanSeriesTitle` normalization. Add an edit-distance function only where needed; it is about 30 lines, so build it.
- **Criticality:** convenience. Title matching itself is critical but belongs in S23's parser port.
- **ADR conflicts:** bleve sits awkwardly with ADR 0018's whole-database backup.

## 8. Reference data and units (S19, S21, S37, S38)

| Candidate | License | Latest release | Verdict |
|---|---|---|---|
| `golang.org/x/text/language` + `/display` | BSD-3-Clause | v0.43.0, 2026-10-09 (verified) | **adopt** for ISO 639, BCP 47 and display names |
| [biter777/countries](https://github.com/biter777/countries) | BSD-2-Clause | v1.7.5, 2024-05-30 (verified); 524 stars, 22 open issues | **borrow**. Embed a generated ISO 3166-1 table (about 250 rows); TMDB and TVDB only send alpha-2/alpha-3 codes, and the UI can name them with `Intl.DisplayNames` |
| [emvi/iso-639-1](https://github.com/emvi/iso-639-1), [barbashov/iso639-3](https://github.com/barbashov/iso639-3) | MIT | 2025-07-25 / 2021-10-20 (verified) | **skip**. x/text covers them |
| [dustin/go-humanize](https://github.com/dustin/go-humanize) | MIT | v1.1.0, 2026-09-17 (verified); 4,830 stars | **adopt** only if server-rendered notification and email text (S37, S42) needs sizes and relative times. Otherwise format in the UI with `Intl` |
| `time/tzdata` (stdlib) | BSD-3-Clause | Go 1.27 | **adopt**. Blank-import it so air times in a network's time zone (S21, S38) work on Windows hosts without a zoneinfo database |
| Genres | n/a | n/a | **build**. Providers supply genre IDs and names; store them as provider data |

Criticality: convenience.

---

## Summary table

| Primitive | Specs | Verdict | Criticality |
|---|---|---|---|
| Release-facts and Coverage parser | S23, S32-S34, S53, S55 | borrow (port Sonarr/Radarr + GPL-3 fixtures) | critical |
| Bencode / .torrent / magnet | S26, S32, S34 | build (~300 lines), libtorrent as test oracle | important |
| ffprobe wrapper, HDR/DV detection | S22, S33, S44 | build (~200 lines), borrow go-ffprobe structs | important |
| Subtitle parse and write | S46 | build (keep raw cue bytes), astisub as oracle | important |
| Charset detection | S46 | wrap saintfish/chardet + x/text/encoding, with a language hint | important |
| Subtitle language detection | S46, S47 | adopt whatlanggo (not lingua-go, +132 MB) | convenience |
| Language codes | S19, S21, S46 | adopt x/text/language | convenience |
| Custom Format and Target rules | S17, S23, S53 | build typed model (no expr/cel) | critical |
| Naming grammar | S24, S53, S55 | borrow FileNameBuilder grammar | critical |
| Path sanitizing | S24, S33, S48 | build (~150 lines) | critical |
| Library search | S20, S41 | adopt FTS5 (verified in modernc) | convenience |
| Humanize, countries, tzdata | S19, S21, S37, S38 | adopt humanize if needed; borrow country table; adopt time/tzdata | convenience |

## Open questions for the maintainer

1. **Should the release-facts parser target byte-for-byte agreement with Sonarr/Radarr on their own test fixtures, so TRaSH formats behave as upstream intends?** Yes / no. *Recommend yes.* It is the only way ADR 0006's "verbatim" holds in practice.
2. **Copy Sonarr's and Radarr's ParserTests fixtures into the repo as golden files, with attribution in the notices file?** Yes / no. *Recommend yes.* They are GPL-3, compatible with ADR 0010.
3. **Which parser should read .torrent files on the Controller?** Pick one:
   - (a) a small pure-Go bencode decoder,
   - (b) the libtorrent shim,
   - (c) anacrolix/torrent/metainfo.

   *Recommend (a), with (b) as a test oracle.*
4. **Should subtitle retiming preserve every non-timestamp byte of the file, including ASS `[Fonts]`/`[Graphics]` sections, which rules out go-astisub for retiming?** Yes / no. *Recommend yes,* per ADR 0008.
5. **Use a naming grammar compatible with Sonarr/Radarr tokens instead of `text/template`?** Yes / no. *Recommend yes,* for the TRaSH naming import (S53) and users' existing templates (S55).
6. **Is FTS5 trigram search (no typo correction) enough for global and library search?** Yes / no. *Recommend yes.* Revisit only if users ask for typo tolerance.
7. **Should subtitle language verification exist at launch (whatlanggo, +0.5 MB), or should SlipStream trust provider language labels until Mark bad (S47) shows a need?** Pick one. *Recommend trusting the labels at launch and adding whatlanggo with S47.*
