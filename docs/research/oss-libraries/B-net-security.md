# B. Identity, security, networking and distribution: libraries for SlipStream v2

> Spec numbers follow the feature graph as renumbered on 2026-10-11, with seven foundation specs (S01 to S07) ahead of the walking skeleton (S09). "S01-S09" means the foundations plus the skeleton.


Researched 2026-10-11. **How it was verified:** latest tags, their dates and branch-head commit dates come from `proxy.golang.org` (`@latest`, `@v/main|master.info`). Stars, open issues and the OpenSSF Scorecard "Maintained" line come from `api.deps.dev`. Licenses, `go.mod` and source details were read from the module zips. The GitHub API was not reachable from this session, so "open issues" is deps.dev's count. "Maint 90d" is Scorecard's count of commits in the last 90 days. Anything marked *inferred* was not checked against a source this session.

License note: deps.dev reports "non-standard" for several repos, including yamux, goupnp, kardianos/service, zeebo/blake3 and klauspost/compress. I read the actual files for yamux (MPL-2.0), libp2p go-yamux (MPL-2.0 plus BSD-3), zeebo/blake3 (CC0-1.0, which the FSF lists as GPL-compatible) and goupnp (BSD-2, per pkg.go.dev). Licenses for klauspost/compress and kardianos/service are *unverified* this session: klauspost is believed to be BSD-3 plus Apache-2.0, kardianos zlib. Either would be GPL-compatible. None of the candidates in this area is AGPL, SSPL, BSL or GPL-2.0-only.

---

## 1. Node link: pinned-key TLS with Ed25519 (S15, S31, S50, S52)

**Current plan:** ADR 0013 and ADR 0003 decide on TLS 1.3 authenticated by each Node's pinned Ed25519 key, with no certificate authority. The ADRs don't say how the key gets into TLS.

**Raw public keys or self-signed certificates.** Go's `crypto/tls` has no RFC 7250 raw-public-key certificate type. `tls.Certificate` carries X.509 DER only, and `Config` has no certificate-type option. That is *inferred from the API*: I found no golang/go proposal for it. The usual pattern, used by libp2p TLS, Syncthing device IDs, and Go packages such as `leantls` and `bazil.org/bazil/util/edtls`, is this:

- Each Node mints a long-lived self-signed X.509 certificate over its Ed25519 key. TLS 1.3 in Go supports Ed25519 certificates.
- The client sets `InsecureSkipVerify: true`.
- The server sets `ClientAuth: tls.RequireAnyClientCert`.
- Both sides verify in a callback: exactly one certificate, its public key is Ed25519, and the key equals the pinned key. Expiry, names and chain are ignored.

**Risks to design around (none reopens ADR 0013):**
- **Use `VerifyConnection`, not `VerifyPeerCertificate`.** Go's docs say `VerifyPeerCertificate` is not called on resumed connections, while `VerifyConnection` runs on every handshake, resumed ones included. Alternatively, turn off session tickets (`SessionTicketsDisabled`) on the link. Getting this wrong means a resumed TLS 1.3 session skips the pin check.
- **Pin the raw 32-byte Ed25519 key, not a hash of the certificate.** The certificate can then be re-minted (for example, if it was given a far-future `NotAfter`) without re-pairing.
- **Write a test for every rejection path:** a wrong key, an RSA certificate, two certificates, and no client certificate. The fake world's Node-link fault proxy is the natural place for these tests.

| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| Go stdlib `crypto/tls` plus a ~80-line verifier | go.dev | BSD-3 | Go 1.27 | — | stdlib | yes (API) |
| libp2p TLS security transport (reference only) | github.com/libp2p/go-libp2p | MIT | v0.50.0 (2026-09-21) | 2026-09 | 6.9k★, active | yes |

**Verdict: build**, borrowing libp2p's handshake checks as a reference. The custom code is about 80 lines plus tests. Risk is low if `VerifyConnection` is used. A dependency would buy nothing.
**Criticality:** critical infra (security).

## 2. Node link multiplexer: yamux (S15, S31, S52)

**Current plan:** yamux, decided in ADR 0013. The ADR doesn't name a module. The candidates are `hashicorp/yamux` and the libp2p fork.

| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| hashicorp/yamux | github.com/hashicorp/yamux | MPL-2.0 | v0.1.2 (2024-09-06) | 2026-09-29 | 2.7k★, 34 issues. Commits since 2025-12 are CI, copyright and CODEOWNERS only. Ownership moved to nomad-eng in 2026-05. Maint 90d: 4 | yes |
| libp2p/go-yamux v5 | github.com/libp2p/go-yamux | MPL-2.0 + BSD-3 | v5.1.0 (2025-07-29) | 2025-08-21 | Small repo (42★), but it is the muxer in go-libp2p and Kubo. It has RTT-measured window auto-tuning (`InitialStreamWindowSize`, `MaxStreamWindowSize`, `MeasureRTTInterval`) and speaks the same yamux wire spec | yes (source) |
| xtaci/smux | github.com/xtaci/smux | MIT | v1.5.57 (2026-03-13) | 2026-05 | 1.6k★, essentially one maintainer, used by kcptun and frp. Its v2 protocol has per-stream windows | yes |
| quic-go streams + datagrams | github.com/quic-go/quic-go | MIT | v0.63.0 (2026-09-22) | 2026-10-11 | 11.8k★, very active | yes |

**Real risks:**
1. **Per-stream throughput cap in hashicorp/yamux.** I read `const.go` and `stream.go` in v0.1.2. The window starts at a fixed 256 KiB and only grows to `MaxStreamWindowSize`, which defaults to the same 256 KiB. With the defaults, one stream tops out at about window ÷ RTT: roughly 21 Mbit/s at 100 ms, or 7 Mbit/s at 300 ms to a VPS. That matters for the Relay (S52), which forwards remote Plex and UI streams down the Node link. Transfers are mostly unaffected, since ADR 0004 already puts them on separate TLS connections. Raising `MaxStreamWindowSize`, for example to 8–16 MiB, fixes it, but memory is then bounded only by window × streams. libp2p/go-yamux auto-tunes the window by measured RTT.
2. **UDP over a TCP multiplexer.** S52 forwards UDP over the Node link (ADR 0012), so every datagram inherits TCP head-of-line blocking and retransmit latency. That suits occasional Plex discovery or GDM-style traffic and is poor for anything latency-sensitive. QUIC datagrams would avoid it, but QUIC would reopen ADR 0013 and fails where UDP is blocked. **Flag, don't reopen.** Document that UDP forwards are best-effort.
3. **Maintenance.** hashicorp/yamux is stable but in caretaker mode: no tag in two years and no functional commits in the last year. It is mature and widely deployed (Consul, Nomad, go-plugin), so the protocol risk is low. Bugs would need a fork.

**Verdict: wrap** `hashicorp/yamux` behind a small `link.Session` interface (open stream, accept stream, close), with `MaxStreamWindowSize` set explicitly. Keep libp2p/go-yamux v5 as the drop-in alternative, since it speaks the same protocol, if relay throughput measurements need auto-tuning. A custom muxer would be 1–2k lines of flow-control code and is not worth it.
**Criticality:** critical infra (if the link stops, the whole multi-Node app stops).

## 3. connect-go over yamux (S15)

**Current plan:** connect-go 2.x over the link, decided in ADR 0013. `connectrpc.com/connect/v2` v2.0.0 was tagged 2026-10-07, four days before this research (verified). v1.21.0 is from 2026-09-08.

**Is it a known pattern?** Not as connect-go specifically. I found no example. The same shape is well proven with gRPC: hashicorp/go-plugin runs gRPC over yamux streams (v1.8.0, 2026-04, 6.1k★, verified as active). The wiring follows:
- A yamux `Session` already satisfies `net.Listener`.
- Each side runs an `http.Server` on its accepted streams with `Protocols` set to unencrypted HTTP/2 (Go 1.24+). Each side also has an `http.Client` whose transport's `DialContext` opens a yamux stream, with unencrypted HTTP/2 on.
- Calls flow both ways regardless of who dialed, as ADR 0003 requires.

**Two design notes:**
- **Double multiplexing.** HTTP/2 multiplexes over one yamux stream. Either accept that (one yamux stream per HTTP/2 connection, and HTTP/2 flow control on top of yamux's), or use HTTP/1.1 with one yamux stream per call, which loses bidirectional streaming. HTTP/2 is simpler and keeps bidi.
- **Window tuning.** Both HTTP/2 windows and yamux windows need to be raised (see section 2).

**Verdict: build** the roughly 100-line adapter. No library exists, and none is needed.
**Criticality:** critical infra.

## 4. Pairing and the Join Code (S15, S25)

**Current plan:** GLOSSARY.md defines the Join Code as "short-lived, single-use", and #93 Q6 says it carries the Controller's addresses. The ADRs don't fix its cryptography.

**What existing tools do:**
- **magic-wormhole** uses SPAKE2 on short, typed codes such as `7-crossover-clockwork`. The Go port, `psanford/wormhole-william`, uses `salsa.debian.org/vasudev/gospake2`.
- **croc** uses `schollz/pake`. Its 2023 CVEs, CVE-2023-43617 to 43621, were protocol and usage bugs in croc, not in the PAKE.
- **Syncthing** exchanges device-key fingerprints.
- **kubeadm and Docker Swarm** paste a high-entropy token plus a CA hash.

| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| filippo.io/cpace (CPace on ristretto255) | github.com/FiloSottile/cpace | BSD-3 | untagged (2021-01-01) | 2021-01 | README: "experimental … really only a weekend project … not standardized"; loosely based on draft-haase-cpace-01 | yes (zip) |
| schollz/pake v3 | github.com/schollz/pake | MIT | v3.2.0 (2026-07-31) | 2026-07 | 240★, one maintainer. Uses Boneh–Shoup PAKE2 over an `siec` curve | yes (zip) |
| gospake2 | salsa.debian.org/vasudev/gospake2 | *unverified* | untagged (2021-05-10) | 2021-05 | stale | yes (proxy) |
| No PAKE: a high-entropy code that embeds the Controller key | — | — | — | — | — | — |

**Verdict: build. Use no PAKE.** The Join Code is pasted (UI to CLI or env, or a Worker's Node address pasted back), never typed on a TV. It can therefore carry the following, encoded as base32 or base64url in about 90–120 characters:
- the Controller's addresses,
- the Controller's full Ed25519 public key, or a 128-bit or longer hash of it,
- a 128-bit one-time secret.

The Worker pins the Controller key from the code, completes the pinned-key TLS handshake from section 1, then presents the secret and its own key. The Controller burns the secret on first use. This gives mutual authentication without a PAKE. PAKE only pays off for short, human-typed codes, and the Go PAKE options are experimental (CPace), single-maintainer (pake) or stale (gospake2). Cost: about 150 lines plus tests.
**Criticality:** critical infra (security).

## 5. Accounts S12: passkeys, PIN, sessions, CSRF, rate limiting, API keys, secrets at rest

### 5a. WebAuthn (S12; also S25 first Admin)
**Current plan:** ADR 0005 sets one relying party per address and no passkeys at IP addresses. No library is decided. v1 used go-webauthn.

| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| go-webauthn/webauthn | github.com/go-webauthn/webauthn | BSD-3 | v0.18.2 (2026-09-19) | 2026-10-10 | 1.3k★, 10 issues, Maint 90d: 30 | yes |

go-webauthn supports per-ceremony relying parties: `WithRegistrationRelyingPartyID`, `WithLoginRelyingPartyID`, `WithRegistrationOrigin` and `WithLoginOrigin` (verified on pkg.go.dev). It also binds the origin into `SessionData`. ADR 0005's rule of one relying party per address therefore needs one `webauthn.WebAuthn` whose `RPOrigins` lists every configured HTTPS address. Each ceremony then passes the RP ID and origin of the request's host. Alternatively, build one cheap `WebAuthn` per address.

**Caveats:**
- The API is still pre-1.0. Expect breaking minor versions.
- The MDS (metadata service) feature fetches from the FIDO servers. Leave it unset, because it is a network boundary with no `Endpoints` entry (ADR 0014).

**Verdict: wrap.** The custom alternative would be COSE, CBOR, attestation formats and the security review that comes with them, several thousand lines. That is not reasonable. The frontend needs nothing. v1's `@simplewebauthn/browser` is optional now that browsers have `PublicKeyCredential.parseCreationOptionsFromJSON`; that claim is *inferred*.
**Criticality:** critical infra (security).

### 5b. PIN hashing (S12)
**Current plan:** ADR 0005 stores PIN failures and a hard stop. The ADRs don't name a hash.

**Candidate:** `golang.org/x/crypto/argon2`. x/crypto is BSD-3, v0.58.0 (2026-10-09), and active (verified).

**Verdict: adopt argon2id, plus a secret pepper.** A 4-digit PIN has 10⁴ values, so argon2id alone falls to an offline attacker holding the database in seconds. The real defence is a keyed hash: HMAC with the credential key from 5f, then argon2id. A stolen database is then useless without the key file. ADR 0018 already puts that key file inside the Backup's encrypted layer.

Also run a dummy hash for unknown emails, so timing doesn't reveal which Accounts exist. Cost: about 40 lines.
**Criticality:** critical infra (security).

### 5c. Sessions (S12)
**Current plan:** "cookie sessions" in S12. Unspecified otherwise.

| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| alexedwards/scs v2 + sqlite3store | github.com/alexedwards/scs | MIT | v2.9.0 (2025-04-17); sqlite3store untagged (2025-10-02) | 2025-10 | 2.6k★, Maint 90d: 0 | yes |
| gorilla/sessions | github.com/gorilla/sessions | BSD-3 | v1.4.0 (2024-08-20) | 2024-08 | dormant | yes |

**Verdict: build.** S12 needs things scs makes awkward:
- the Admin re-confirm timestamp (15 minutes),
- a session list per Account, with "sign out everywhere" on PIN reset,
- sessions that survive Handover in the database.

scs stores gob-encoded blobs that SQL can't query, and its store schema sits outside the per-module goose and sqlc convention. The custom version is a `sessions` table that stores SHA-256(token), plus account, created and last-seen times, re-confirm time and address. The cookie holds a random 256-bit token with `HttpOnly`, `SameSite=Lax`, and `Secure` on HTTPS. About 200 lines.
**Criticality:** critical infra (security).

### 5d. CSRF (S12)
**Candidate:** Go 1.25+ `net/http.CrossOriginProtection`, which uses `Sec-Fetch-Site` and falls back to comparing `Origin` with `Host`. gorilla/csrf v1.7.3 (2025-01) is dormant and token-based.

**Verdict: adopt the stdlib.** It fits multiple addresses, because each address is same-origin with itself. Two caveats:
- Browsers send `Sec-Fetch-*` only to secure contexts, so plain-HTTP LAN addresses use the `Origin`/`Host` fallback. That fallback can't tell `http://host` from `https://host` (Alex Edwards' write-up). Pair it with `SameSite=Lax` cookies.
- Requests that carry an API key have no browser headers and pass through. That is fine, since they don't use cookies.

GET must stay side-effect free.
**Criticality:** critical infra (security).

### 5e. Per-IP rate limiting (S12)
| Candidate | Repo | License | Latest | Last commit | Verified |
|---|---|---|---|---|---|
| golang.org/x/time/rate | go.googlesource.com/time | BSD-3 | v0.16.0 (2026-08-19) | 2026-08 | yes |
| sethvargo/go-limiter | github.com/sethvargo/go-limiter | Apache-2.0 | v1.2.0 (2026-07-17) | 2026-10 | yes |
| ulule/limiter v3 | github.com/ulule/limiter | MIT | v3.11.2 (2023-05) | 2024-10 | yes |

**Verdict: build** on `x/time/rate`: a map from IP or /64 to a limiter with LRU or idle eviction, about 60 lines. ADR 0005's lockout is a stored per-Account counter, so it is SQL anyway.

**Real risk:** relayed clients (ADR 0012) reach the Serving Node as yamux streams from the relay, not as TCP from their own IP. If the client address isn't carried, every remote client shares one "IP". One attacker could then rate-limit every remote user, and the limiter would be useless against that attacker. The relay must send the original remote address when it opens the forward stream. In-band link metadata is enough: no PROXY protocol is needed, because the Serving Node accepts the stream itself (S52 to S12 dependency). Media Server forwards stay unchanged, as ADR 0012 says.
**Criticality:** critical infra (security).

### 5f. API keys and credential encryption at rest (S12 P-350; ADR 0018)
| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| stdlib `crypto/cipher` AES-256-GCM, or x/crypto `chacha20poly1305` (XChaCha) | go | BSD-3 | — | — | stdlib / x | yes |
| tink-go v2 | github.com/tink-crypto/tink-go | Apache-2.0 | v2.8.0 (2026-08-11) | 2026-10 | 279★, Google, active | yes |
| x/crypto/nacl/secretbox | go | BSD-3 | — | — | frozen-style API | yes |
| filippo.io/age | github.com/FiloSottile/age | BSD-3 | v1.3.2 (2026-08-29) | 2026-08 | 23.8k★ | yes |

**Verdict: build.** Use XChaCha20-Poly1305 or AES-256-GCM with a random nonce. Bind each value with associated data `table|column|row id`, so a ciphertext can't be swapped into another row. Use a 32-byte key file beside the database that goes into the Backup's encrypted layer (ADR 0018), plus a key-version byte for rotation. About 100 lines.
- Tink adds keyset JSON, primitives registration and a large API surface for one primitive.
- age is a file format with large per-value overhead.
- Store API keys as SHA-256 of a random 256-bit token, compared in constant time. The key is shown once. About 40 lines.

**Criticality:** critical infra (security).

## 6. SMTP (S12 Signup and Reset emails; S37 and S42 email notifications)
| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| wneessen/go-mail | github.com/wneessen/go-mail | MIT | v0.8.1 (2026-07-09) | 2026-09-22 | 1.5k★, 16 issues, Maint 90d: 30 | yes |
| net/smtp (stdlib) | go | BSD-3 | — | — | frozen package: no MIME, no implicit-TLS helpers | yes (*frozen status inferred from package docs*) |
| emersion/go-smtp (client side) | github.com/emersion/go-smtp | MIT | v0.25.0 (2026-08-13) | 2026-09 | 2k★. ADR 0014 already names its **server** as the SMTP fake | yes |

**Verdict: wrap go-mail.** It covers MIME multipart (HTML plus text), STARTTLS and implicit TLS policies, and auth mechanisms. Its `WithDialContextFunc`, `WithTLSConfig` and `WithPort` options (verified) make it fakeable against the go-smtp server over loopback or `net.Pipe`. The custom alternative on `net/smtp` would need 300–500 lines of MIME and TLS policy code.
**Criticality:** important (Signup and Reset links depend on it).

## 7. Reachability diagnostic S51

**Current plan:** ADR 0012 and ADR 0014 name a fake STUN (pion/stun as the reference), a fake gateway (UPnP IGD, NAT-PMP, PCP) and connect-back.

| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| pion/stun v3 | github.com/pion/stun | MIT | v3.1.7 (2026-08-18) | 2026-08 | 861★, Maint 90d: 11, Pion org | yes |
| huin/goupnp | github.com/huin/goupnp | BSD-2 | v1.3.0 (2023-08-29) | 2025-04 | 462★, quiet. HTTP client is a **package global** (`HTTPClientDefault`), and SSDP goes to the fixed multicast group | yes |
| jackpal/go-nat-pmp | github.com/jackpal/go-nat-pmp | Apache-2.0 | v1.1.0 (2026-08-31) | 2026-09 | 135★, one maintainer. Fixed port 5351 *(inferred)* | yes |
| jackpal/gateway (default-gateway discovery) | github.com/jackpal/gateway | BSD-3 | v1.2.0 (2026-04-25) | 2026-09 | 249★ | yes |
| tailscale `net/portmapper` (UPnP + NAT-PMP + PCP) | github.com/tailscale/tailscale | BSD-3 | v1.104.1 (2026-10-07) | 2026-10 | 37k★. Importing it pulls in the giant `tailscale.com` module | yes |
| PCP-only Go libraries | — | — | none found maintained | — | — | *inferred* |

**Verdicts:**
- **STUN: adopt `pion/stun`.** The server address is an `Endpoints` entry, and the same library is already the fake (ADR 0014).
- **UPnP, NAT-PMP and PCP: build, borrowing from tailscale's `portmapper`.** goupnp's global HTTP client and hardcoded SSDP group conflict with ADR 0014's injectable-boundary rule. go-nat-pmp covers only one of the three protocols. tailscale's portmapper is the best-tested Go code for all three, including router quirks, but importing `tailscale.com` is far too heavy. Port the protocol files instead, with attribution in the notices file (ADR 0010).
  - Custom cost: about 400 lines for SSDP M-SEARCH plus IGD SOAP (GetExternalIPAddress, AddPortMapping), about 150 for NAT-PMP, about 200 for PCP.
  - The gateway and SSDP target addresses must be injectable so the fake gateway works over loopback.
- **Default gateway: borrow** from `jackpal/gateway`. It parses routes per OS and is BSD-3. Adopting it is acceptable too.

**Criticality:** important. The diagnostic, not data or security.

## 8. Relay Node S52: SNI pass-through and UDP

| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| inet.af/tcpproxy | github.com/inetaf/tcpproxy | Apache-2.0 | untagged (2023-11-02) | 2023-11 | 1.3k★, dormant. SNI and ACME routing helpers | yes |
| mholt/caddy-l4 | github.com/mholt/caddy-l4 | Apache-2.0 | v0.1.2 (2026-07-15) | 2026-07 | 1.8k★, active, but a Caddy module that brings in Caddy | yes |
| ClientHello peek with `crypto/tls` `GetConfigForClient` on a read-only recording conn (Andrew Ayer's "SNI proxy in 115 lines" trick) | — | — | — | — | uses Go's own parser | *inferred (well-known technique)* |

**Verdict: build.**
- **TCP and SNI:** peek the ClientHello with the stdlib trick, look up the SNI-to-forward map, open a yamux stream to the Serving Node, replay the peeked bytes, then splice. About 150 lines.
  - ACME TLS-ALPN-01 needs no special case: the challenge connection carries the same SNI and passes through to CertMagic at home.
  - tcpproxy is dormant, and its targets are dial addresses rather than "open a link stream".
  - caddy-l4 is far too heavy.
- **UDP: build.** Keep a per-(public port, client addr) flow table with idle timeouts. Frame datagrams with a length prefix on one link stream per flow. About 200 lines. Section 2 covers the head-of-line caveat.

**Criticality:** important (remote access); security-relevant for the SNI parse.

## 9. Updater S57: self-replace

**Current plan:** ADR 0015 decides on custom signed per-file manifests, stored and resumable phases, the current and previous versions kept, a health trial and rollback. Every install except Docker self-replaces.

| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| minio/selfupdate | github.com/minio/selfupdate | Apache-2.0 | v0.6.0 (2022-10-19) | 2025-10 | 935★, slow. About 300 lines in `apply.go`: rename to `.old`, rename the new file in, roll back on failure, hide `.old` on Windows. Has a minisign verifier through aead.dev/minisign v0.2.0 | yes (zip) |
| creativeprojects/go-selfupdate | github.com/creativeprojects/go-selfupdate | MIT | v1.6.0 (2026-07-08) | 2026-08 | 149★. Built around GitHub, GitLab or Gitea release discovery | yes |
| inconshreveable/go-update, rhysd/go-github-selfupdate, sanbornm/go-selfupdate | — | — | 2016 / 2021 / 2023 | — | abandoned | yes |

**Verdict: build, borrowing minio/selfupdate's Windows technique.** On Windows, a running `.exe` can be renamed but not deleted. Move it aside, delete it on the next start, and hide it until then. All the libraries swap one file. ADR 0015 swaps a set (the binary, ffmpeg, ffprobe and `dovi_tool`), keeps the previous set, and rolls back after a failed health trial. A versioned directory per release with an atomic switch fits that better than per-file renames. creativeprojects/go-selfupdate also owns release discovery, which ADR 0015 assigns to the Controller's manifest reader. Cost: about 300–400 lines plus tests on all three OSes.

**Real risk, macOS (S56 and S57):** replacing the executable inside a signed `.app` breaks the bundle's code-signature seal, and AMFI rejects it with `-67030`; that applies to the developer-ID case (eclecticlight 2018). Mitigations: swap the whole `.app` atomically, or, since #93 uses ad-hoc signing, re-sign the swapped bundle locally with `codesign -s -`. Whether macOS's App Management protection lets an ad-hoc-signed app modify its own bundle is *inferred and unverified*. **Add a macOS smoke test of a real update to S56's native smoke tests.**
**Criticality:** critical infra. A bad swap stops the app, and ADR 0015 makes the update path frozen.

## 10. Run modes S56: service, unit, tray, menu bar

**Current plan (#93 Q5):** Windows runs as a tray app only, with no service. macOS runs as a menu bar app with Start at login. Linux runs a system unit from deb and rpm, and `slipstream service install --user` writes a user unit with linger.

| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| kardianos/service | github.com/kardianos/service | zlib *(unverified)* | v1.3.0 (2026-07-06) | 2026-08 | 4.8k★, 166 issues, Maint 90d: 1 | yes |
| x/sys/windows/svc | go | BSD-3 | v0.49.0 (2026-10-06) | 2026-10 | not needed: no Windows service | yes |
| coreos/go-systemd v22 (`daemon.SdNotify`) | github.com/coreos/go-systemd | Apache-2.0 | v22.7.0 (2026-01-27) | 2026-07 | 2.7k★ | yes |
| fyne.io/systray | github.com/fyne-io/systray | Apache-2.0 | v1.12.2 (2026-06-09) | 2026-09-13 | 362★, Maint 90d: 9. Fork of getlantern that drops GTK on Linux | yes |
| energye/systray | github.com/energye/systray | Apache-2.0 | v1.0.3 (2026-01-31) | 2026-01 | 181★, quiet | yes |
| getlantern/systray | github.com/getlantern/systray | Apache-2.0 | v1.2.2 (2023-03-27) | 2023-11 | 3.7k★, **abandoned**, 114 issues | yes |

**Verdicts:**
- **Service and unit: build.** The Linux unit is a static file shipped by nfpm. The `--user` unit is a template plus `loginctl enable-linger`. Windows Start at login is an HKCU `Run` value through `x/sys/windows/registry`. macOS login is a LaunchAgent plist. kardianos's service abstraction doesn't match "tray app, not service". `sd_notify` is about 20 lines; adopting go-systemd for it is acceptable.
- **Tray: wrap `fyne.io/systray`.** The menu is four items. It is pure Go on Windows and uses cgo with Cocoa on macOS, and cgo is already on. On macOS it must own the main thread, so the headless or `--no-tray` path has to skip it cleanly. v1's tailscale/walk and darwinkit are heavier and Windows- or macOS-only.

**Criticality:** important.

## 11. Packaging S56: GoReleaser, nfpm, MSI, DMG, AppImage

| Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| GoReleaser OSS v2 | github.com/goreleaser/goreleaser | MIT | v2.18.3 (2026-10-08) | 2026-10 | 16k★, active | yes |
| GoReleaser Pro | goreleaser.com/pro | **closed-source EULA**, from $165/yr | — | — | Pro-only: split and merge builds (the cgo case), MSI, DMG, `.app`, notarization, prebuilt builder | yes (pricing page) |
| goreleaser-cross (Docker cross toolchains) | github.com/goreleaser/goreleaser-cross | MIT | *unverified* | *unverified* | 193★ | partly |
| nfpm v2 | github.com/goreleaser/nfpm | MIT | v2.47.0 (2026-06-20) | 2026-10 | 2.7k★, active; deb, rpm, apk, archlinux | yes |
| WiX Toolset v6/v7 | github.com/wixtoolset/wix | MS-RL. Open Source Maintenance Fee: v6 requires it from revenue-generating users; v7's EULA 1.1 only above US$10k/yr | — | — | build tool only; its license doesn't touch the output | yes (release notes) |
| mh-cbon/go-msi | github.com/mh-cbon/go-msi | MIT | untagged (2023-02) | 2023-02 | abandoned, wraps WiX 3 | yes |
| appimagetool / go-appimage | github.com/AppImage/appimagetool, github.com/probonopd/go-appimage | *unverified* / MIT | — | — | go-appimage: 1.2k★, Maint 90d: 2 | partly |

**Verdict: build the pipeline. Adopt nfpm, and adopt WiX and appimagetool as CI tools.** Static libtorrent and cgo already force a native runner per OS: macOS needs a Mac, and MSVC or MinGW libtorrent needs Windows. GoReleaser OSS can't merge per-runner cgo builds, and its MSI, DMG and `.app` support is Pro-only. Pro is closed-source, a dependency that clashes with the project's GPL-only ethos, though it is legal, since it's a build tool. ADR 0015's signed `manifest.json` is custom anyway.

A small Go `cmd/release` tool can do the whole release: build per OS job, run nfpm as a library for deb and rpm, run `wix build` on the Windows runner, run `hdiutil` for the DMG, run appimagetool for the static binary, then hash, write `manifest.json`, sign with minisign and upload. Cost: about 500–800 lines of release tooling against the YAML of GoReleaser. The WiX fee doesn't apply to a non-revenue project, but recheck the EULA before adopting v6 rather than v7.
**Criticality:** important. A broken release blocks updates.

## 12. Formats the ADRs chose, and which implementation

| Primitive (spec) | Candidate | Repo | License | Latest | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|---|
| minisign verify (S57, ADR 0015) | aead.dev/minisign | github.com/aead/minisign | MIT | v0.3.0 (2024-05-19) | 2025-01 | 140★. By Andreas Auernhammer (MinIO). Deps: x/crypto, x/term | yes (zip) |
| | jedisct1/go-minisign | github.com/jedisct1/go-minisign | MIT | untagged (2026-05-27) | 2026-05 | 120★. By minisign's author | yes (zip) |
| age (S49, S50, ADR 0018) | filippo.io/age | github.com/FiloSottile/age | BSD-3 | v1.3.2 (2026-08-29) | 2026-08 | 23.8k★, Filippo Valsorda | yes |
| zstd (S49, S50, ADR 0018) | klauspost/compress/zstd | github.com/klauspost/compress | BSD-3 + Apache-2.0 *(unverified)* | v1.20.1 (2026-09-25) | 2026-10 | 5.7k★, 5 issues, Maint 90d: 28 | yes |
| BLAKE3 (S31, ADR 0004) | zeebo/blake3 | github.com/zeebo/blake3 | CC0-1.0 | v0.2.4 (2024-08-14); **untagged head 2026-09-03** | 2026-09 | 498★, Maint 90d: 22. Head adds arm64 NEON and SVE2 assembly | yes (zip) |
| | lukechampine.com/blake3 | github.com/lukechampine/blake3 | MIT | v1.4.1 (2025-05-08) | 2025-05 | 425★. amd64 assembly only; arm64 is pure Go | yes (zip) |

**Verdicts:**
- **minisign: adopt `aead.dev/minisign`, with build as the fallback.** It verifies the standard prehashed (`ED`) format and is small. A custom verifier would be about 120 lines of Ed25519 plus BLAKE2b. That is attractive because the update path is frozen (ADR 0003), but nothing forces it.
  - Either way, pin by key ID against the compiled-in key set.
  - Test against signatures produced by the real `minisign` CLI in release CI.
- **age: adopt.** Its passphrase recipient is scrypt-based. Test that restore round-trips a fixture from every released version (ADR 0018).
- **zstd: adopt klauspost.** Go's stdlib has only an internal decoder.
- **BLAKE3: adopt `zeebo/blake3`, pinned to a pseudo-version.** Transfers run on arm64 NAS and VPS Workers, where lukechampine's pure-Go path is the slower one; the throughput gap is *inferred, not benchmarked*. If tagged releases are required, use lukechampine and benchmark chunk hashing on arm64 first.

**Criticality:**
- age, zstd and BLAKE3: critical infra (Backups and Transfer integrity).
- minisign: critical (update security).

---

## ADR conflicts
- None of the recommendations reopens an ADR.
- Flagged, not reopened: TCP-multiplexed UDP forwarding and the per-stream window cap under ADR 0013 and ADR 0012 (section 2); QUIC would reopen ADR 0013.
- goupnp and go-nat-pmp hardcode their transport (a global HTTP client, fixed SSDP and port 5351), which sits poorly with ADR 0014. That is why section 7 says build.
- GoReleaser Pro is closed-source. That isn't a GPL (ADR 0010) violation, since it's a build tool, but CI would depend on a paid licence key.

## Open questions for the maintainer
1. Should the Join Code be a pasted high-entropy code that embeds the Controller's public key and a 128-bit secret, with no PAKE? **Recommend yes.** Choose no only if you want codes short enough to type, which needs a PAKE whose best Go option is experimental.
2. For yamux: hashicorp/yamux with a raised fixed `MaxStreamWindowSize`, or libp2p/go-yamux with RTT auto-tuning? **Recommend hashicorp now, behind an interface.** Switch if relay benchmarks at 100–300 ms RTT show the cap.
3. Do UDP forwards through the Relay ship as explicitly best-effort over the TCP link? **Recommend yes**, and document it in S52.
4. Must the relay send each forwarded client's original address to the Serving Node, so per-IP limits and logs see real clients? **Recommend yes.** It is required for ADR 0005's per-IP limiting to mean anything remotely.
5. Should the PIN be hashed as HMAC-with-a-key-file followed by argon2id (a pepper from the P-350 key file), rather than argon2id alone? **Recommend yes.**
6. Sessions: a custom sqlc table or alexedwards/scs? **Recommend custom.**
7. Release pipeline: a custom `cmd/release` with nfpm, WiX and appimagetool, GoReleaser OSS, or GoReleaser Pro? **Recommend the custom one plus nfpm.**
8. On macOS, should the updater swap the whole `.app` bundle, or the binary inside it and then re-sign ad hoc? **Recommend swapping the whole bundle.** Prove it with a native smoke test in S56 either way.
9. BLAKE3: zeebo/blake3 at an untagged pseudo-version (arm64 SIMD), or lukechampine v1.4.1 (tagged, amd64 SIMD only)? **Recommend zeebo**, after a quick arm64 benchmark.
10. minisign: aead.dev/minisign, or a roughly 120-line in-tree verifier for the frozen update path? **Recommend aead.dev/minisign**, either choice is sound.
