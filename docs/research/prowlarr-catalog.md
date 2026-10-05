# Prowlarr catalog: vendor or reimplement

Research for wayfinder ticket #9. It feeds the decision behind charting answer Q12 in #2, which says Prowlarr parity means the indexer catalog only, searched internally.

This file lays out facts and options. It does not make the decision.

## Sources and snapshot

All counts were taken on 2026-10-04.

- Prowlarr source. The local clone at `~/Git/Prowlarr` (branch `develop`, `azure-pipelines.yml` sets `majorVersion: '2.3.2'`). Every count that matters was checked against upstream `Prowlarr/Prowlarr` `develop` at commit `3c6e1d97ac485a70cc9d4063dd5d17af7a77584d`. The latest release is `v2.6.5.5623` (2026-09-16), per the GitHub releases API.
- Definition catalog. The live index at `https://indexers.prowlarr.com/master/11` and the package at `https://indexers.prowlarr.com/master/11/package.zip`. Both were downloaded and tallied with a script.
- Definition repository. `github.com/Prowlarr/Indexers` (README, `VERSIONS`, `.github/workflows/ci.yml`, `definitions/v11/`).
- Jackett. `github.com/Jackett/Jackett` (LICENSE, `src/Jackett.Common/Definitions/`).
- FlareSolverr. `github.com/FlareSolverr/FlareSolverr` (README, LICENSE metadata).
- License texts. The GPL-3.0 text in `~/Git/Prowlarr/LICENSE`. The GNU GPL FAQ at `https://www.gnu.org/licenses/gpl-faq.html`. GitHub's "Licensing a repository" doc.
- SlipStream v1. `docs/research/v1-inventory/backend.md` on branch `research/v1-inventory`, and the v1 tree at `~/Git/SlipStream-v1`.

Paths below that start with `src/` are relative to the Prowlarr repository root.

## 1. How the catalog is built

Prowlarr has two kinds of indexer.

1. **Cardigann definitions.** YAML files that one generic C# class, `Cardigann`, interprets at runtime (`src/NzbDrone.Core/Indexers/Definitions/Cardigann/`).
2. **Native C# indexers.** One class per site or protocol, under `src/NzbDrone.Core/Indexers/Definitions/`.

### 1.1 Counts

| Kind | Count | Source |
| --- | --- | --- |
| Cardigann YAML definitions served for schema v11 | 577 | `indexers.prowlarr.com/master/11` returned 577 entries. `package.zip` held 577 `.yml` files. `Prowlarr/Indexers` `definitions/v11/` holds 578 files, which are 577 YAML files plus `schema.json`. |
| Of those, by privacy | 436 private, 82 public, 59 semi-private | `type` field in the index |
| Of those, by protocol | 577 torrent, 0 usenet | `protocol` field in the index |
| Concrete native C# indexer classes | 69 | Upstream `develop`. The local clone has 67. Upstream adds `AnimeZ` (an `AvistazBase`) and `DesiGaane` (a `GazelleBase`). Counted by walking each class's base chain to `TorrentIndexerBase` or `UsenetIndexerBase`. |
| Of those, marked `[Obsolete]` | 11 | `Animedia`, `BinSearch`, `CGPeers`, `FunFile`, `HDSpace`, `PornoLab`, `Rarbg`, `SceneTime`, `Shizaproject`, `Xthor`, `ZonaQ`. Reasons include "Moved to YML for Cardigann" and "Site has shutdown". `IndexerFactory.GetDefaultDefinitions` skips any provider where `IsObsolete()` is true. |
| Of those, generic protocol adapters | 5 | `Cardigann`, `Newznab`, `Torznab`, `TorrentRssIndexer`, `TorrentPotato` |
| Active site-specific native indexers | 53 | 69 minus 11 obsolete minus 5 generic. Only one is usenet (`Headphones`). |
| Newznab presets | 18 named plus "Generic Newznab" | `yield return GetDefinition(...)` calls in `Definitions/Newznab/Newznab.cs`. Examples are NZBgeek, DrunkenSlug, NZBFinder and DOGnzb. |
| Torznab presets | 3 named plus "Generic Torznab" | `Definitions/Torznab/Torznab.cs` (AnimeTosho, MoreThanTV, Torrent Network) |

The local clone also has `Unit3dBase` (`Definitions/UNIT3D/Unit3dBase.cs`). It is abstract and has no concrete subclass in either the clone or upstream `develop`.

### 1.2 What the native indexers are

The native classes cover sites whose API or auth does not fit Cardigann. Seventeen definition files or folders reference an API key or API user, among them `BeyondHD`, `BroadcastheNet`, `HDBits`, `MTeamTp`, `PassThePopcorn`, `Redacted` and `Orpheus` (grep of `src/NzbDrone.Core/Indexers/Definitions`). Twelve match a looser grep for passkey, 2FA, TOTP or OTP terms, among them `AnimeBytes`, `FileList`, `HDBits` and `Avistaz`. That grep can over-match, so treat twelve as an upper bound. Shared settings bases exist for no-auth, username/password and cookie sites (`src/NzbDrone.Core/Indexers/Settings/`).

Prowlarr hides 18 YAML definitions because C# now implements them. The list is `_definitionBlocklist` in `src/NzbDrone.Core/IndexerVersions/IndexerDefinitionUpdateService.cs`, with the comment "Used when moving yml to C#". It names `hdbits`, `mteamtp`, `mteamtp2fa`, `beyond-hd`, `blutopia`, `aither` and others. None of the 18 appear in the current v11 package.

Usenet support in the catalog comes from Newznab and its presets, not from YAML. Every v11 YAML definition is a torrent indexer (section 1.1).

### 1.3 Where the definitions come from

The `Prowlarr/Indexers` README says the definitions sync from Jackett "with some differences and a few exceptions", run daily by GitHub Actions through `scripts/indexer-sync-v2.sh`, with problem indexers excluded by `scripts/blocklist.txt`. A filename comparison found that 574 of Prowlarr's 577 v11 definitions share a filename with Jackett's 583 definitions in `src/Jackett.Common/Definitions/`. The three Prowlarr-only files are `brasiltracker`, `theparadiese` and `torrentoyunindir`. Matching filenames does not prove identical content. That part is inferred.

## 2. How definitions are versioned and shipped

### 2.1 Schema versions

The schema has a version number, and each version needs matching runtime code. The `Prowlarr/Indexers` README says "Versions require Prowlarr Cardigann C# modifications." `VERSIONS` sets `CURRENT_VERSION=11`, `MIN_VERSION=11`, `MAX_VERSION=11` and `NEXT_VERSION=12`. The README marks v11 as ACTIVE and says v10 is deprecated, with all indexers moved to v11 as of 2025-08-24. It lists what each version added. For example, v3 added JSON APIs, v5 JSON filters, v8 `htmlencode` and `htmldecode`, v9 `allowEmptyInputs` and `missingAttributeEqualsNoResults`, v10 the `info_cookie`, `info_flaresolverr` and `info_useragent` setting types, and v11 `info_category_8000` and optional `selectorinputs` and `getselectorinputs`.

Each version folder holds a `schema.json`. CI lints every definition with `yamllint` and validates it against its schema with `scripts/validate.py` (`.github/workflows/ci.yml`). A reimplementation can validate against the same `schema.json`.

### 2.2 Distribution

`IndexerDefinitionUpdateService` hard-codes `DEFINITION_BRANCH = "master"` and `DEFINITION_VERSION = 11`. Upstream `develop` has the same constants and blocklist. The only difference from the clone is internal refactoring. The service does four things.

1. **List.** `GET https://indexers.prowlarr.com/master/11` returns a JSON array of meta definitions. Each entry carries `id`, `file`, `name`, `description`, `type`, `language`, `links`, `settings`, `caps`, `login` (captcha only, in practice), `protocol`, `implementation` and `sha` (`CardigannMetaDefinition.cs`, and the live response). If the request fails, the service reads `*.yml` from `<AppData>/Definitions` instead.
2. **Bulk update.** On application start, and every 24 hours (`src/NzbDrone.Core/Jobs/TaskManager.cs`, `Interval = 24 * 60`), it downloads `.../master/11/package.zip`, extracts it into `<AppData>/Definitions`, and clears its cache. The package was 1.24 MB on 2026-10-04.
3. **Single fetch.** If a definition is missing on disk, it fetches `.../master/11/<id>`.
4. **Custom definitions.** It merges `*.yml` from `<AppData>/Definitions/Custom` into the list, and skips any duplicate file name or indexer name.

The service also normalizes each definition on load. It defaults `encoding` to UTF-8 and `login.method` to `form`, adds username and password settings when a definition has none, and rewrites a single `search.path` into `search.paths` (`CleanIndexerDefinition`).

The README says "Prowlarr will fall back to a previous version if no YML exists for the current version." A comment in the service says the same.

## 3. What the definition runner must support

The C# runtime is 3,537 lines across 11 files in `Definitions/Cardigann/`. The largest are `CardigannRequestGenerator.cs` (1,235), `CardigannBase.cs` (930) and `CardigannParser.cs` (726). It uses AngleSharp for HTML and CSS selectors and Newtonsoft `JObject` for JSON (the `using` lines in `CardigannRequestGenerator.cs`).

The data model is in `CardigannDefinition.cs`. It has the blocks `settings`, `caps` (categories, category mappings, search modes, `allowrawsearch`), `login`, `ratio`, `search` (paths, inputs, headers, keyword filters, preprocessing filters, rows, fields, error), and `download` (selectors, method, `before`, `infohash`, headers).

These usage counts come from a tally of the 577 v11 files.

| Feature | Definitions using it |
| --- | --- |
| Search response type `html` / `json` / `xml` | 420 / 135 / 22 |
| `{{ if }}...{{ else }}` templates | 523 |
| `{{ range }}` templates | 412 |
| `or`, `and`, `eq`, `ne` template functions | 172, 156, 149, 7 |
| `join` / `re_replace` template functions | 28 / 15 |
| `requestDelay` | 74 |
| `legacylinks` | 201 |
| `certificates` | 15 |
| `download.before` | 15 |
| `download.infohash` | 8 |
| Distinct languages | 38 |

Twenty-two distinct filters are used. By use count they are `re_replace` 1555, `replace` 841, `dateparse` 589, `append` 587, `regexp` 313, `querystring` 287, `andmatch` 60, `prepend` 42, `split` 41, `trim` 34, `timeago` 28, `fuzzytime` 20, `diacritics` 14, `validate` 14, `htmldecode` 13, `validfilename` 8, `tolower` 5, `urldecode` 4, `toupper` 3, `urlencode` 3, `jsonjoinarray` 2 and `timeparse` 1. `andmatch` is a row filter that `CardigannParser.cs` handles separately. The others are in the switch in `CardigannBase.cs`.

The template language is not Go's `text/template`. `ApplyGoTemplateText` in `CardigannBase.cs` is a regex-driven subset. It evaluates `re_replace` and `join` first, then the logic functions `and`, `or`, `eq` and `ne`, then `if`/`else`/`end`, then `range`. An `if` condition must be a variable reference, or the code throws `NotImplementedException`. A runner built on a real template engine can diverge from these semantics. That is inferred, not tested.

## 4. Auth flows

### 4.1 Cardigann login methods

| `login.method` | Definitions | What it requires (code in `CardigannRequestGenerator.cs`) |
| --- | --- | --- |
| `cookie` | 141 | The user pastes a browser cookie string. Prowlarr's UI shows step-by-step DevTools instructions for `info_cookie` (`src/Prowlarr.Api.V1/Indexers/IndexerResource.cs`). Fourteen definitions also ask for the browser's User-Agent (`info_useragent`). |
| `get` | 134 | Templated inputs go into a query string on the login path. |
| `form` (explicit or default) | 125 | The runner fetches the login page, finds the form (default selector `form`), collects its inputs, and fills them. It supports `selectors`, `selectorinputs` and `getselectorinputs`. It auto-solves a `simpleCaptcha` widget if the page includes one. |
| `post` | 92 | A direct POST of templated inputs. |
| none | 85 | No login. |
| `oneurl` | 0 in v11 | The code supports it. No v11 definition uses it. |

All 46 definitions with a `login.captcha` use `type: image`. The runner fetches the image during setup (`GetConfigurationForSetup` and `GetCaptcha`), so the UI must show the image and accept the typed answer. `IndexerFactory.MapCardigannDefinition` adds a `cardigannCaptcha` field for these. Any other captcha type throws `NotImplementedException`.

Definitions can also declare a `login.test` page and selector to confirm the session, and `login.error` selectors to read failure messages (`LoginBlock`, `ErrorBlock`).

### 4.2 Native indexer auth

Native classes carry per-site settings such as API keys, passkeys, RSS keys and 2FA codes (section 1.2). Each site needs its own code.

## 5. FlareSolverr and proxies

**What FlareSolverr is.** A separate MIT-licensed service (GitHub license metadata). Its README says it uses Selenium with undetected-chromedriver to run Chrome, solve the Cloudflare or DDoS-GUARD challenge, and return the HTML and cookies. It recommends Docker "because the project depends on an external browser". It warns that each request launches a browser and uses a lot of memory. The latest release is v3.5.2 (2026-09-12).

**How Prowlarr uses it.** FlareSolverr is one of four indexer proxy types, next to HTTP, SOCKS4 and SOCKS5 (`src/NzbDrone.Core/IndexerProxies/`). `FlareSolverr.cs` is 303 lines and works in five steps.

1. It sends the original request first, with HTTP errors suppressed.
2. It checks the response with `CloudFlareDetectionService.IsCloudflareProtected`. That check needs a `Server` header of `cloudflare`, `cloudflare-nginx` or `ddos-guard`, plus a 403 or 503 status with one of several known challenge page titles. A separate heuristic catches some custom-protected sites.
3. Only when the check succeeds, it POSTs `{cmd: "request.get" | "request.post", url, maxTimeout, proxy}` to `<host>/v1`. POST is supported only for `application/x-www-form-urlencoded`. The default host is `http://localhost:8191/` and the default timeout is 60 s (`FlareSolverrSettings.cs`).
4. It caches the returned User-Agent per host, injects the returned cookies, and replays the original request directly.
5. It forwards Prowlarr's own HTTP or SOCKS proxy settings to FlareSolverr.

**Which indexers use it.** Proxies attach to indexers by tag. An indexer uses a proxy that shares one of its tags, and FlareSolverr runs last in the chain (`IndexerHttpClient.GetProxies`). Fifty-eight v11 definitions carry an `info_flaresolverr` setting, which the UI shows as a note that the site may need FlareSolverr. The runner treats `info_*` setting types as no-ops (`CardigannBase.cs`).

**What a built-in catalog needs.** It can call an external FlareSolverr over HTTP, or it can run a headless browser itself. The second choice puts Chrome inside SlipStream's packaging. That is inferred from the README's Docker note.

## 6. Licenses

### 6.1 The texts

| Component | License | Source |
| --- | --- | --- |
| Prowlarr (C# code, including the Cardigann runtime and native indexers) | GPL-3.0 | `LICENSE` in the clone, README "GNU GPL v3", GitHub `license.spdx_id` |
| `Prowlarr/Indexers` (the YAML definitions) | **None declared** | GitHub reports `license: null`. The repository root has no LICENSE file. |
| Jackett (where 574 of 577 definitions share a name) | GPL-2.0 | `LICENSE` is GPL version 2. GitHub reports `GPL-2.0`. Neither the README nor the API says "or later". The "or later" status is unknown. |
| Original Cardigann (Go, by lox) | MIT | `cardigann/cardigann`. Last push 2021-02-22. |
| FlareSolverr | MIT | GitHub license metadata |
| SlipStream | None declared | Private repository. No LICENSE file on `main`. |

Prowlarr contributors sign a CLA (`CLA.md`). It grants Prowlarr a license to the contribution, and binds Prowlarr to license contributions "only under the terms of the license or licenses which We are using on the Submission Date". It grants nothing to third parties.

### 6.2 What GPL-3.0 and the FSF FAQ say

- GPL-3.0 §0 says "To 'convey' a work means any kind of propagation that enables other parties to make or receive copies. Mere interaction with a user through a computer network, with no transfer of a copy, is not conveying."
- GPL-3.0 §2 says "You may make, run and propagate covered works that you do not convey, without conditions so long as your license otherwise remains in force." The FAQ entry `#GPLRequireSourcePostedPublic` says private modification and use carry no release duty.
- GPL-3.0 §5(c) says that when you convey a modified work, "You must license the entire work, as a whole, under this License".
- GPL-3.0 §5 (final paragraph) and FAQ `#MereAggregation` cover aggregates. Separate programs distributed together stay separate. The FSF's criterion weighs the communication mechanism and its semantics. FAQ `#GPLPlugins` says fork and exec with intimate exchange of complex data structures can make one program, while simple fork and exec does not.
- FAQ `#IfInterpreterIsGPL` and `#InterpreterIncompat` say an interpreted program is "just data" to the interpreter. A GPL interpreter does not impose its license on the programs it runs, and a non-GPL interpreter can run GPL programs.
- FAQ `#v2v3Compatibility` says GPLv2-only and GPLv3 are not compatible. Code under "version 2 or later" is compatible.
- GitHub's licensing doc says that without a license "the default copyright laws apply ... no one may reproduce, distribute, or create derivative works from your work". GitHub's terms still let other users view and fork a public repository.

### 6.3 Inferences for SlipStream

These are inferred from the texts above. They are not legal advice.

1. SlipStream will be conveyed, because Q3 says it is packaged for strangers and Q24 says it ships packages and Docker images. So the conveying clauses apply to anything GPL that SlipStream ships inside one work.
2. Linking or compiling Prowlarr's C# into SlipStream makes the whole of SlipStream subject to GPL-3.0 when it is conveyed (§5(c)).
3. Translating Prowlarr's C# runner or native indexers into another language is very likely a derivative work. GPL-3.0 §0 defines "modify" as adapting a work "in a fashion requiring copyright permission". The text does not name translation outright.
4. A runner written without copying Prowlarr's code, which reads definition YAML as data, is not bound by the definitions' license under the FSF's interpreter reading (`#IfInterpreterIsGPL`).
5. Fetching definitions from `indexers.prowlarr.com` at runtime, as Prowlarr does, means SlipStream does not distribute them. Bundling the YAML in a SlipStream package or image does distribute them. The `Prowlarr/Indexers` repository declares no license, and Jackett's copies are GPL-2.0. The license that would cover a bundled copy is unclear.
6. If the bundled YAML counts as GPL-2.0-only and SlipStream's code is GPL-3.0, the two can sit side by side as an aggregate. They cannot be merged into one work (`#v2v3Compatibility`). Whether YAML data merged into a binary is "one work" is a legal question. The FSF FAQ leaves that to the mechanism and semantics test.
7. No terms of use for `indexers.prowlarr.com` were found in the sources read. Whether a third-party client may use it is unknown.

## 7. SlipStream v1's starting point

v1 already reimplements the runner in Go. `docs/research/v1-inventory/backend.md` section 6 lists the parts, and the files are in `~/Git/SlipStream-v1/internal/indexer/cardigann/`.

- **Size.** 5,666 lines including 861 lines of tests (`wc -l`).
- **Template engine.** Go `text/template` (`template.go`), not a port of Prowlarr's regex subset.
- **Login methods.** `post`, `form`, `cookie`, `oneurl` and `get` (`login.go`).
- **Captcha.** `definition.go` marks `CaptchaBlock` "not fully implemented".
- **Definition source.** `https://indexers.prowlarr.com`, version `"11"` (`repository.go`). Definitions download into `data/definitions/`, which `.gitignore` excludes, so v1 does not ship them.
- **Missing filters.** `andmatch` (used by 60 definitions), `validfilename` (8), `jsonjoinarray` (2) and `timeparse` (1) are absent. `applyFilter` in `filters.go` returns the input unchanged for an unknown filter name, so these fail silently.
- **FlareSolverr.** Not integrated. `login.go` only returns an error that suggests FlareSolverr "may be required".
- **External mode.** v1 also has a separate external Prowlarr mode that sends all search through Prowlarr's API (`internal/prowlarr/`).
- **Provenance.** Whether v1's runner was written from Prowlarr's C# or from the YAML and wiki is not recorded in the sources read. Unknown.

## 8. Options for the decision

Each option below lists the facts that bear on it. Tradeoffs marked inferred are not measured.

### A. Run Prowlarr's own code

- **A1. Link or embed it.** This needs a .NET stack (Q4 leaves the stack open). It makes SlipStream GPL-3.0 when conveyed (6.3 item 2). It brings the 69 native indexers, the runner and FlareSolverr support with no port.
- **A2. Ship Prowlarr as a separate process that SlipStream calls over its API.** This is the FSF's aggregate pattern (§5, `#MereAggregation`). v1's external Prowlarr mode already works this way. It adds a .NET runtime and a second process to every package. Q12 asks for the catalog to be "searched internally", which A2 meets only in packaging, not in code (inferred).

### B. Port Prowlarr's runner and native indexers

- **Size.** About 3.5k lines of runner, plus 53 active site-specific native indexers and 2 generic protocol adapters (Newznab, Torznab) with 21 presets between them.
- **License.** Very likely a derivative work, so GPL-3.0 when conveyed (6.3 item 3).
- **Fidelity.** Matches Prowlarr's quirks, such as the template subset, by construction.

### C. Reimplement the runner from the schema and consume the same definitions

- **Starting point.** v1 is this option in Go (section 7). Its known gaps are four filters (71 definition uses), image captcha, and FlareSolverr.
- **Spec.** `schema.json` per version, the wiki definition page, and the 577 live files act as the spec and the corpus. A conformance run over the corpus is possible because CI already validates every file against the schema (inferred).
- **License.** SlipStream's license is unconstrained by Prowlarr's when no code is copied (6.3 item 4).
- **Native indexers.** The 53 site-specific ones still need separate code per site under any option except A.

### D. Where definitions come from (applies to B and C)

- **D1. Runtime fetch from `indexers.prowlarr.com/master/11`.** This is what Prowlarr and v1 do. SlipStream distributes nothing. It depends on Servarr infrastructure and its unstated terms. It is pinned to schema 11, and a move to `NEXT_VERSION=12` needs runner changes. Offline CI with egress blocked (Q17) cannot reach it, so the offline suite needs its own fixture definitions (inferred).
- **D2. Mirror or bundle the YAML in SlipStream.** This makes SlipStream a distributor of files with no declared license, or GPL-2.0 in Jackett's copies (6.3 items 5 and 6).
- **D3. Source from Jackett directly.** A GPL-2.0 repository with 583 definitions in a flat folder. The Prowlarr README notes "some differences" between the two formats, so the schemas are not identical.
- **D4. SlipStream-authored definitions only.** No third-party license question. The catalog starts empty except for what SlipStream writes. Test-only definitions for the offline suite fit here (inferred).

### E. FlareSolverr (independent of A to D)

- **E1. Talk to an external FlareSolverr over its HTTP API, as Prowlarr does.** MIT, so no license constraint. The user runs one more container.
- **E2. Embed a headless browser.** Chrome goes into SlipStream's packaging and memory budget (FlareSolverr README).
- **E3. Leave it out.** The 58 definitions flagged `info_flaresolverr` may fail behind Cloudflare.

## 9. Open facts not settled here

- The license of the YAML in `Prowlarr/Indexers`, and whether Jackett is GPL-2.0-only or "or later". Asking either project would settle it.
- The terms of use for `indexers.prowlarr.com`.
- The provenance of v1's Go runner.
- How many v11 definitions v1's runner actually passes end to end. Answering that needs a conformance run, which this ticket did not do.
