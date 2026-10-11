# D. Clients for external boundaries

> Spec numbers follow the feature graph as renumbered on 2026-10-11, with seven foundation specs (S01 to S07) ahead of the walking skeleton (S09). "S01-S09" means the foundations plus the skeleton.


Area: the client side of every vendor boundary in `docs/offline-boundaries.md` rows 1–3, 7–12, 16–20, 23, 25–28 (metadata, indexers, download clients, media servers, notifiers, subtitle providers, TRaSH, Radarr/Sonarr), plus the HTTP plumbing they share.

## How this was checked (2026-10-11)

- Versions and dates come from `proxy.golang.org` (`@latest` and the default-branch pseudo-version, which gives the last-commit date). Licenses come from pkg.go.dev's detector, and I read the raw `LICENSE` file wherever a license affects the verdict. "Imported by" counts come from pkg.go.dev.
- The GitHub API and github.com HTML were blocked in this session (403), so **star counts and open-issue counts are unverified** throughout. I used pkg.go.dev "imported by" as the adoption signal instead.
- For every library whose fit depended on its internals, I downloaded the module zip and read the constructor. The question each time was whether the base URL and the `http.Client` are per-instance and injectable.
- I ran one probe against the live Prowlarr v11 catalog (`indexers.prowlarr.com/master/11/package.zip`, 579 definitions, cached in scratch only and never in the repo, per ADR 0010). It checked every selector with cascadia, every `regexp`/`re_replace` filter with Go `regexp` and `regexp2`, and every `{{…}}` string with Go `text/template`. Results are under S26.

## The rule that decides most verdicts here

ADR 0014 makes every fake wire-level. The simulation tier runs fakes over `net.Pipe` inside `testing/synctest`, and the release binary reaches them through the typed `Endpoints` block. A client library fits only when all of these hold:

1. The base URL is per instance. A package-level global fails, because two clients in one process collide and parallel tests race.
2. Its `*http.Client`, or at least its Transport/DialContext, is injectable. The simulation tier dials over a pipe, not TCP.
3. It takes the caller's `context.Context`. synctest and cancellation need it.
4. It doesn't sleep or retry on its own. Retry and rate-limit policy must be ours: the shared fault vocabulary plays 429 with Retry-After, and Indexer limits are product features.

Most wrappers in this area fail at least one of these. They also cover 5–20× more API than SlipStream calls. **v1's hand-written clients were small:** qBittorrent 701 lines, Transmission 588, Deluge 518, rTorrent 710, TMDB 1,473, TVDB 625, OMDb 310, Discord 776, Slack 405, Pushover 265, Cardigann 5,666 (non-test), arr import 3,635. That is the custom cost to weigh against a dependency.

---

## 1. Shared HTTP plumbing (S01-S09, used by every later client spec)

**Current plan:** Unspecified. Huma v2 on `net/http` is server-side only (ADR 0013).

| Candidate | License | Latest | Last commit | Signal | Notes | Verified |
|---|---|---|---|---|---|---|
| `net/http` + an in-repo RoundTripper chain | BSD-3 (Go) | Go 1.27 | n/a | stdlib | Retry-After, per-host `x/time/rate` limiter, slog logging, secret redaction, User-Agent | n/a |
| [hashicorp/go-retryablehttp](https://github.com/hashicorp/go-retryablehttp) | MPL-2.0 | v0.7.8, 2025-06-18 | 2023-04 on `master` (release tags newer) | imported by 2,571 | Pulls `go-hclog` and `fatih/color`. Has its own `Client` type, so callers stop holding a plain `*http.Client`. | yes |
| [imroc/req/v3](https://github.com/imroc/req) | MIT | v3.61.0, 2026-08-13 | 2026-09 | imported by 1,080 | Pulls quic-go, utls, brotli and digest. Its own client type and fingerprint impersonation. | yes |
| [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) | BSD-3 | v0.16.0, 2026-08-19 | 2026-08 | Go team | Token bucket for per-Indexer and per-provider limits | yes |

**Verdict: build**, plus **adopt** `x/time/rate`.

- The middleware is about 200–300 lines: Retry-After parsing, bounded retry on idempotent requests only, a limiter keyed by Indexer or provider, slog request logging with credential redaction (S30 "redaction"), and a timeout policy.
- It is the one place every client gets its `*http.Client`. The Fake world also needs a single seam for the `SSL_CERT_FILE` trust and the `net.Pipe` dialer.
- go-retryablehttp replaces `*http.Client` with its own type and brings hclog. req brings HTTP/3 and TLS impersonation nobody asked for.

**Criticality:** important. A wrong retry policy against a private tracker's site can get the account rate-banned.

## 2. Metadata: TMDB v3, TVDB v4, OMDb (S01-S09 TMDB, S19 OMDb, S21 TVDB)

**Current plan:** Unspecified. Fakes `fake-tmdb`, `fake-tvdb` and `fake-omdb` are decided (offline-boundaries rows 1–3).

| Candidate | License | Latest | Last commit | Signal | Injection | Verified |
|---|---|---|---|---|---|---|
| [cyruzin/golang-tmdb](https://github.com/cyruzin/golang-tmdb) | MIT | v1.9.4, 2026-07-15 | 2026-07 | imported by 63; active | **Fails.** `baseURL` is a package-level `var`, and `SetCustomBaseURL` mutates the global. Requests use `context.Background()` with no caller context. `SetClientAutoRetry` calls `time.Sleep` on 429. The `http.Client` is injectable (by value). | read source |
| [ryanbradynd05/go-tmdb](https://github.com/ryanbradynd05/go-tmdb) | MIT | none tagged; last commit 2023-01-08 | 2023-01 | imported by 28; stale | not examined further (stale) | yes |
| [dashotv/tvdb](https://github.com/dashotv/tvdb) (Speakeasy-generated, v4) | MIT | v0.5.2, 2024-07-08 | 2024-07 | imported by 2; README says "alpha, not ready for production" | unverified | yes (metadata) |
| pioz/tvdb, rbtr/go-tvdb | — | 2022 or older | — | target retired API v2/v3 | n/a | per search |
| OMDb Go clients | — | — | — | none worth naming. The API is one GET with `apikey`. | n/a | search |

**Verdict: build** all three, and **borrow** the response structs from golang-tmdb (MIT) and TVDB's [published OpenAPI spec](https://github.com/thetvdb/v4-api).

- golang-tmdb's global base URL fails the `Endpoints` rule: two Controllers in one test process would share it. Its `context.Background()` defeats cancellation.
- No maintained TVDB v4 Go client exists.
- The custom versions are about 800 (TMDB), 500 (TVDB, including token refresh on 401) and 150 (OMDb) lines, covering only the endpoints SlipStream reads. Risk is low, and the fakes test them at the wire.

**Criticality:** important. Metadata stops, but no data is lost.

## 3. Indexers: Cardigann runner, Torznab, RSS, scraping, cookies, challenge solver (S01-S09 Torznab; S26; S27)

**Current plan:**
- ADR 0010: GPL-3, the catalog is pulled at runtime, and no YAML is bundled.
- ADR 0013: a Cardigann runner, and the 53 Prowlarr C# indexers ported to Go.
- ADR 0017: one home Node per Indexer.
- Offline-boundaries rows 7–12: fakes for the catalog, sites, challenge solver, Torznab and RSS.

### 3a. Cardigann YAML runner

| Candidate | License | Latest | Last commit | Signal | Notes | Verified |
|---|---|---|---|---|---|---|
| [cardigann/cardigann](https://github.com/cardigann/cardigann) (lox's original Go) | MIT | v1.10.2, 2018-06-30 | 2020-04 | imported by 0; README asks for maintainers; abandoned | Predates Jackett's and Prowlarr's schema growth (v11 adds `selectorinputs`, `getselectorinputs`, `info_category_8000` and more). Usable only as a design reference. | yes |
| [Prowlarr/Prowlarr](https://github.com/Prowlarr/Prowlarr) Cardigann engine (C#) | GPL-3.0 | n/a | active | the reference implementation the catalog targets | Port its semantics (ADR 0010 already allows porting Prowlarr) | LICENSE read |
| [Jackett/Jackett](https://github.com/Jackett/Jackett) | **GPL-2.0, version 2 only** (README says "GNU GPL version 2", with no "or later") | n/a | active | — | **Do not port from Jackett.** GPL-2.0-only code cannot be combined into a GPL-3.0 binary. Port from Prowlarr only. | LICENSE + README read |
| [Prowlarr/Indexers](https://github.com/Prowlarr/Indexers) `definitions/v11/schema.json` | **no LICENSE file** (checked master: LICENSE, LICENSE.md, COPYING all 404) | — | active | JSON Schema draft 2019-09 with 22 block types | Read it as the spec and don't vendor it (same reasoning as ADR 0010) | yes |
| [sergiobonfiglio/tomagnet](https://pkg.go.dev/github.com/sergiobonfiglio/tomagnet) | MIT | v0.3.12, 2026-09-07 | 2026-09 | one author; runner sits in `internal/` (not importable); targets Jackett definitions | reference only | yes |
| v1 `internal/indexer/cardigann` | ours | — | — | 5,666 lines on goquery + text/template | Assumed suboptimal (brief). Worth reading for the gotchas list. | read |

**Probe against the live v11 catalog (579 definitions):**

- **YAML:** `gopkg.in/yaml.v3` **rejects 4 definitions** (arabtorrents, btarg, swarmazon-api, tormac). They use the YAML 1.2 `\/` escape in double-quoted strings. [goccy/go-yaml](https://github.com/goccy/go-yaml) (MIT, v1.19.2, 2026-01-08, imported by 2,983) parses all 4. → Use goccy/go-yaml, or yaml.v3 behind a pre-pass. The Cardigann loader must parse every live definition (boundary row 7).
- **CSS selectors** (HTML-response definitions only; 162 definitions answer in JSON or XML): **5,889 of 5,899 compile in cascadia.** The 10 failures are `:scope >` in torrentproject2 (5), `:has(~ …)` relative selectors in ebookbay, and one malformed selector in puntotorrent that AngleSharp evidently tolerates. → cascadia covers 99.8%. The runner needs a small `:scope` rewrite (row-relative selection) and has to accept a few dead definitions.
- **Filter regexes** (1,603 `regexp`/`re_replace` args): Go `regexp` fails 83. `regexp2` fails 25, all from one pattern repeated across definitions, `\p{IsCyrillic}`. That is a .NET named block regexp2 doesn't know, and rewriting `IsCyrillic`→`Cyrillic` fixes it. → Use regexp2 (already in the stack, ADR 0006) with a tiny .NET-block alias table.
- **Templates** (5,097 `{{…}}` strings): Go `text/template` fails 40. The causes are hyphenated config keys (`{{ .Config.cat-id }}`), `.2f`-style format args, and paren forms. Cardigann's template language is Jackett/Prowlarr's own regex-driven evaluator, not Go templates, even though it looks like them. → **Write a small evaluator** that ports Prowlarr's `ApplyGoTemplateText` semantics (`if/else/end`, `range`, `and`/`or`, `eq`/`ne`, `re_replace`, `join`, variables). Don't reuse `text/template`.

**Verdict: build** the runner (Prowlarr semantics), **borrow** from Prowlarr C# (GPL-3) and lox's Go layout (MIT). **Adopt** cascadia, via goquery or directly, and goccy/go-yaml.

- No maintained Go runner targets the v11 schema.
- Estimated size: about 4–6k lines, similar to v1. The risk is behavioural drift from Prowlarr. The boundary-7 test that "every real definition is parsed and builds its requests offline" is the guard, and the probe above is a cheap starting version of it.

**Criticality:** critical infrastructure. A bad login or retry loop against a private tracker risks the account (ratio or ban).

**Catalog version:**
- The Prowlarr/Indexers README now lists **v12**, which adds `base64decode`/`base64encode` filters, and `indexers.prowlarr.com/master/12` is live (HTTP 200).
- The boundary row pins `master/11`, which the README still marks ACTIVE with 522+ definitions.
- See the open questions.

### 3b. HTML scraping, XPath, JSON selection

| Candidate | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|
| [PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery) | BSD-3 | v1.13.0, 2026-08-27 | 2026-10 | imported by 13,845 | yes |
| [andybalholm/cascadia](https://github.com/andybalholm/cascadia) | BSD-2 | v1.3.5, 2026-09-04 | 2026-09 | imported by 758; goquery's engine | yes |
| [antchfx/htmlquery](https://github.com/antchfx/htmlquery) (XPath) | MIT | v1.3.7, 2026-10-05 | 2026-10 | imported by 987 | yes |
| [tidwall/gjson](https://github.com/tidwall/gjson) | MIT | v1.20.0, 2026-10-05 | 2026-10 | imported by 10,420 | yes |

**Verdict: adopt goquery** (CSS over `x/net/html`, matching how Prowlarr uses AngleSharp).

- **XPath (htmlquery) is not needed.** Cardigann selectors are CSS, and Prowlarr's XML responses are also queried with CSS. Leave it out unless a ported C# indexer needs XPath, which none should, since Prowlarr uses AngleSharp CSS.
- **JSON selection:** Cardigann JSON selectors are Newtonsoft `SelectToken` paths (`files[0].name`, `$.numFound`, `..large_cover_image`). gjson's syntax differs (`files.0.name`), so a 150-line path walker over `encoding/json` (v1 did this in `jsonpath.go`) beats a translation layer. **Build** that walker.

**Criticality:** important.

### 3c. Torznab and generic RSS (S01-S09, S26)

| Candidate | License | Latest | Last commit | Signal | Injection | Verified |
|---|---|---|---|---|---|---|
| [mmcdole/gofeed](https://github.com/mmcdole/gofeed) | MIT | v1.5.0, 2026-09-25 | 2026-09 | imported by 1,235 | `Parser.Client *http.Client` and `ParseURLWithContext`, or parse an `io.Reader` | read source |
| `encoding/xml` | BSD-3 | stdlib | — | — | n/a | — |

**Verdict:**
- **Build** Torznab on `encoding/xml`. It is about 200 lines: caps, `torznab:attr`, and error codes. The shape is fixed, and S01-S09 must own it end to end.
- **Adopt** gofeed's `Parse(io.Reader)` for generic RSS (S26), fetching through our own client. Real-world RSS is malformed often enough that gofeed's lenient parser (goxpp) earns its place.

**Criticality:** important.

### 3d. Cookies and logins

- `net/http/cookiejar` can't be serialized.
- ADR 0003 says Workers hold no durable state, and ADR 0017 says an Indexer searches from one home Node. So the session cookies need a Controller-owned store that is shipped to the home Node and survives failover.
- **Build** a small `http.CookieJar` backed by a typed cookie list (about 150 lines). Use `golang.org/x/net/publicsuffix` (BSD-3) for the domain rules.
- For S27's 2FA/TOTP auth family, [pquerna/otp](https://github.com/pquerna/otp) (Apache-2.0, v1.5.0, 2024-12-31, imported by 822) works. RFC 6238 is also about 40 lines of `crypto/hmac`. **Build**; the dependency adds nothing.

### 3e. Cloudflare: challenge solver and TLS fingerprinting (S26)

| Candidate | License | Latest | Last commit | Signal | Notes | Verified |
|---|---|---|---|---|---|---|
| [FlareSolverr](https://github.com/FlareSolverr/FlareSolverr) (external service, protocol only) | MIT | unverified | unverified | 2026 secondary sources say it lags Cloudflare changes | `POST /v1` JSON; we write the client | LICENSE read |
| [Byparr](https://github.com/ThePhaseless/Byparr) (drop-in FlareSolverr API) | GPL-3.0 | unverified | unverified | — | Same protocol, so our client works unchanged | LICENSE read |
| [refraction-networking/utls](https://github.com/refraction-networking/utls) | BSD-3 | v1.8.2, 2026-01-12 | 2026-10 | imported by 1,206 | ClientHello mimicry. HTTP/2 needs a custom transport. | yes |
| [bogdanfinn/tls-client](https://github.com/bogdanfinn/tls-client) | **BSD-4-Clause** | v1.16.0, 2026-09-02 | 2026-09 | imported by 201 | **GPL-incompatible.** The advertising clause is at LICENSE lines 11–12. It also forks net/http (`fhttp`), utls and quic-go. | LICENSE read |
| [Danny-Dasilva/CycleTLS](https://github.com/Danny-Dasilva/CycleTLS) | GPL-3.0 | v1.0.30, 2025-10-02 | 2026-04 | imported by 111 | Compatible, but a heavy forked stack | yes |
| [Noooste/azuretls-client](https://github.com/Noooste/azuretls-client) | MIT | v1.13.2, 2026-04-17 | 2026-04 | imported by 35 | Small community | yes |

**Verdict:**
- **Build** the FlareSolverr-protocol client. It is about 120 lines: `request.get`/`request.post`, then reuse the returned cookies and User-Agent on the session. Boundary row 9 already decides "anything speaking FlareSolverr's API", and any compatible solver (FlareSolverr, Byparr) works through the same client.
- **Don't adopt a TLS-fingerprint library now.** Prowlarr itself uses plain .NET `HttpClient`, so parity doesn't need it.
- If one is wanted later, use utls (BSD-3) behind the shared transport.
- **Reject bogdanfinn/tls-client on license (BSD-4).** The CI allow-list (ADR 0010) should list BSD-4-Clause as denied.

**Criticality:** important. Without a solver, Cloudflare-fronted sites fail, but nothing is lost.

### 3f. Ported C# indexers (S27)

There is no library question here: ADR 0013 decides a port of Prowlarr's GPL-3 C#, about 99 files. Two notes:

- Ports must come from Prowlarr, never Jackett (GPL-2.0-only, see 3a).
- Every port should get its `*http.Client` and base URL from the shared plumbing in §1 and must not call `http.DefaultClient`. A lint, alongside the `Endpoints` lint, catches regressions.

---

## 4. External Download Clients (S30)

**Current plan:** Offline-boundaries row 16 fixes the wire for each client:
- qBittorrent: WebUI API v2 with a cookie SID.
- Transmission: RPC with the 409 session-id.
- **Deluge: Web JSON-RPC `/json`.**
- rTorrent: XML-RPC `/RPC2`.

The library choice is unspecified.

| Candidate | License | Latest | Last commit | Signal | Fit | Verified |
|---|---|---|---|---|---|---|
| [autobrr/go-qbittorrent](https://github.com/autobrr/go-qbittorrent) | MIT | v1.19.0, 2026-09-19 | 2026-09 | imported by 15; used by autobrr and qui | `WithHTTPClient` and Host config work. About 9.7k lines, of which SlipStream needs roughly 8 calls. stdlib `log.Logger`; pulls `avast/retry-go` v3 and `pkg/errors`. Has built-in retry (5 attempts by default). | read source |
| [hekmon/transmissionrpc/v3](https://github.com/hekmon/transmissionrpc) | MIT | v3.0.0, 2023-10-30 | 2023-10 | imported by 13; **no commits since 2023** | `Config.CustomClient` and endpoint URL work, with ctx on every call. But it speaks only the **old bespoke RPC (rpc_version 17).** Transmission 4.1.0 moved to JSON-RPC 2.0 and snake_case, and the old protocol is "deprecated and will be removed" ([rpc-spec.md](https://github.com/transmission/transmission/blob/main/docs/rpc-spec.md) lines 2–4). | read source + spec |
| [autobrr/go-deluge](https://github.com/autobrr/go-deluge) | GPL-2.0-**or-later** (file headers) | v1.4.0, 2025-05-03 | 2026-07 | imported by 3 | **Wrong protocol.** It speaks the daemon's rencode RPC on port 58846, not Web JSON-RPC `/json`. Depends on `gdm85/go-rencode`, which is GPL-2.0 with unverified "or later" wording. | read source |
| [autobrr/go-rtorrent](https://github.com/autobrr/go-rtorrent) | MIT | v1.13.0, 2026-10-07 | 2026-10 | imported by 2 | `WithCustomClient` and `NewClientWithHTTPClient` work; 1.6k lines. Its go.mod pulls testcontainers, but only for tests. | read source |
| [mrobinsn/go-rtorrent](https://github.com/mrobinsn/go-rtorrent) | MIT | v1.8.0, 2022-01-17 | 2025-01 | imported by 0 | superseded by the autobrr fork | yes |
| [kolo/xmlrpc](https://github.com/kolo/xmlrpc) | MIT | 2022-09-21 pseudo-version | 2022-09 | imported by 277 | stale | yes |

**Verdict: build** four thin clients over a shared interface, and **borrow** autobrr's qBittorrent and rTorrent code (MIT) for field names and version quirks.

- SlipStream needs about 8 operations per client: add by magnet or file with category, save path and paused state; list by category or label; get files; remove with or without data; set category; pause and resume; read version and free space; test.
- v1 managed each in 500–700 lines.
- The wrappers either speak a dead or wrong protocol (Transmission, Deluge) or bring 10× the surface plus their own retry policy (qBittorrent), which fights §1.
- A thin Transmission client can speak JSON-RPC 2.0 and fall back to the legacy protocol, based on `X-Transmission-Rpc-Version` in the 409 response. That decision belongs in S30.

**Criticality:** critical infrastructure. Remove-with-data or wrong-category bugs delete user data or stop seeding (ratio loss).

---

## 5. Media Servers and plex.tv (S35, S43)

**Current plan:**
- ADR 0009: sessions are watched, and SlipStream advises only.
- Offline-boundaries rows 17–20 and 45 list a deliberately small set of calls: refresh by path, item refresh, `Media[]` versions, sessions and transcode sessions, `/myplex/account`, websockets, PIN and JWT, `shared_servers`, invites, and Jellyfin/Emby users and API keys.

| Candidate | License | Latest | Last commit | Signal | Fake-friendliness | Verified |
|---|---|---|---|---|---|---|
| [LukeHagar/plexgo](https://github.com/LukeHagar/plexgo) (Speakeasy-generated) | MIT | v0.29.2, 2026-08-21 | 2026-08 | imported by 3 | `WithServerURL` and `WithClient(HTTPClient)` work for PMS. **plex.tv operations default to hardcoded `https://plex.tv/api/v2`** and are overridden only per call (`operations.WithServerURL`), so every call site must remember it. About **243k generated lines**. No websocket support. | read source |
| [jrudio/go-plex-client](https://github.com/jrudio/go-plex-client) | Apache-2.0 | untagged; 2025-01-27 | 2025-01 | imported by 30 | unverified | yes |
| [sj14/jellyfin-go](https://github.com/sj14/jellyfin-go) (openapi-generator) | MIT | v0.5.0, 2026-09-09 | 2026-10 | small; built for the author's `jellyctl` | `Configuration.Servers` and `HTTPClient` work. **About 183k generated lines.** | read source |
| Jellyfin OpenAPI + [ogen](https://github.com/ogen-go/ogen) or [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) on a filtered spec | Apache-2.0 (generators) | ogen v1.24.0, 2026-08-07; oapi-codegen v2.8.0, 2026-07-17 | active | — | Injectable, but needs a spec-pruning step | yes (metadata) |
| Emby | — | — | — | no Go client found | n/a | search |
| [coder/websocket](https://github.com/coder/websocket) (for Plex `/:/websockets/notifications`, Jellyfin `/socket`, Emby `/embywebsocket`) | ISC | v1.8.15, 2026-06-15 | 2026-06 | imported by 1,844 | `DialOptions.HTTPClient` is injectable, and it takes a context | yes |
| [gorilla/websocket](https://github.com/gorilla/websocket) | BSD-2 | v1.5.3, 2024-06-14 | 2022-12 on `master` | imported by 48,574 | works, but has slow upkeep | yes |

**Verdict:**
- **Build** the Plex, plex.tv, Jellyfin and Emby clients by hand. Jellyfin and Emby share a dialect (Jellyfin forked Emby), so one client with per-server differences covers both.
- **Borrow** request and response shapes from plexgo's OpenAPI spec (MIT) and Jellyfin's published OpenAPI.
- **Adopt coder/websocket** for the three notification sockets.
- The generated SDKs are fake-*capable*, but plexgo's per-call plex.tv override is exactly the trap ADR 0014 warns about: one missed call reaches the internet unnoticed. Hundreds of thousands of generated lines for about 25 calls is the opposite of the maintainer's cleanliness goal.
- Plex PIN auth is two calls (create PIN, poll), and the JWT rotation is a few more. Estimated size: Plex plus plex.tv about 1,200 lines, Jellyfin/Emby about 800.

**Criticality:** important. A wrong path mapping or refresh doesn't lose data. A bad `shared_servers` call (S43) could revoke a member's access.

---

## 6. Notifications (S37)

**Current plan:**
- Offline-boundaries row 23: `notify-sink` records every request, and "each notifier keeps a golden payload test against what the sink captured".
- Kept providers: Discord, Slack, Telegram, Pushover, Gotify, ntfy, Pushbullet, Apprise API, and generic Webhook. SMTP is separate (S12, go-smtp fake).

| Candidate | License | Latest | Last commit | Signal | Fit | Verified |
|---|---|---|---|---|---|---|
| [containrrr/shoutrrr](https://github.com/containrrr/shoutrrr) | MIT | v0.8.0, 2023-08-20 | stale | imported by 95 | superseded by the fork | yes |
| [nicholas-fedor/shoutrrr](https://github.com/nicholas-fedor/shoutrrr) (maintained fork) | MIT | v0.21.2, 2026-10-06 | 2026-10-11 | imported by 7; one maintainer | Each service has `SetHTTPClient` and `SetDialContext`. **Hosts are hardcoded** for Discord (`discord.com/api/webhooks/`), Slack (`hooks.slack.com`, `slack.com/api`), Telegram (`api.telegram.org`), Pushover and Pushbullet. A fake works only through a host-rewriting Transport. Config is URL strings (`discord://token@id`). The `router` package links all 36 services, including MQTT (paho), XMPP (mellium) and oauth2. About 36k lines. | read source |
| [nikoksr/notify](https://github.com/nikoksr/notify) | MIT | v1.6.0, 2026-08-31 | 2026-10 | imported by 71 | wraps per-vendor SDKs (discordgo, slack-go…), so injection varies by vendor | metadata only |
| [Apprise API](https://github.com/caronc/apprise-api) (external service) | MIT (apprise core BSD-2) | unverified | — | — | One `POST /notify` with `urls`. Our client is about 60 lines. | LICENSE read |
| Vendor SDKs: [gregdel/pushover](https://github.com/gregdel/pushover) (MIT, v1.4.0, 2025-09-22), [bwmarrin/discordgo](https://github.com/bwmarrin/discordgo) (BSD-3, v0.29.0, 2025-05-24), [slack-go/slack](https://github.com/slack-go/slack) (BSD-2, v0.30.1, 2026-10-04), [go-telegram-bot-api v5](https://github.com/go-telegram-bot-api/telegram-bot-api) (MIT, v5.5.1, 2021-12-13, stale) | — | — | — | — | Each is far larger than a webhook POST. discordgo is a gateway bot library. | yes |

**Verdict: build** one small client per provider, about 80–250 lines each, around 1,200 lines total. **Borrow** payload shapes and error semantics from shoutrrr (MIT).

- Every provider in the list is a single HTTPS POST of JSON or form data with a token.
- The golden-payload tests in row 23 make the exact payload SlipStream's own contract, which a library that formats Discord embeds or Slack blocks its own way works against.
- shoutrrr's hardcoded hosts would force a host-rewriting transport into the shipped binary just for tests. That is the "defensive plumbing" the maintainer wants to avoid.

**Criticality:** convenience.

---

## 7. Subtitle Providers (S46)

**Current plan:** ADR 0008 decides native sync. Offline-boundaries row 25 gives one fake per provider. Clients are unspecified.

| Candidate | License | Latest | Signal | Notes | Verified |
|---|---|---|---|---|---|
| Go clients for SubDL, SubSource, Gestdown | — | — | none found (one CLI, `kakeetopius/subg`, supports SubDL inside an app) | — | search |
| [Bazarr](https://github.com/morpheus65535/bazarr) `custom_libs/subliminal_patch/providers/{subdl,subsource,gestdown}.py` | GPL-3.0 | n/a | active | subdl.py 1,278 lines (`api.subdl.com/api/v1/`, `api_key` in query, a quota/back-off policy), subsource.py 650 lines, gestdown.py 196 lines | LICENSE + files read |

**Verdict: build**, and **borrow** request and error handling from Bazarr's providers (GPL-3, so porting is allowed).

- Each client is about 150–400 lines in Go.
- Bazarr's code is the best available record of each API's undocumented quotas and failure shapes, which the fakes must also play (row 25).

**Criticality:** convenience.

## 8. TRaSH Guides (S53)

**Current plan:**
- ADR 0006: formats run verbatim, are tracked per field by `trash_id`, and run on regexp2.
- Offline-boundaries row 26: pinned snapshots on disk.
- Releases bundle a snapshot (MIT, notice kept).

| Candidate | License | Language | Notes | Verified |
|---|---|---|---|---|
| [TRaSH-Guides/Guides](https://github.com/TRaSH-Guides/Guides) data | MIT | JSON | `metadata.json` lists `json_paths` per app (custom_formats, qualities, naming, quality_profiles, cf-groups, quality-profile-groups, conflicts). JSON Schemas live under `schemas/` (e.g. `radarr-cf.schema.json`, HTTP 200). | read |
| [Recyclarr](https://github.com/recyclarr/recyclarr) | MIT | C# | Reference for merge and sync semantics. ADR 0006 deliberately departs from its overwrite model. | LICENSE read |
| [configarr](https://github.com/raydak-labs/configarr) | **AGPL-3.0** | TS | **Don't port code.** AGPL terms would attach. Reading for behaviour is fine. | LICENSE read |
| [Notifiarr](https://github.com/Notifiarr/notifiarr) | MIT | Go | v0.9.7, 2026-08-30. A client for the Notifiarr website, not a TRaSH JSON consumer as far as checked (unverified in depth). | metadata |

**Verdict: build.**

- The consumer is a typed loader over the JSON files listed in `metadata.json`, plus the per-field three-way diff that ADR 0006 requires and no existing tool implements.
- Fetch a commit's tarball through one `Endpoints` URL (codeload or the GitHub archive), not a git client. That keeps the boundary to one HTTPS GET and keeps go-git out.
- Validate the loader against TRaSH's own JSON Schemas in tests.

**Criticality:** important. A wrong merge silently destroys an admin's score tuning, which is the exact harm ADR 0006 exists to prevent.

## 9. Radarr and Sonarr import (S55)

**Current plan:** Offline-boundaries rows 27–28 decide both an API path (`fake-radarr`/`fake-sonarr`) and a DB-file path.

| Candidate | License | Latest | Last commit | Signal | Fit | Verified |
|---|---|---|---|---|---|---|
| [golift/starr](https://github.com/golift/starr) (`golift.io/starr`) | MIT | v1.4.1, 2026-09-07 | default-branch pseudo-version reads 2022-12; release tags are recent | imported by 30; used by Notifiarr and Unpackerr | `Config{Client *http.Client, URL, APIKey}`, with context variants on calls. Single runtime dependency `x/net`. About 26k lines across radarr, sonarr, lidarr, readarr and prowlarr; only the radarr and sonarr packages would be linked. | read source |
| Radarr/Radarr, Sonarr/Sonarr schemas | GPL-3.0 | — | — | — | the reference for DB table shapes | LICENSE read |
| modernc SQLite (already decided, ADR 0013) | BSD-3 | — | — | — | Open the user's DB file read-only. Copying db+wal first, or `VACUUM INTO` a temp file, avoids touching a live WAL. | — |

**Verdict:**
- **Wrap golift/starr** behind the Import engine's reader interface for the API path. It meets every injection rule, is MIT, small in dependencies, and maintained, and the Radarr/Sonarr v3 type surface is large and tedious to retype.
- **Build** the DB-file reader as plain SQL on modernc. It is read-only, with a schema-version check against the app's `VersionInfo`/migration table.
- The API path also covers users whose Radarr runs on Postgres, which the DB path can't reach.

**Criticality:** important. It is read-only on the source side, but a wrong path translation (Windows `C:\` and `\\nas\`) imports files to the wrong place.

---

## ADR conflicts and license flags

- **No library recommended here reopens an ADR.** Everything adopted is MIT, BSD-2/3, ISC or Apache-2.0: goquery, cascadia, goccy/go-yaml, gofeed, x/time, x/net/publicsuffix, coder/websocket and golift/starr.
- **License flags for the ADR 0010 allow-list:**
  - bogdanfinn/tls-client is **BSD-4-Clause** (GPL-incompatible).
  - Jackett is **GPL-2.0-only**: no porting.
  - configarr is **AGPL-3.0**: no porting.
  - Prowlarr/Indexers has **no license**: read the schema, never vendor it (consistent with ADR 0010).
  - autobrr/go-deluge is GPL-2.0-or-later (compatible), but its `gdm85/go-rencode` dependency says only "GPL v2" (unverified "or later"). Moot if Deluge goes through Web JSON-RPC as row 16 says.
- **Offline-boundaries row 16 vs reality:** Transmission's 409 session-id dance survives in JSON-RPC 2.0, but the method and field names changed in 4.1.0. Row 16 should say which protocol the fake speaks.
- **Offline-boundaries row 7:** it pins `master/11`. v12 is published.

## Open questions for the maintainer

1. **Cardigann catalog version:** target v11 only (as row 7 says), or v11 plus v12 from the start? *Recommend v11 plus v12.* v12 adds only `base64decode`/`base64encode` on top of v11, and supporting it now avoids a forced migration when Prowlarr flips the active version.
2. **Cardigann templates:** port Prowlarr's own template evaluator, or adapt Go `text/template` with a pre-pass? *Recommend porting the evaluator.* `text/template` rejects 40 live strings (hyphenated keys, printf formats, paren forms), and a pre-pass would be a second dialect to maintain.
3. **YAML parser for definitions:** goccy/go-yaml, or yaml.v3 with an escape pre-pass? *Recommend goccy/go-yaml.* yaml.v3 rejects 4 live definitions over the YAML 1.2 `\/` escape.
4. **Transmission:** speak JSON-RPC 2.0 only (Transmission ≥ 4.1.0), or both protocols, chosen by the 409's `X-Transmission-Rpc-Version` header? *Recommend both.* 4.0.x is still widely deployed, and the legacy protocol is deprecated but not removed.
5. **Notifications:** hand-write each provider (about 1,200 lines total), or adopt nicholas-fedor/shoutrrr with a host-rewriting transport for the fakes? *Recommend hand-writing.* The golden-payload rule makes the payload ours anyway.
6. **Media servers:** hand-write the Plex, plex.tv and Jellyfin/Emby clients, or generate them from OpenAPI (plexgo or a pruned Jellyfin spec)? *Recommend hand-writing.* About 25 calls total, and plexgo's per-call plex.tv override is an `Endpoints` leak waiting to happen.
7. **Radarr/Sonarr API:** wrap golift/starr, or hand-write the v3 subset? *Recommend wrapping starr.* It meets every injection rule and the type surface is large.
8. **TLS fingerprinting:** leave it out of S26 and rely on the FlareSolverr-protocol solver only? *Recommend yes.* Prowlarr doesn't fingerprint. If it is ever added, use utls (BSD-3) and never bogdanfinn/tls-client (BSD-4).
9. **Retry and rate limiting:** one in-repo RoundTripper chain shared by every client, with no library retries enabled anywhere? *Recommend yes.* It is the only way the fault vocabulary's 429/Retry-After behaviour is consistent and testable.
10. **License CI:** add BSD-4-Clause, AGPL-3.0, GPL-2.0-only and "no license" to the deny list explicitly, and have the S01-S09 skeleton check that no file is ported from Jackett? *Recommend yes for the deny list.* The Jackett rule belongs in AGENTS.md, since CI can't detect ported code.
