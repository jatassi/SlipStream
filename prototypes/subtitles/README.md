# Subtitles prototype

Throwaway prototype for [Subtitle Provider setup and subtitle screens](https://github.com/jatassi/SlipStream/issues/69). It never merges. It sits on top of `prototype/ui-direction` and reuses Lit console's stylesheet, so every screen looks like the settled direction.

## Run it

```bash
prototypes/subtitles/run.sh
```

Then open http://localhost:4175/subtitles/?variant=A. The yellow bar switches the variant, the screen, the scenario and the device. Press `←` and `→` to cycle variants and `\` to hide the bar.

- **Screens.** Providers (Settings › Subtitles), Profile (Wanted Languages on Cinema), Movie file (Dune: Part Two, two library files) and Episodes (Severance season 2).
- **Scenarios.** Fresh install (no keys, Gestdown only), Keys added (all ready) and Quota out, key revoked (SubDL paused until its reset, SubSource's key rejected).

Artwork is copied from v1's local cache at run time and is never committed.

## Provider facts used

Looked up on 2026-10-11 for the cards. The key checks and rejection replies were captured live with a dummy key.

| Provider | Key page and steps | Bad-key reply | Allowance |
| -------- | ------------------ | ------------- | --------- |
| SubDL | Free account, then the API page in your panel, `subdl.com/panel/api`. Generate a key. | `403 {"error":"Not Authorized"}` from `GET api.subdl.com/api/v1/me` and from search | Free key. The developer page says 2,000 searches and 50 downloads a day. The older API doc says 2,000 requests a day. Bazarr resets it at midnight GMT. |
| SubSource | Free account, then "My Profile" on the account dashboard. Generate a key, which can be regenerated any time. | `401 {"error":"Invalid API key","message":"The provided API key is invalid or inactive"}` | 60 requests a minute, 1,800 an hour, 7,200 a day per key. Rate-limit headers come on every response. |
| Gestdown | No key. It proxies Addic7ed for TV. `GET api.gestdown.info/shows/search/Severance` answered without credentials. | None | None published. |

Sources: [SubSource API docs](https://subsource.net/api-docs), [SubDL developers](https://subdl.com/developers), [SubDL API doc](https://subdl.com/api-doc), Bazarr's [SubDL](https://github.com/morpheus65535/bazarr/blob/master/custom_libs/subliminal_patch/providers/subdl.py) and [SubSource](https://github.com/morpheus65535/bazarr/blob/master/custom_libs/subliminal_patch/providers/subsource.py) providers. subdl.com sits behind a Cloudflare challenge, so its pages were read through search results, not fetched directly. The panel path is inferred from a third-party README and SubDL's own "panel → API" wording.

## Files

- `index.html` holds the switcher and loads one variant.
- `data.js` is the fixture: providers, scenarios, the state table, Profiles, the two Works and the manual search candidates.
- `shell.js` is Lit console's admin chrome, lifted from `prototypes/ui-direction`.
- `variants/a-cards`, `variants/b-ranked` and `variants/c-coverage` are the three variants.
