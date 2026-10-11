# How Media Servers classify viewers through a relay

Research for [#32](https://github.com/jatassi/SlipStream/issues/32), following the open questions in [CGNAT, remote access and relay techniques](https://github.com/jatassi/SlipStream/issues/15). Labels: **observed** (seen on the chef's live setup), **measured**, **code** (read in source), **doc**, **user report**, **inferred**.

## Summary

- Through the chef's current relay, Plex classes remote viewers as **LAN**. **Observed** by the chef on the Plex dashboard, 2026-10-11.
- The likely cause is that the relay's tunnel address sits on a subnet the Plex host owns, and Plex counts the server's own subnets as local when "LAN Networks" is blank. **Inferred.**
- Jellyfin would do the same with this relay. Every RFC1918 address is local by default, and forwarded headers are ignored until the relay is listed in `KnownProxies`. **Code.**
- A relay that makes remote viewers local skips Plex's remote-playback paywall and Jellyfin's remote limits. That is the terms risk #15 inferred, now confirmed as the default outcome rather than an edge case.
- Userspace WireGuard (`wireguard-go` with `tun/netstack`) moved 645 to 966 Mbit/s on Apple M4 loopback, 4.5x or more over a 144 Mbit/s UHD Blu-ray peak. **Measured.** The chef's kernel WireGuard relay already carries 4K remuxes. **Observed.**

## The live relay

Read-only survey of the VPS (`5.161.67.8`, Ubuntu 24.04, 2 vCPU, 2 GB) on 2026-10-11:

- Kernel WireGuard `wg0` at `10.8.0.1/24`, listening on UDP 51820. The home peer is `10.8.0.2`, a Windows host, about 18 ms away. Lifetime counters show 656 GB received from home.
- Caddy terminates TLS for `plex.atassi.org` (certificate by Cloudflare DNS-01) and runs `reverse_proxy 10.8.0.2:32400`.
- So Plex receives every remote viewer from TCP source `10.8.0.1`, with Caddy's `X-Forwarded-For` holding the viewer's public IP. Caddy overwrites incoming `X-Forwarded-For` but passes `X-Real-IP` through from the client untouched ([Caddy reverse_proxy defaults](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy#defaults)).
- From a machine off the Plex LAN, `https://plex.atassi.org/identity` answered 200 with `claimed="1"`, version `1.43.4.10903`.

## Plex

Closed source. The rule is documented only partly.

- With "LAN Networks" blank, "only the server's subnet is considered to be on the local network" ([Plex: Network settings](https://support.plex.tv/articles/200430283-network/); in-app text quoted by a moderator in [this thread](https://forums.plex.tv/t/cross-subnet-server-access/729086)).
- The Plex host's WireGuard adapter is on `10.8.0.0/24`, so `10.8.0.1` is on one of the server's subnets. **Inferred** as the reason the chef sees LAN. If the chef's LAN Networks lists `10.8.0.0/24` explicitly, the result is the same.
- Plex has no documented trusted-proxy setting. User reports say the log shows "Using X-Forwarded-For: … as remote address" ([forum](https://forums.plex.tv/t/native-support-for-x-forwarded-for-proxy-protocol-for-client-ip-detection/944016)) and that private values are ignored ([forum](https://forums.plex.tv/t/x-forwarded-for-trust-configuration/889105)). The chef's observation shows that, whatever Plex does with the header here, the public XFF value does not make the session remote. **Observed.**
- Since 2025-04-29, "remote" for the paywall means the player "cannot make a connection to the Plex Media Server on the same subnet of the same local network" ([Plex: Requirements for Remote Playback](https://support.plex.tv/articles/requirements-for-remote-playback-of-personal-media/)).
- The session API exposes the verdict: `Session.location` (`lan`/`wan`), `Player.local`, `Player.relayed`, `Player.secure`, `Player.address` ([developer.plex.tv/pms](https://developer.plex.tv/pms/)). SlipStream can read these to check its own relay.
- One user-reported workaround keeps a proxy host remote with a LAN Networks netmask that excludes it ([forum](https://forums.plex.tv/t/possible-to-treat-local-ip-as-remote/612090)). Untested.

## Jellyfin

Read in source at [`a45d1415`](https://github.com/jellyfin/jellyfin/tree/a45d1415d4da7c9b69d6e7a6895fc63d2d68e8f1). **Code.**

- Forwarded headers are off while `KnownProxies` is empty, the default ([`ApiServiceCollectionExtensions.cs#L166-L188`](https://github.com/jellyfin/jellyfin/blob/a45d1415d4da7c9b69d6e7a6895fc63d2d68e8f1/Jellyfin.Server/Extensions/ApiServiceCollectionExtensions.cs#L166-L188)). With entries, it reads `X-Forwarded-For`, `-Proto` and `-Host` only, never `X-Real-IP` or `Forwarded`.
- ASP.NET checks only that the peer is a known proxy, not whether the forwarded value is public, so a private XFF from a known proxy counts as local.
- With `LocalNetworkSubnets` blank, local means loopback, 10/8, 172.16/12, 192.168/16, fc00::/7 and fec0::/10 ([`NetworkManager.cs#L309-L350`](https://github.com/jellyfin/jellyfin/blob/a45d1415d4da7c9b69d6e7a6895fc63d2d68e8f1/src/Jellyfin.Networking/Manager/NetworkManager.cs#L309-L350), [`NetworkConstants.cs#L48-L58`](https://github.com/jellyfin/jellyfin/blob/a45d1415d4da7c9b69d6e7a6895fc63d2d68e8f1/MediaBrowser.Common/Net/NetworkConstants.cs#L48-L58)). CGNAT 100.64/10 is not local.
- Local skips `RemoteClientBitrateLimit` ([`MediaInfoHelper.cs#L518-L537`](https://github.com/jellyfin/jellyfin/blob/a45d1415d4da7c9b69d6e7a6895fc63d2d68e8f1/Jellyfin.Api/Helpers/MediaInfoHelper.cs#L518-L537)), the per-user remote-access permission ([`UserManager.cs#L633-L641`](https://github.com/jellyfin/jellyfin/blob/a45d1415d4da7c9b69d6e7a6895fc63d2d68e8f1/Jellyfin.Server.Implementations/Users/UserManager.cs#L633-L641)), `EnableRemoteAccess` and `RemoteIPFilter` ([`NetworkManager.cs#L731-L768`](https://github.com/jellyfin/jellyfin/blob/a45d1415d4da7c9b69d6e7a6895fc63d2d68e8f1/src/Jellyfin.Networking/Manager/NetworkManager.cs#L731-L768)).
- So a relay at `10.8.0.1` or a LAN dial makes every viewer local. Listing the relay's single address in `KnownProxies` and sending `X-Forwarded-For` classifies them correctly. Jellyfin has no paywall, so the cost is lost admin controls, not terms.

## Emby

Docs and forum only.

- With LAN Networks blank, "only the server's subnet and common private IP subnets" are local, with 10.x not guaranteed ([Emby: Hosting Settings](https://emby.media/support/articles/Hosting-Settings.html)).
- Emby reads `X-Real-IP` or `X-Forwarded-For` from any peer by default, with no trusted-proxy list (same page). CVE-2023-33193 let spoofed headers flip local detection ([advisory](https://github.com/EmbySupport/security/security/advisories/GHSA-fffj-6fr6-3fgf)); later releases ignore private values in these headers ([forum](https://emby.media/community/topic/119706-x-forwarded-for-for-local-addresses)).
- A relay must therefore set `X-Real-IP` itself. Caddy passes the client's value through unless told otherwise. **Inferred.**

## Userspace WireGuard throughput

`wgbench/main.go` beside this file builds two netstack peers over real UDP on 127.0.0.1, MTU 1420, pushes TCP through the tunnel for 10 s, and checks bytes received against bytes sent. Three interleaved runs per config on an Apple M4. **Measured.**

| Config | Median Mbit/s | Range | CPU-s per GB, both peers |
|---|---|---|---|
| Plain TCP, GOMAXPROCS=10 | 80774 | 75161 to 82489 | 0.21 |
| Netstack WireGuard, GOMAXPROCS=10 | 964 | 932 to 966 | 57.04 |
| Netstack WireGuard, GOMAXPROCS=2 | 672 | 645 to 710 | 22.86 |
| Netstack WireGuard, GOMAXPROCS=1 | 729 | 726 to 738 | 10.89 |

- The limit is the sending peer's single `RoutineSequentialSender` goroutine, at 93% of a core. It issues one `sendmsg` per packet on macOS, about 10.6 µs each, which caps it near 1.07 Gbit/s. ChaCha20 is 1 to 2% of CPU. **Measured** by pprof with per-peer labels.
- High GOMAXPROCS burns CPU in the scheduler. The work itself is about 11 CPU-s per GB.
- A 144 Mbit/s stream at that cost needs about a quarter of an M4 core on the sending peer. On Linux, `wireguard-go` batches UDP sends, so per-packet cost should fall. **Inferred.**
- gVisor's TCP receive buffer grows to 4 MB, capping one stream near 4 MB per RTT. On the chef's 18 ms VPS-to-home link that is about 1.8 Gbit/s. Viewer latency does not matter, because the viewer's TCP ends at the relay. **Inferred.**
- Not measured: netstack across the internet. The chef's kernel WireGuard relay over the same path already streams 4K remuxes. **Observed.**

Rerun with `go run . loopback -reps 3 -dur 10s -procs 10,2,1`. The same binary has `server` and `client` modes for an internet run.

## What this means for SlipStream's relay

- In the netstack design from #15, the home Node dials the Media Server over the LAN, so the source is a LAN address and both Plex and Jellyfin see every relayed viewer as local by default. The chef's current setup already shows that outcome for Plex.
- Jellyfin can be fixed by SlipStream: list its own address as a Known Proxy and send `X-Forwarded-For`.
- Plex has no supported knob that keeps a proxied viewer remote while the proxy sits on the server's subnet. Options are the unverified LAN Networks netmask trick, dialing Plex from an address outside its subnets, or accepting local classification.
- SlipStream can verify its outcome at runtime through `Session.location` and `Player.local`, which fits the reachability diagnostic.
