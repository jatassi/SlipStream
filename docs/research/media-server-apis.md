# Plex, Jellyfin and Emby APIs

Research for wayfinder ticket #21. It feeds Q16 of the charting decisions (#2): Plex, Jellyfin and Emby are first-class behind one Media Server contract.

## Question

What do Plex, Jellyfin and Emby expose for library refresh, path mapping, sessions and player capabilities, auth, and remote access status?

## Versions and sources

Each claim cites its source inline. Claims marked **inferred** are reasoning from the cited material, not something the source states.

| Media Server | Version read | Primary source |
| --- | --- | --- |
| Plex Media Server | PMS API 1.2.3 (PMS 1.43.4 or later) | Official API reference at [developer.plex.tv/pms](https://developer.plex.tv/pms/). It is a Redoc page with the OpenAPI 3.1 spec embedded. PMS itself is closed source. |
| Jellyfin | 12.1 (released 2026-09-15), API 12.1.0 | Source at tag [`v12.1`](https://github.com/jellyfin/jellyfin/tree/v12.1) and the generated spec at [api.jellyfin.org](https://api.jellyfin.org/openapi/jellyfin-openapi-stable.json). |
| Emby Server | 4.10.1.0 (stable, 2026-09-29) | Official SDK repo [MediaBrowser/Emby.SDK](https://github.com/MediaBrowser/Emby.SDK/tree/1faf8176df7ede9152c38f27a5925e6ccaf5e12b): the OpenAPI spec at `Resources/OpenApi/openapi_v3.json` and the REST docs under `Documentation/doc/restapi/`. Emby Server is closed source, so behavior beyond the spec and docs is unverified. |

Links below shorten the SDK path as `Emby.SDK@1faf817`.

Jellyfin and Emby share an ancestor (Jellyfin forked Emby 3.5). Many endpoints have the same path and body in both. The differences that matter are called out below.

## Summary

| Area | Plex | Jellyfin | Emby |
| --- | --- | --- | --- |
| Whole-library refresh | `POST /library/sections/refresh` (all) or `POST /library/sections/{id}/refresh` (one section) | `POST /Library/Refresh`, admin only | `POST /Library/Refresh`, admin only |
| Path-scoped refresh | `POST /library/sections/{id}/refresh?path=...` | `POST /Library/Media/Updated` with a list of paths, any authenticated user, debounced 60 s by default | `POST /Library/Media/Updated` with a list of paths, any user |
| Library folder paths | `Location[].path` on each section | `GET /Library/VirtualFolders` `Locations[]`, admin only | `GET /Library/VirtualFolders/Query` `Locations[]`, any user |
| Server-side path rewriting | None documented | `PathSubstitutions`, applied only to paths the server hands out | `PathSubstitutions` field exists; behavior unverified |
| Sessions list | `GET /status/sessions` | `GET /Sessions` | `GET /Sessions` |
| Per-session play decision | `Part.decision`, `Stream.decision`, `TranscodeSession` with hardware flags | `PlayState.PlayMethod`, `TranscodingInfo` with `TranscodeReasons` and `HardwareAccelerationType` | `PlayState.PlayMethod`, `TranscodingInfo` with `TranscodeReasons` and encoder and decoder hardware fields |
| Client device profile visible to a third party | No. Profiles are built in per client product, plus per-request augmentations. | Yes. `Capabilities.DeviceProfile` on each session and device. | No. Session and device DTOs omit the profile. |
| Stop a session | `POST /status/sessions/terminate` with a reason shown to the user | Playstate `Stop` command, plus a separate message command | Playstate `Stop` command, plus a separate message command |
| Push events | WebSocket or EventSource at `/:/websocket/notifications` and `/:/eventsource/notifications` | WebSocket on any path, typed `SessionMessageType` messages | WebSocket, documented message types |
| Auth for a service | plex.tv account token via PIN flow, then a per-server `accessToken`. New JWT flow with 7-day tokens. | API key (admin creates), or user token. Default header is `Authorization: MediaBrowser Token="..."`. | API key, sent as `X-Emby-Token` or `api_key` |
| Remote access status | `myPlexMappingState` and `myPlexSigninState` on `GET /`. plex.tv lists connections, including relay. | Config flags only. No reachability report. | `WanAddress` and `RemoteAddresses` on `/System/Info`, plus config flags. Emby Connect lists a remote URL. |

## 1. Library refresh

### Plex

- `POST /library/sections/{sectionId}/refresh` starts a section refresh. It takes `force` and `path`, and `path` is documented as "Restrict refresh to the specified path" ([librarySectionPostRefresh](https://developer.plex.tv/pms/#tag/Library/operation/librarySectionPostRefresh)).
- `POST /library/sections/refresh` refreshes every section. It returns 503 when it "cannot refresh a music library when not signed in" ([libraryPostSectionsRefresh](https://developer.plex.tv/pms/#tag/Library/operation/libraryPostSectionsRefresh)).
- `DELETE /library/sections/{sectionId}/refresh` cancels a section refresh, and `DELETE /library/sections/all/refresh` stops all refreshes (same spec, operations `librarySectionDeleteRefresh` and `libraryDeleteSectionsAllRefresh`).
- `PUT /library/metadata/{ids}/refresh` refreshes one item's metadata from its agent ([libraryMetadataPutRefresh](https://developer.plex.tv/pms/#tag/Library/operation/libraryMetadataPutRefresh)).
- Scan progress has three sources. Each section has a `refreshing` boolean, "whether this library section is currently scanning" (`librarySection` schema on [libraryGetSections](https://developer.plex.tv/pms/#tag/Library/operation/libraryGetSections)). `GET /activities` lists running activities with `type`, `progress` and `cancellable` ([activitiesGetSlash](https://developer.plex.tv/pms/#tag/Activities/operation/activitiesGetSlash)). The notification stream pushes events ([websocketGetSlash](https://developer.plex.tv/pms/#tag/Events/operation/websocketGetSlash)). The spec does not list the event type names.
- The spec documents refresh as `POST`. Whether `GET` still works for older clients is not stated.

### Jellyfin

- `POST /Library/Refresh` runs `ValidateMediaLibrary` over the whole library. It requires the `RequiresElevation` policy, which means an admin ([LibraryController.cs L337-L345](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Api/Controllers/LibraryController.cs#L337-L345)).
- `POST /Library/Media/Updated` takes `{"Updates":[{"Path":..., "UpdateType":...}]}`. It needs only `[Authorize]`, so any authenticated user or API key can call it. The handler calls `_libraryMonitor.ReportFileSystemChanged(path)` for each entry and ignores `UpdateType` ([LibraryController.cs L648-L659](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Api/Controllers/LibraryController.cs#L648-L659)).
- `ReportFileSystemChanged` drops ignored paths, then creates or merges a `FileRefresher`. Siblings are rebased onto their common parent, and children merge into a parent refresher ([LibraryMonitor.cs L374-L459](https://github.com/jellyfin/jellyfin/blob/v12.1/Emby.Server.Implementations/IO/LibraryMonitor.cs#L374-L459)). The refresher waits `LibraryMonitorDelay` seconds before it scans, and the default is 60 ([ServerConfiguration.cs L172](https://github.com/jellyfin/jellyfin/blob/v12.1/MediaBrowser.Model/Configuration/ServerConfiguration.cs#L172), [FileRefresher.cs L79-L83](https://github.com/jellyfin/jellyfin/blob/v12.1/Emby.Server.Implementations/IO/FileRefresher.cs#L79-L83)).
- `POST /Library/Movies/Updated?tmdbId=|imdbId=` and `POST /Library/Series/Updated?tvdbId=` look up items by provider ID and report each item's existing path. A new item with no row yet matches nothing ([LibraryController.cs L577-L640](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Api/Controllers/LibraryController.cs#L577-L640)). **Inferred:** these endpoints only refresh items Jellyfin already knows, so a path-based report is the one that picks up new imports.
- `POST /Items/{itemId}/Refresh` queues a metadata refresh for one item, with refresh-mode and replace flags ([OpenAPI `RefreshItem`](https://api.jellyfin.org/openapi/jellyfin-openapi-stable.json)).
- Scan progress shows up in two places. `VirtualFolderInfo` carries `RefreshProgress` and `RefreshStatus` (OpenAPI `GetVirtualFolders`). The WebSocket sends `RefreshProgress`, `LibraryChanged`, `ScheduledTasksInfo` and `ScheduledTaskEnded` messages (OpenAPI `SessionMessageType` enum).

### Emby

- `POST /Library/Refresh` "Requires authentication as administrator" ([openapi_v3.json `postLibraryRefresh`, Emby.SDK@1faf817](https://github.com/MediaBrowser/Emby.SDK/blob/1faf8176df7ede9152c38f27a5925e6ccaf5e12b/Resources/OpenApi/openapi_v3.json)).
- `POST /Library/Media/Updated` takes the same `Updates[{Path, UpdateType}]` body and "Requires authentication as user" (same spec, `postLibraryMediaUpdated`). The spec marks `/Library/Movies/*` and `/Library/Series/*` as "Deprecated. Use /Library/Media/Updated".
- `ServerConfiguration.LibraryMonitorDelaySeconds` is "the delay in seconds that we will wait after a file system change" (same spec). **Inferred:** Media/Updated is debounced the same way as in Jellyfin. The Emby source is closed, so this is unverified.
- `POST /Items/{Id}/Refresh` adds a `Recursive` flag (same spec, `postItemsByIdRefresh`).
- `VirtualFolderInfo` carries `RefreshProgress` and `RefreshStatus` (same spec).

## 2. Path mapping

None of the three Media Servers rewrites an inbound path that a caller reports. Each one expects the path as the server itself sees it on disk. Under v2 topology, a file lands on a storage Node, and the Media Server may mount it under a different path. **Inferred:** SlipStream must translate its own path to the server's path before it calls refresh. v1 already did this for Plex only ([v1 backend inventory §10.3](https://github.com/jatassi/SlipStream/blob/research/v1-inventory/docs/research/v1-inventory/backend.md)).

What each server exposes to support that translation:

- **Plex.** Each library section has `Location[]` with `id` and `path`, "a top-level location on disk where media in this library section is stored" (`librarySection` schema, [libraryGetSections](https://developer.plex.tv/pms/#tag/Library/operation/libraryGetSections)). `GET /library/sections/{id}/location` also lists folder locations ([librarySectionGetLocations](https://developer.plex.tv/pms/#tag/Content/operation/librarySectionGetLocations)). The API reference documents no server-side path substitution.
- **Jellyfin.** `GET /Library/VirtualFolders` returns `Name`, `CollectionType`, `ItemId` and `Locations[]` per library. The controller requires `FirstTimeSetupOrElevated`, which means an admin after setup ([LibraryStructureController.cs L31-L32](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Api/Controllers/LibraryStructureController.cs#L31-L32)). `GET /Library/PhysicalPaths` flattens all locations (OpenAPI `GetPhysicalPaths`). `ServerConfiguration.PathSubstitutions` is a `From`/`To` list ([ServerConfiguration.cs L221](https://github.com/jellyfin/jellyfin/blob/v12.1/MediaBrowser.Model/Configuration/ServerConfiguration.cs#L221)). It is applied only in `GetPathAfterNetworkSubstitution`, which serves outbound DTO and NFO paths for clients that open files directly ([LibraryManager.cs L3584-L3595](https://github.com/jellyfin/jellyfin/blob/v12.1/Emby.Server.Implementations/Library/LibraryManager.cs#L3584-L3595), [DtoService.cs L1833](https://github.com/jellyfin/jellyfin/blob/v12.1/Emby.Server.Implementations/Dto/DtoService.cs#L1833)). It does not rewrite inbound `Library/Media/Updated` paths.
- **Emby.** `GET /Library/VirtualFolders/Query` returns `QueryResult<VirtualFolderInfo>` with `Locations[]` and "Requires authentication as user". `GET /Library/PhysicalPaths` also exists. `ServerConfiguration` has `PathSubstitutions` and `MetadataNetworkPath` (all from the same spec). **Inferred:** Emby uses `PathSubstitutions` for the same outbound purpose as Jellyfin, since the two share an ancestor. This is unverified.

All three report a library's paths, so the server-side root of each library can be discovered. **Inferred:** a mapping UI could pre-fill the server side from these lists, and a check could flag any SlipStream root that maps outside every library location.

## 3. Sessions and player capabilities

### Plex

- `GET /status/sessions` lists current playbacks ([statusGetSlash](https://developer.plex.tv/pms/#tag/Status/operation/statusGetSlash)). The documented example has the following per session:
  - `User` with `id` and `title`.
  - `Player` with `product`, `platform`, `platformVersion`, `address`, `remotePublicAddress`, `state`, `local`, `relayed` and `secure`.
  - `Session` with `id`, `bandwidth` and `location` (`lan` or `wan`).
  - `Part.decision`, one of `directplay`, `transcode` or `none`, and per-`Stream` `decision`, one of `copy`, `transcode`, `burn`, `unavailable`, `ignore` or `none`.
  - A `TranscodeSession` object with `sourceVideoCodec`, `sourceAudioCodec`, `protocol`, `speed`, `throttled`, `transcodeHwRequested` and `transcodeHwFullPipeline`.
- `POST /status/sessions/terminate?sessionId=&reason=` kicks a session and shows `reason` in the client. It returns 401 when the "Server does not have the feature enabled" ([statusPostTerminate](https://developer.plex.tv/pms/#tag/Status/operation/statusPostTerminate)). **Inferred:** the 401 refers to a Plex Pass gate. The spec does not name the gate.
- `GET /status/sessions/history/all` lists playback history ([statusGetHistoryAll](https://developer.plex.tv/pms/#tag/Status)).
- A third party cannot read a client's capabilities. A client names a built-in profile with `X-Plex-Client-Profile-Name` ("Generally should only be used to specify the Generic profile"). It extends that profile per request with `X-Plex-Client-Profile-Extra` directives: `add-direct-play-profile`, `add-limitation`, `add-transcode-target` and others ([Profile Augmentations](https://developer.plex.tv/pms/#section/API-Info/Profile-Augmentations)). No endpoint lists those profiles.
- `GET /{transcodeType}/:/transcode/universal/decision` returns a playback decision for a profile plus settings. Its response includes `directPlayDecisionCode/Text`, `transcodeDecisionCode/Text` and `availableBandwidth` ([transcodeDecision](https://developer.plex.tv/pms/#tag/Transcoder/operation/transcodeDecision)). **Inferred:** SlipStream could call this with the Generic profile plus augmentations to predict direct play. It cannot ask "what does this user's TV support".
- `GET /` reports `transcoderActiveVideoSessions` and the `transcoderVideo` capability flags ([getSlash](https://developer.plex.tv/pms/#tag/General/operation/getSlash)).

### Jellyfin

- `GET /Sessions` returns `SessionInfoDto[]`. It takes `controllableByUserId`, `deviceId` and `activeWithinSeconds`, and needs only `[Authorize]`. The server passes `IsApiKey` to the session filter ([SessionController.cs L52-L69](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Api/Controllers/SessionController.cs#L52-L69)).
- Each session carries the following (OpenAPI `SessionInfoDto`, `TranscodingInfo`, `TranscodeReason`, `HardwareAccelerationType`):
  - `UserName`, `Client`, `DeviceName`, `DeviceType`, `ApplicationVersion`, `RemoteEndPoint` and `NowPlayingItem`.
  - `PlayState.PlayMethod`, one of `DirectPlay`, `DirectStream` or `Transcode`.
  - `TranscodingInfo` with `IsVideoDirect`, `IsAudioDirect`, codecs, size, `HardwareAccelerationType` (`none`, `amf`, `qsv`, `nvenc`, `v4l2m2m`, `vaapi`, `videotoolbox` or `rkmpp`) and `TranscodeReasons[]` (28 values, such as `VideoCodecNotSupported`, `ContainerBitrateExceedsLimit` and `VideoRangeTypeNotSupported`).
- `Capabilities` on each session includes the client's `DeviceProfile` as the client posted it via `POST /Sessions/Capabilities/Full` ([DeviceManager.cs L299-L311](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Server.Implementations/Devices/DeviceManager.cs#L299-L311), [SessionManager.cs L1243](https://github.com/jellyfin/jellyfin/blob/v12.1/Emby.Server.Implementations/Session/SessionManager.cs#L1243)). `GET /Devices` returns `DeviceInfoDto.Capabilities` as well (OpenAPI). A `DeviceProfile` holds `MaxStreamingBitrate`, `DirectPlayProfiles`, `TranscodingProfiles`, `ContainerProfiles`, `CodecProfiles` and `SubtitleProfiles` ([DeviceProfile.cs](https://github.com/jellyfin/jellyfin/blob/v12.1/MediaBrowser.Model/Dlna/DeviceProfile.cs)). **Inferred:** the profile is only as complete as the client app made it, and a client that posts only basic capabilities has none.
- `POST /Items/{itemId}/PlaybackInfo` accepts a `DeviceProfile` and returns the server's decision per media source (OpenAPI `GetPostedPlaybackInfo`).
- Control works through commands. `POST /Sessions/{id}/Playing/Stop` (`PlaystateCommand`), `POST /Sessions/{id}/Message` and `POST /Sessions/{id}/Command/{GeneralCommandType}` cover it (OpenAPI). No single call terminates a session with a reason.

### Emby

- `GET /Sessions` returns `SessionInfo[]` with `PlayState.PlayMethod`, `TranscodingInfo`, `PlayableMediaTypes`, `SupportedCommands`, `RemoteEndPoint` and `Protocol` (Emby.SDK@1faf817 spec, `getSessions` and `Session.SessionInfo`).
- `TranscodingInfo` is the richest of the three on hardware. It has `VideoDecoder`, `VideoDecoderIsHardware`, `VideoDecoderHwAccel`, `VideoEncoder`, `VideoEncoderIsHardware`, `VideoEncoderHwAccel`, `VideoPipelineInfo`, `ProcessStatistics`, `CurrentThrottle` and `TranscodeReasons[]` (27 values) (same spec).
- The session DTO has no `Capabilities` field. `Devices.DeviceInfo` has none either (same spec). Clients do post `ClientCapabilities` with a `DeviceProfile` to `/Sessions/Capabilities/Full`. **Inferred:** Emby keeps the profile internally but does not expose it to a third party.
- `POST /Items/{Id}/PlaybackInfo` exists, as in Jellyfin (same spec).
- Control uses the same command family as Jellyfin: `POST /Sessions/{Id}/Playing/{Command}`, `/Message` and `/Command/{Command}` (same spec). The [Remote-Control](https://github.com/MediaBrowser/Emby.SDK/blob/1faf8176df7ede9152c38f27a5925e6ccaf5e12b/Documentation/doc/restapi/Remote-Control.html) doc covers the details.

## 4. Auth

### Plex

- PMS requests carry `X-Plex-Token` and normally `X-Plex-Client-Identifier`. Any `X-Plex-*` header may be sent as a query parameter instead ([Headers](https://developer.plex.tv/pms/#section/API-Info/Headers)). The API version is chosen with `X-Plex-Pms-Api-Version`, and version `0.0` is assumed without it ([API Versioning](https://developer.plex.tv/pms/#section/API-Info/API-Versioning)).
- Tokens come from plex.tv ([Authenticating with Plex](https://developer.plex.tv/pms/#section/API-Info/Authenticating-with-Plex)). Two flows are documented:
  - **JWT (recommended).** The device generates an Ed25519 key. It creates a PIN at `POST https://clients.plex.tv/api/v2/pins` with its JWK, and the user approves it at `app.plex.tv/auth` (or `plex.tv/link` for 4-digit PINs). The device then exchanges the PIN with a signed device JWT. Tokens "expire after 7 days" and are refreshed with nonce, device JWT and `POST /api/v2/auth/token`. A refresh works "at any time, including after expiration". Registering a JWK with an existing legacy token makes "Your legacy token ... expire after this process".
  - **Legacy.** `POST https://plex.tv/api/v2/pins?strong=true`, then the user signs in at `app.plex.tv/auth#?clientID=&code=&forwardUrl=`, then the app polls `GET /api/v2/pins/{id}` for `authToken`. `GET https://plex.tv/api/v2/user` returns 200 or 401 to validate a stored token.
- The account token is not the server token. "Once you have a token to talk to plex.tv, you will need to obtain a different set of tokens used to talk to PMS instances." `GET https://clients.plex.tv/api/v2/resources?includeHttps=1&includeRelay=1&includeIPv6=1` returns each server's `accessToken` and connection URLs (same section).
- `POST /security/token?type=delegation&scope=all` mints a transient token with the caller's access level. It is "valid for up to 48 hours" and destroyed on server restart ([securityPostToken](https://developer.plex.tv/pms/#tag/General/operation/securityPostToken)).
- **Inferred:** v1's "Connect to Plex" PIN flow is the legacy flow. If SlipStream moves to JWT, it must store an Ed25519 private key per install and run a refresh at least every 7 days, or after the fact, to keep a working token.

### Jellyfin

- `AuthorizationContext` reads the token from `Authorization: MediaBrowser Client="...", Device="...", DeviceId="...", Version="...", Token="..."` or from the `ApiKey` query parameter. It accepts `X-Emby-Token`, `X-MediaBrowser-Token`, `X-Emby-Authorization`, the `Emby` scheme and the `api_key` query parameter only when `EnableLegacyAuthorization` is true ([AuthorizationContext.cs L84-L111](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Server.Implementations/Security/AuthorizationContext.cs#L84-L111), [L229-L261](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Server.Implementations/Security/AuthorizationContext.cs#L229-L261)).
- `EnableLegacyAuthorization` defaults to false ([ServerConfiguration.cs L290](https://github.com/jellyfin/jellyfin/blob/v12.1/MediaBrowser.Model/Configuration/ServerConfiguration.cs#L290)). A 12.x migration forces it to false on existing installs too ([20260531160000_DisableLegacyAuthorization.cs](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Server/Migrations/Routines/20260531160000_DisableLegacyAuthorization.cs)). **Inferred:** a client that sends Emby-style `X-Emby-Token` gets 401 from a default Jellyfin 12 server.
- API keys are listed, created and revoked through `GET/POST /Auth/Keys` and `DELETE /Auth/Keys/{key}`, all `RequiresElevation` ([ApiKeyController.cs L36-L55](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Api/Controllers/ApiKeyController.cs#L36-L55)). `POST /Auth/Keys?app=` returns 204 with no body, so the key must be read back from `GET /Auth/Keys` (OpenAPI `CreateKey`). An API key authenticates as the server itself, with `IsApiKey` set ([AuthorizationContext.cs L195-L216](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Server.Implementations/Security/AuthorizationContext.cs#L195-L216)).
- User login uses `POST /Users/AuthenticateByName {Username, Pw}`, which returns `AccessToken` (OpenAPI).
- Quick Connect is a PIN-style login with no password. The steps are as follows (OpenAPI `InitiateQuickConnect`, `GetQuickConnectState`, `AuthenticateWithQuickConnect`, [QuickConnectController.cs L104-L105](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Api/Controllers/QuickConnectController.cs#L104-L105)):
  1. The app calls `POST /QuickConnect/Initiate` and gets `Secret` and `Code`.
  2. A signed-in user approves `Code` with `POST /QuickConnect/Authorize`.
  3. The app polls `GET /QuickConnect/Connect?secret=`.
  4. The app exchanges the secret at `POST /Users/AuthenticateWithQuickConnect`.

  It is on by default through `QuickConnectAvailable = true` ([ServerConfiguration.cs L83](https://github.com/jellyfin/jellyfin/blob/v12.1/MediaBrowser.Model/Configuration/ServerConfiguration.cs#L83)).

### Emby

- API keys are created in the dashboard under Advanced, then Security. A key is sent as the `X-Emby-Token` header or the `api_key` query parameter ([API-Key-Authentication](https://github.com/MediaBrowser/Emby.SDK/blob/1faf8176df7ede9152c38f27a5925e6ccaf5e12b/Documentation/doc/restapi/API-Key-Authentication.html)). The spec also has `GET/POST /Auth/Keys`, admin only (spec `postAuthKeys`).
- User auth uses `Authorization: Emby UserId="...", Client="...", Device="...", DeviceId="...", Version="..."` plus `POST /Users/AuthenticateByName` with `pw`. The returned `AccessToken` goes in `X-Emby-Token` ([User-Authentication](https://github.com/MediaBrowser/Emby.SDK/blob/1faf8176df7ede9152c38f27a5925e6ccaf5e12b/Documentation/doc/restapi/User-Authentication.html)). The spec says the header may be named `Authorization` or `X-Emby-Authorization` (`postUsersAuthenticatebyname`).
- Emby Connect is an optional cloud account. Sign-in at `connect.emby.media` returns a server list with an `AccessKey` per server. `GET /Connect/Exchange` on the server swaps that key for a local `AccessToken`, and the exchange "must be done each time you connect" ([Emby-Connect](https://github.com/MediaBrowser/Emby.SDK/blob/1faf8176df7ede9152c38f27a5925e6ccaf5e12b/Documentation/doc/restapi/Emby-Connect.html)).
- The documented base URL includes an `/emby` prefix, for example `http://localhost:8096/emby/System/Info?api_key=...` (API key doc, and spec `servers: http://emby.media/emby`). The Jellyfin spec has no such prefix.

### Where Jellyfin and Emby auth diverge

The two servers share request bodies, but a default Jellyfin 12 rejects Emby's documented header and query names (sources above). The working forms are:

| Purpose | Jellyfin 12 default | Emby 4.10 documented |
| --- | --- | --- |
| Token header | `Authorization: MediaBrowser Token="..."` | `X-Emby-Token: ...` |
| Token query | `ApiKey=` | `api_key=` |
| Client identity scheme | `MediaBrowser ...` | `Emby ...` |

**Inferred:** one shared Jellyfin/Emby client needs a per-server auth strategy. Whether Emby accepts the `MediaBrowser` scheme is not documented and was not tested.

## 5. Remote access status

### Plex

- `GET /` exposes `myPlexMappingState` and `myPlexSigninState`, plus `myPlex`, `myPlexUsername`, `myPlexSubscription` and `certificate` ([getSlash](https://developer.plex.tv/pms/#tag/General/operation/getSlash)). The spec gives only the examples `"mapped"` and `"ok"` and no enum.
- `GET /identity` reports `claimed`, which says whether the server has been claimed by a user ([getIdentity](https://developer.plex.tv/pms/#tag/General/operation/getIdentity)).
- The plex.tv resources call lists connection URLs. "Connections labeled as `local` should be preferred ... and `relay` should only be used as a last resort as bandwidth on relay connections is limited" ([Authenticating with Plex](https://developer.plex.tv/pms/#section/API-Info/Authenticating-with-Plex)). The spec's server URL template is `https://{IP}.{identifier}.plex.direct:{port}` (spec `servers`).
- Server settings are read and written through `GET/PUT /:/prefs` ([preferencesGetSlash](https://developer.plex.tv/pms/#tag/Preferences)). The spec does not enumerate remote-access pref IDs.
- The Remote Access settings page shows three states: enabled, not configured, and disabled. It also has "Manually specify public port", and you "MUST enable this setting" after a manual port forward ([support.plex.tv: Remote Access](https://support.plex.tv/articles/200289506-remote-access/)). Plex does not document who runs the reachability test.

### Jellyfin

- Remote access is a config switch, not a measured state. `NetworkConfiguration.EnableRemoteAccess` defaults to true ([NetworkConfiguration.cs L129](https://github.com/jellyfin/jellyfin/blob/v12.1/MediaBrowser.Common/Net/NetworkConfiguration.cs#L129)). `ShouldAllowServerAccess` allows LAN addresses, rejects everything else when the switch is off, and otherwise applies `RemoteIPFilter` ([NetworkManager.cs L731-L767](https://github.com/jellyfin/jellyfin/blob/v12.1/src/Jellyfin.Networking/Manager/NetworkManager.cs#L731-L767)). Each user also has an `EnableRemoteAccess` permission ([UserManager.cs L633](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Server.Implementations/Users/UserManager.cs#L633)).
- `EnableUPnP` is marked `[Obsolete("No longer supported")]` ([NetworkConfiguration.cs L111-L114](https://github.com/jellyfin/jellyfin/blob/v12.1/MediaBrowser.Common/Net/NetworkConfiguration.cs#L111-L114)). Jellyfin no longer maps ports.
- `SystemInfo` and `PublicSystemInfo` report `LocalAddress` but no WAN address (OpenAPI). A search of the v12.1 source for `WanAddress` finds nothing.
- `GET /System/Endpoint` returns `IsLocal` and `IsInNetwork` for the caller's own request ([SystemController.cs L185-L194](https://github.com/jellyfin/jellyfin/blob/v12.1/Jellyfin.Api/Controllers/SystemController.cs#L185-L194)).
- Reverse proxy and address publishing are configured through `KnownProxies`, `BaseUrl`, `PublishedServerUriBySubnet`, `PublicHttpPort` and `PublicHttpsPort` (OpenAPI `NetworkConfiguration`, [NetworkConfiguration.cs L144-L165](https://github.com/jellyfin/jellyfin/blob/v12.1/MediaBrowser.Common/Net/NetworkConfiguration.cs#L144-L165)). **Inferred:** behind SlipStream's forwarder, Jellyfin judges LAN or remote by the forwarder's IP unless that IP is listed in `KnownProxies`.
- Jellyfin's docs describe the server as "entirely self-hosted". Internet exposure is left to reverse proxies, VPNs or port forwarding, and no relay service exists ([jellyfin.org networking](https://jellyfin.org/docs/general/post-install/networking/)).

### Emby

- `SystemInfo` and `PublicSystemInfo` include `WanAddress`, `RemoteAddresses[]`, `LocalAddress` and `LocalAddresses[]` (Emby.SDK@1faf817 spec). `/System/Info/Public` needs no auth, while `/System/Info` requires a user.
- `ServerConfiguration` has `EnableRemoteAccess`, `EnableUPnP`, `PublicPort`, `PublicHttpsPort`, `IsPortAuthorized`, `WanDdns`, `RequireHttps`, `RemoteIPFilter` and `PreferredDetectedRemoteAddressFamily` (same spec). **Inferred:** Emby still does UPnP mapping, unlike Jellyfin. The spec describes no reachability test.
- Emby Connect returns each server's `Url` (remote access URL) and `LocalAddress` ([Emby-Connect](https://github.com/MediaBrowser/Emby.SDK/blob/1faf8176df7ede9152c38f27a5925e6ccaf5e12b/Documentation/doc/restapi/Emby-Connect.html)).
- `GET /System/Endpoint` returns `IsLocal` and `IsInNetwork`, as in Jellyfin (same spec).
- Clients discover servers on the LAN with a UDP broadcast "who is EmbyServer?" on port 7359 ([Locating-the-Server](https://github.com/MediaBrowser/Emby.SDK/blob/1faf8176df7ede9152c38f27a5925e6ccaf5e12b/Documentation/doc/restapi/Locating-the-Server.html)).

## Options and tradeoffs for the Media Server contract

These are the facts the contract decision has to absorb. This document makes no decision.

1. **Refresh granularity.** All three support a path-scoped refresh, so a contract operation like "these paths changed" maps onto every server. Plex needs a section ID plus a path, so SlipStream must resolve which section owns the path from `Location[]`. Jellyfin and Emby take bare paths and resolve the library themselves. Jellyfin adds a 60 s default debounce before the scan starts. A contract that waits for "scan finished" needs per-server polling or events: Plex `refreshing` and `/activities`, and Jellyfin and Emby `RefreshStatus` or WebSocket messages.
2. **Permission needed.** Path-scoped refresh needs only a user-level credential on Jellyfin and Emby. Reading library locations needs admin on Jellyfin and only a user on Emby. Full refresh needs admin on both. Plex is all-or-nothing per server token.
3. **Path mapping.** No server rewrites inbound paths, so translation lives in SlipStream whatever the contract looks like. All three expose library roots, which allows auto-discovery and validation of mappings.
4. **Player capabilities.** Only Jellyfin exposes each client's device profile. Plex and Emby expose the per-session outcome (direct play or transcode, reasons, hardware use) but not what the client could have played. A capability-driven Rendition choice (Q8, Q14) would be exact on Jellyfin and inferred on the other two. The inference could come from session history, or from a decision or PlaybackInfo query with a guessed profile.
5. **Session control.** Plex has a single terminate-with-reason call. Jellyfin and Emby need a stop command plus a separate message.
6. **Auth.** Plex needs a plex.tv round trip and a separate per-server token, and its recommended flow rotates tokens every 7 days. Jellyfin and Emby take a static admin-issued API key, but their header and query names now differ by default.
7. **Remote access.** Only Plex reports a measured mapping state, through `myPlexMappingState`. Emby reports its detected WAN address. Jellyfin reports only configuration. A SlipStream forwarder (Q10) that fronts the Media Server would need its own reachability check for Jellyfin and Emby to match what Plex reports. **Inferred.**

## Not verified

- Plex notification event type names and the `myPlexMappingState` enum. The spec gives neither.
- Plex remote-access pref IDs under `/:/prefs`.
- Emby runtime behavior: the debounce of `/Library/Media/Updated`, what `PathSubstitutions` affects, whether the `MediaBrowser` auth scheme works, and whether `WanAddress` is measured or configured. Emby Server is closed source, and no Emby server was run.
- Plex webhooks. The support article returned HTTP 403 to the fetcher.
- No live server of any of the three was exercised. Every claim comes from specs, docs or source.
