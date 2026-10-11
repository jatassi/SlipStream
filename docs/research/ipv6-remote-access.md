# IPv6 remote access facts

Research for [#114](https://github.com/jatassi/SlipStream/issues/114). It feeds [IPv6 reachability verdicts (#115)](https://github.com/jatassi/SlipStream/issues/115), which settles how the reachability diagnostic judges each service over IPv6. The IPv4 side is in [CGNAT, remote access and relay techniques (#15)](https://github.com/jatassi/SlipStream/issues/15) ([findings](https://github.com/jatassi/SlipStream/blob/research/cgnat-relay/docs/research/cgnat-relay.md)), [Relay Role and reachability diagnostic (#16)](https://github.com/jatassi/SlipStream/issues/16) and ADR 0012. Terms follow `GLOSSARY.md`.

Sources were read, and code and stats measured, on 2026-10-11. Each claim carries a tag:

- **[S]** sourced. The claim comes from the linked primary source: vendor docs, source code, release notes, an RFC, a staff post or a measurement owner's publication.
- **[M]** measured. I read it in code at a stated commit, ran it, or read a number off a live stats file.
- **[I]** inferred. This is reasoning from the sourced and measured facts, not something a source states.

Forum posts by users, not staff, are marked *user report*. They are pointers, not evidence. This file lays out facts and tradeoffs. It does not pick verdicts; that is #115's job.

## Answer in brief

1. **Media Servers.**
   - **Plex** gained real IPv6 support only in PMS 1.43.3 (beta 2026-06-22, public 2026-08-18). The release notes call it "TEMPORARILY" a Plex Pass feature, so free servers don't get it. The server publishes its IPv6 address to plex.tv as a connection, with the server's own port. Plex's published connect-back workers include two IPv6 addresses, so the check can test IPv6, though nothing primary describes how. Staff say dual-stack "should" work. User reports say a CGNAT server shows reachable only with "IPv6 only" selected. Staff call the indicator "flaky".
   - **Jellyfin** 12.2 and 10.11 default `EnableIPv6` to false. When IPv6 is on alongside IPv4, Jellyfin binds dual-stack `[::]`. It has no cloud service, so it advertises nothing off the LAN.
   - **Emby** 4.10 listens on IPv6 out of the box. It detects its WAN address over IPv4 unless a hidden setting says otherwise, and Emby Connect hands clients that WAN address.
2. **Clients.**
   - **Plex apps ignore IPv6 connections** unless the user turns on a per-app "Include IP V6 Connections" toggle. plex.tv returns IPv6 connections only when the client asks with `includeIPv6=1`.
   - **Roku channels can't make IPv6 requests at all**, according to Roku staff. That rules out Jellyfin for Roku, and very likely the Plex and Emby Roku apps.
   - **Emby's Android TV app deliberately prefers IPv4.**
   - **Clients that connect by hostname** (Jellyfin web, Swiftfin, the Android SDK) leave the choice to the platform's Happy Eyeballs, which prefers IPv6 and falls back to IPv4.
3. **Pinholes.**
   - **Two protocols can open one:** PCP MAP in firewall mode, and UPnP IGDv2 `WANIPv6FirewallControl`. Both open a pinhole only for the request's own source address. The OS picks a temporary address as source by default, so a Node must send the request from its stable address.
   - **Router support is patchy.**
     - FRITZ!Box documents both protocols.
     - OpenWrt ships miniupnpd uninstalled. When it is installed, it hides the UPnP service, which leaves PCP.
     - pfSense has PCP but no IGDv2. OPNsense has neither for IPv6.
     - Google Nest, eero and Xfinity are configured by hand, or not documented.
     - Starlink's router supports no IPv6 firewall rules at all.
     - No survey of router support exists.
   - **Default firewall:** the RFC 6092 recommendation is to block unsolicited inbound traffic, but it doesn't make that the default. OpenWrt and FRITZ!Box do block it. A CCS 2026 paper found 14.9 million residential IPv6 hosts reachable anyway.
4. **Address discovery.**
   - **Listing interfaces** finds the address. Go's `IsGlobalUnicast` accepts ULAs, so `IsPrivate` must also be checked. Go's standard library can't tell a temporary address from a stable one, so that check is per OS.
   - **STUN over IPv6** echoes the source address, which is usually temporary. It differs from the interface address only under NPTv6 or NAT66, or in a Docker container.
   - **Delegated prefixes** can change on every router reboot, and daily at some German ISPs.
   - **Only a connect-back from outside proves reachability.** Ping proves nothing: OpenWrt forwards ICMPv6 echo while it drops TCP.
5. **Prevalence.**
   - **CGNAT on IPv4 with native IPv6 is a common pattern.** In an August 2026 Akamai study, about half of the traffic from massively shared IPv4 addresses came from blocks that also had IPv6, and 52% of the top providers studied ran shared IPv4 alongside unshared IPv6. Starlink and DS-Lite lines are documented examples.
   - **About 56% of users worldwide are not IPv6-capable** (APNIC). Google sees 48.7% of its users arriving over IPv6.
   - **Viewers on mobile data mostly have IPv6** (87–96% on the big US carriers).
   - **Some large fixed ISPs are nearly IPv4-only:** Virgin Media, TalkTalk, Plusnet, Frontier, CenturyLink, Optimum and Viasat.
   - **Hotel and office networks lag**, but only proxy figures exist.
6. **Relay over IPv6.**
   - **The Relay Node could dial home over IPv6.** Every VPS provider checked offers IPv6, free. Dialing home needs a pinhole and an address the relay keeps current. It also exposes the home Node.
   - **The latency path is the same either way.** The only gain is escaping head-of-line blocking on the single TCP Node link. **[I]**
   - **The more useful IPv6 fact is the reverse direction.** The home Node can dial the Node link to the relay over IPv6 with no pinhole. That bypasses the CGN, DS-Lite or 464XLAT state, along with their idle timeouts. **[I]**
   - **IPv6 never removes the relay for IPv4-only viewers.** It does let IPv6 viewers skip the relay. **[I]**

## 1. Do the Media Servers listen on and advertise IPv6?

Versions at the time of writing:

- **Jellyfin 12 exists.** v12.0 shipped 2026-09-08 and v12.2 on 2026-10-05. The 10.11 line is at v10.11.11. **[M]** (`gh api repos/jellyfin/jellyfin/releases`)
- **Emby Server:** the stable release is 4.10.1.0 (2026-09-29). **[M]** (`gh api repos/MediaBrowser/Emby.Releases/releases`)
- **Plex Media Server:** the public release is 1.43.4.10903 and the beta is 1.43.5.11029 ([release notes thread](https://forums.plex.tv/t/plex-media-server/30447/718)). **[S]**

### Plex

- **IPv6 is new and gated.**
  - PMS 1.43.3.10768 beta notes, posted 2026-06-22 by PlexReleaser (Plex Employee): "(Network) IPv6 support has TEMPORARILY been made a plex pass feature (PM-5140)" and "Support for ipv6 only networks (PM-5140)". [Release notes](https://forums.plex.tv/t/plex-media-server/30447/708). **[S]**, verified by fetching the post and its author.
  - That build went public on 2026-08-18 ([post 714](https://forums.plex.tv/t/plex-media-server/30447/714)). **[S]**
  - Plex's Lifetime Plex Pass post listed "IPv6 Support" as a commitment ([2026-05-18](https://forums.plex.tv/t/new-lifetime-plex-pass-pricing/938910/1)). **[S]**
  - **Why the gate:** Atomatth (Plex Employee) said local IPv6 addresses were being misclassified as remote, which would trip the paid Remote Watch gate for free users ([2026-06-22](https://forums.plex.tv/t/ipv6-support-in-plex-server/939875/14)). **[S]**
- **The setting changed form.**
  - 1.41.9 replaced the "enable ipv6 preference with client network" (PM-2912, [2025-06-26](https://forums.plex.tv/t/plex-media-server/30447/681)). **[S]**
  - Staff describe the new setting, Settings > Network > Client Network, as offering IPv4 only, IPv6 only, or both ([Atomatth, 2026-06-22](https://forums.plex.tv/t/ipv6-support-in-plex-server/939875/6)). **[S]**
  - Plex's support articles are stale.
    - The Network article still lists "Enable server support for IPv6" ([support](https://support.plex.tv/articles/200430283-network/)). **[M]**
    - The hidden-settings article lists `EnableIPv6`, default `0` ([support](https://support.plex.tv/articles/201105343-advanced-hidden-server-settings/), modified 2026-09-30). **[M]**
  - No primary source gives the default of Client Network.
- **Binding.** No primary source says whether PMS binds `[::]` dual-stack.
  - Old release notes imply the setting changes socket setup. One says "Fallback to IPv4 only if IPv6 networking fails to setup" ([2017](https://forums.plex.tv/t/plex-media-server/30447/178)). Another, from FreeBSD, says "Enabling IPv6 support could prevent IPv4 connections from being accepted" ([2018](https://forums.plex.tv/t/plex-media-server/30447/241)). **[S]**
  - The FreeBSD symptom fits a v6 socket with that OS's default `IPV6_V6ONLY=1`. **[I]**
- **Advertising.**
  - Plex's API docs tell clients to fetch `https://clients.plex.tv/api/v2/resources?includeHttps=1&includeRelay=1&includeIPv6=1`. They also say a `plex.direct` host label can encode "IPv4 or IPv6 address components" ([developer.plex.tv/pms](https://developer.plex.tv/pms/)). **[M]**
  - python-plexapi sends `includeIPv6=1`, reads a boolean `IPv6` attribute on each `<connection>`, and tries IPv4 before IPv6 (`DEFAULT_IP_ORDER = ['ipv4','ipv6']`) ([myplex.py](https://github.com/pushingkarmaorg/python-plexapi/blob/8a9ade7f364582cabb6d5bc200994fddb2113d48/plexapi/myplex.py#L1456-L1633), [PR #1627](https://github.com/pushingkarmaorg/python-plexapi/pull/1627), merged 2026-07-10). **[M]**
  - So plex.tv publishes IPv6 connections, and a client gets them only if it asks. **[I]**
  - `plex.direct` resolves the full eight-group dashed form of an IPv6 address to an AAAA record. The `--` compressed form returns nothing. **[M]** (`dig`)
- **Ports.** PMS 1.43.5 beta (2026-10-06) does three things:
  - It ignores the manually specified public port for IPv6 connections.
  - It sends subnet information to plex.tv "so IPv6 and non-private local connections are detected correctly" (PM-5364).
  - It adds IPv6 to the preferred network interface (PM-6017).

  [Release notes](https://forums.plex.tv/t/plex-media-server/30447/718). **[S]** Plex Web 4.162.1 matches: "a manually specified public port applies to IPv4 addresses only", and it fixed "servers reachable only over IPv6 being treated as remote instead of nearby" ([2026-09-30](https://forums.plex.tv/t/plex-web/20528/471)). **[S]** So an IPv6 connection is published with the server's own listening port, and only a pinhole is needed, not a port remap. **[I]**
- **The connect-back check over IPv6.** No primary source describes how plex.tv's reachability worker treats IPv6, but the evidence points to it testing IPv6.
  - Plex publishes its worker source addresses ([Troubleshooting Remote Access](https://support.plex.tv/articles/200931138-troubleshooting-remote-access/)). Today the [list](https://s3-eu-west-1.amazonaws.com/plex-sidekiq-servers-list/sidekiqIPs.txt) holds eight IPv4 and two IPv6 addresses in `2600:1f18:5538:4808::/63`. **[M]** So the workers can connect back over IPv6. **[I]** A pinhole that admits only Plex's check must allow those two addresses.
  - In dual-stack mode "both should be supported", according to Atomatth (Plex Employee, [2026-06-22](https://forums.plex.tv/t/ipv6-support-in-plex-server/939875/10)). **[S]**
  - *User report:* "finally the connectivity check also works over IPv6" ([certuna, 2026-07-16](https://forums.plex.tv/t/bypass-cg-nat-via-ipv6-with-newest-beta-release/940261/23)). Before that, it had only ever tested IPv4 ([/940261/5](https://forums.plex.tv/t/bypass-cg-nat-via-ipv6-with-newest-beta-release/940261/5)).
  - Atomatth (Plex Employee) also told users to set IPv6 and "Double check the Remote Access page to see your IPv6 address" ([2026-06-22](https://forums.plex.tv/t/ipv6-support-in-plex-server/939875/8)). **[S]**
  - Staff call the Remote Access indicator "known to be flaky" ([Atomatth, 2026-08-31](https://forums.plex.tv/t/remote-access-again-seems-to-be-problematic/942115/105)) and say to "Ignore the indicator" behind reverse proxies ([chris_decker08, 2026-09-10](https://forums.plex.tv/t/remote-access-again-seems-to-be-problematic/942115/214)). **[S]**
  - *User reports* (pointers only):
    - In dual-stack mode the Remote Access page shows only IPv4. "IPv6 only" shows the IPv6 address and a green check ([939875/21–22](https://forums.plex.tv/t/ipv6-support-in-plex-server/939875/21), [/35](https://forums.plex.tv/t/ipv6-support-in-plex-server/939875/35)).
    - Behind CGNAT IPv4, dual-stack falls back to Plex Relay, and only "IPv6 only" connects directly ([940261/24](https://forums.plex.tv/t/bypass-cg-nat-via-ipv6-with-newest-beta-release/940261/24)).
    - On 1.43.4 the tester probes the ULA instead of the global address ([943544](https://forums.plex.tv/t/server-tests-ula-instead-of-gua-address/943544/1)).
  - **So for a CGNAT server with an open IPv6 pinhole:** Plex plausibly reports reachable in "IPv6 only" mode. In dual-stack mode, staff say it should work, and users say it doesn't. No primary source confirms either. **[I]**
- **A related breakage.** plex.tv work done for IPv6 support broke publishing when more than one server sat on the same network, and was reverted ([Atomatth, 2026-08-31](https://forums.plex.tv/t/remote-access-again-seems-to-be-problematic/942115/89)). **[S]**
- **Staff on CGNAT:** "If you ISP, server, and all clients support it you can set up IPv6. IPv6 support isn't perfect yet" ([chris_decker08, 2026-09-09](https://forums.plex.tv/t/remote-access-again-seems-to-be-problematic/942115/193)). **[S]**

### Jellyfin

- **Settings.** `EnableIPv4` defaults to true and `EnableIPv6` to false. This is the same at v12.2 and v10.11.8 ([`NetworkConfiguration.cs`](https://github.com/jellyfin/jellyfin/blob/v12.2/MediaBrowser.Common/Net/NetworkConfiguration.cs#L124)). **[M]** The dashboard shows it as an "Enable IPv6" checkbox. **[M]** (jellyfin-web, `src/apps/dashboard/routes/networking/index.tsx`)
- **Binding.** `GetAllBindInterfaces` in `src/Jellyfin.Networking/Manager/NetworkManager.cs` at v12.2 decides what Jellyfin binds. **[M]**
  - IPv4 and IPv6 both on: it binds `IPv6Any`, which Kestrel opens dual-mode.
  - IPv4 only: it binds `0.0.0.0`.
  - IPv6 only: it binds each IPv6 interface address, because `IPv6Any` would bind IPv4 too.
  - If the OS lacks IPv6, it logs a warning and skips IPv6.
- **Outbound and discovery.**
  - Outbound Happy Eyeballs uses IPv6 only when `EnableIPv6` is on (`NetworkManager.cs:649`). **[M]**
  - LAN auto-discovery on UDP 7359 binds IPv4 `Any` only (`AutoDiscoveryHost.cs:60`). **[M]**
- **No advertising.**
  - Jellyfin has no cloud or relay service, and UPnP is `[Obsolete("No longer supported")]`. **[M]**
  - Remote reachability is the admin's DNS. The docs say only to "publish an IPv6 DNS record" ([Jellyfin: Networking](https://jellyfin.org/docs/general/post-install/networking/)). **[S]**

### Emby

- **Listening.** Luke (Emby lead developer) said IPv6 access "should work out of the box" ([2022-07-23](https://emby.media/community/topic/110744-emby-server-over-ipv6/)). **[S]** No primary source describes the socket setup.
- **WAN address detection.**
  - Luke: "We use ipify… Currently we use the ipv4 url." Since 4.8.0.45, a hidden `system.xml` setting, `PreferredDetectedRemoteAddressFamily=InterNetworkV6`, switches to ipify's dual-stack URL.
  - Luke added in 2025 that this is "not a guarantee" and that admins should set the remote address by hand ([topic 114025](https://emby.media/community/topic/114025-force-ipv6/)). **[S]**
- **Emby Connect** "will use the address that is shown on your Emby server dashboard" ([ebr, Emby staff, 2023-07-26](https://emby.media/community/topic/120207-getting-ipv6-automatically/)). **[S]** So by default Emby Connect hands out the IPv4 WAN address. **[I]**
- **Settings.** The only documented IPv6 setting is "Network Request Protocol" (Auto or IPv4). It covers outgoing requests only ([Hosting Settings](https://emby.media/support/articles/Hosting-Settings.html)). **[S]**
- **Connect-back.** No primary source describes an Emby reachability test, over IPv4 or IPv6.

## 2. How do client apps treat an IPv6 address?

### Plex

- **The opt-in toggle.**
  - On Android the setting is "Settings > Advanced > Include IP V6 Connections" ([Atomatth, Plex Employee, 2026-06-22](https://forums.plex.tv/t/ipv6-support-in-plex-server/939875/10)). **[S]**
  - Plex HTPC has "Include IPv6 Connections when discovering servers from plex.tv" ([HTPC settings](https://support.plex.tv/articles/htpc-settings/)). **[M]**
  - The Android, Android TV and Fire TV beta of 2026-10-05 fixed "With Include IP V6 Connections on, servers reached over IPv6 could fail to connect" ([notes](https://forums.plex.tv/t/plex-for-mobile-tvs-android-ios-android-tv-tvos-beta-release-notes/912796/100)). **[S]**
  - Together with `includeIPv6=1` being opt-in (section 1), a Plex app ignores the server's IPv6 connections unless the user turns this on. **[I]**
- **Connection order.**
  - python-plexapi tests local, then remote, then relay; https before http; and IPv4 before IPv6. It runs the tests in parallel and takes the first success in that order (`myplex.py`). **[M]**
  - No primary source gives the official apps' order. *User report:* dual-stack Apple clients prefer IPv4 ([943544/2](https://forums.plex.tv/t/server-tests-ula-instead-of-gua-address/943544/2)).
  - Side finding: plexapi builds `http://{address}:{port}` without brackets, so its URL is invalid for an IPv6 literal (`myplex.py:1633`). **[M]**
- **Unknown platforms.** No primary source covers Plex for Roku, Samsung, LG, PlayStation or iOS. *User reports* say PS5 and Samsung Tizen fail ([939875/37](https://forums.plex.tv/t/ipv6-support-in-plex-server/939875/37), [/11](https://forums.plex.tv/t/ipv6-support-in-plex-server/939875/11)).
- **Plex Web** runs on the browser's network stack. **[I]**

### Roku, the platform

- **Roku OS 12 added dual-stack support** "for supported devices" ([release notes](https://support.roku.com/article/roku-os-release-notes)). **[S]**
- **Channels still can't use it.** RokuBen (Roku staff) on the developer forum, 2026-07-06: "we have not enabled general IPv6 support in our client interfaces, so you can't make an API call or stream from an IPv6-only server right now". The internal media player resolves IPv6 hosts named in a DASH manifest, "but roUrlTransfer-based requests won't use it". Prime Video gets IPv6 only through a partner-only NDK ([thread](https://forum.developer.roku.com/t/cant-get-ipv6-requests-to-work/11499)). **[S]**, verified by fetching the posts and their staff flag.
- **Jellyfin for Roku** closed its IPv6 issue as not planned: "we have no method to support IPv6" ([jellyfin-roku#813](https://github.com/jellyfin/jellyfin-roku/issues/813), 2026-07-07). **[S]** Its URL parser rejects IPv6 literals anyway ([`parsedUrl.bs`](https://github.com/jellyfin/jellyfin-roku/blob/72cfe985d4567e40f98dc098490a73eb141de85a/source/utils/parsedUrl.bs#L8)). **[M]**
- **Plex and Emby on Roku** face the same SDK limit unless they hold NDK partner status. **[I]**

### Jellyfin clients

- **The Kotlin SDK** (used by Android and Android TV):
  - It accepts `[::1]` and `[::1]:8096` as server addresses (`DiscoveryServiceTests.kt:45`). **[M]**
  - It uses OkHttp 5.5.0 with the default builder, where `fastFallback` (Happy Eyeballs) defaults to true. **[M]**
  - An open issue reports no fallback to IPv4 when the AAAA record is broken ([jellyfin-android#1601](https://github.com/jellyfin/jellyfin-android/issues/1601), last comment 2026-09-12). **[S, issue]** Whether that still holds on OkHttp 5 is unverified.
- **jellyfin-android** renders jellyfin-web in a WebView. **[M]**
- **jellyfin-androidtv:** an old image-library bug broke IPv6 literals ([#2018](https://github.com/jellyfin/jellyfin-androidtv/issues/2018), 2022). **[S]** The app has since moved to Coil 3 on OkHttp, so the bug is likely gone. **[I]**
- **jellyfin-web** passes the address straight to `fetch` (`connectionManager.js:660–690`). **[M]** The browser handles bracketed literals and Happy Eyeballs. **[I]**
  - The Tizen and webOS clients wrap jellyfin-web, so they inherit the TV's browser engine. **[I]**
  - No primary source covers Samsung's or LG's IPv6 support.
- **Swiftfin** parses addresses with Foundation `URL` (`ConnectToServerViewModel.swift:58–70`). **[M]**
  - On macOS Foundation, `http://[2001:db8::1]:8096` parses. **[M]**
  - A bare `2001:db8::1:8096` fails as an invalid URL. **[M]**

### Emby clients

- **The Android TV app** "is definitely going to prefer v4 because some of the components within it do not fully support v6" ([ebr, Emby staff, 2025-04-25](https://emby.media/community/topic/138393-emby-prefers-ipv4-over-ipv6-after-initial-server-connect/)). **[S]**
- **Its image library throws on an IPv6 literal with a port.** Staff said in 2025 that this won't be fixed without rewriting image handling, and to use the standard Android app ([topic 145002](https://emby.media/community/topic/145002-image-processing-problem-when-using-ipv6-addresses/)). **[S]**
- No primary source covers the iOS, tvOS, Samsung or LG apps.

### The OS mechanism when a client connects by hostname

- **RFC 8305 (Happy Eyeballs v2):** clients race the AAAA and A answers and prefer IPv6, with a 50 ms resolution delay and a 250 ms connection-attempt delay. RFC 6724 ranks a global IPv6 address above IPv4 ([RFC 8305](https://www.rfc-editor.org/rfc/rfc8305), [RFC 6724](https://www.rfc-editor.org/rfc/rfc6724)). **[S]**
- **Chromium** tries IPv6 first and IPv4 after `kIPv6FallbackTime` of 300 ms ([`transport_connect_job.h`](https://github.com/chromium/chromium/blob/main/net/socket/transport_connect_job.h#L130-L133)). **[M]**
- **Apple's** Network.framework and URLSession race the two ([WWDC18 session 715](https://developer.apple.com/videos/play/wwdc2018/715/)). **[S]**
- **Android** has no platform-wide Happy Eyeballs for plain sockets, so it depends on the HTTP library. **[I]** No Android doc was found.
- **Implications for SlipStream.** **[I]**
  - **A hostname with both A and AAAA records is safe for every client.** Browsers, Apple clients and OkHttp clients use IPv6 when they can and fall back otherwise. Roku and IPv4-only networks use the A record.
  - **An AAAA-only hostname fails** on Roku, on Emby Android TV, and for every viewer without IPv6.
  - **Plex differs.** Its apps find servers through plex.tv's connection list, so the per-app opt-in matters more than DNS.

## 3. How does a home Node open an inbound IPv6 pinhole?

### The protocols

- **PCP (RFC 6887).**
  - A MAP request to a pure IPv6 firewall is "purely a firewall" function (§10, §11.1). The request goes over UDP to port 5351 on the default router or a configured server (§8.1).
  - Protocol 0 with port 0 asks for every port, like a DMZ host (§11.3), and a lifetime of 0 deletes the mapping.
  - The server picks the lifetime. The minimum SHOULD be 120 s, the maximum is 24 h, and the client renews at between 1/2 and 5/8 of the lifetime (§15, §8.1.1).
  - The client-IP field must equal the packet's source address, or the server answers ADDRESS_MISMATCH (§8.2).
  - [RFC 6887](https://www.rfc-editor.org/rfc/rfc6887). **[S]**
- **NAT-PMP is IPv4 only.** RFC 6886 lists IPv6 as what PCP adds (§1.1). **[S]** ([RFC 6886](https://www.rfc-editor.org/rfc/rfc6886))
- **UPnP IGDv2 `WANIPv6FirewallControl:1`** (UPnP Forum, 2010-12-10; [spec](https://openconnectivity.org/wp-content/uploads/2015/11/UPnP-gw-WANIPv6FirewallControl-Service.pdf)). **[S]**
  - **Actions:** GetFirewallStatus, AddPinhole, UpdatePinhole, DeletePinhole and GetPinholePackets are required. CheckPinholeWorking is optional.
  - **Lease time:** 1 to 86400 s.
  - **Status flags:** `FirewallEnabled=0` means everything passes. `InboundPinholeAllowed=0` refuses new pinholes and deletes existing ones.
  - **Default policy for unauthenticated control points:** InternalPort must be ≥1024, and InternalClient must equal the requester's own address.
  - **Limits:** a control point cannot list pinholes. CheckPinholeWorking only reports whether traffic was seen, and returns 709 NoTrafficReceived otherwise.
- **Both protocols bind the pinhole to the requester's source address.** A Node must send the PCP or SOAP request from the stable global address it wants opened. Under RFC 6724 Rule 7 the OS would otherwise pick the temporary address (section 4). **[I]**, from the sourced rules above.

### Router and firmware support

No survey of PCP-for-IPv6 or `WANIPv6FirewallControl` deployment was found. **[S, absence]** What vendors document or ship:

- **miniupnpd** (v2.3.11, [miniupnp@f8c4f4c](https://github.com/miniupnp/miniupnp)) is the daemon most open firmware uses. **[M]**
  - **Build:** `WANIPv6FirewallControl` is compiled in only with both `--igd2` and `--ipv6`. PCP is always compiled in.
  - **PCP:** a PCP MAP for a non-v4-mapped address becomes a firewall pinhole (`pcpserver.c`: `is_fw = af == AF_INET6`).
  - **Status is unreliable:** GetFirewallStatus always reports `FirewallEnabled=1` and `InboundPinholeAllowed=1`, because nothing in the tree ever sets those flags (`upnpsoap.c`, `upnpglobalvars.h`). Its answer says nothing about the router's real firewall.
  - **The service can be hidden:**
    - `force_igd_desc_v1` removes the `WANIPv6FirewallControl` service from the device description (`upnpdescgen.c`).
    - So does a User-Agent containing "microsoft" (`upnphttp.c`).
- **OpenWrt** ([openwrt/packages@6fb711f](https://github.com/openwrt/packages/tree/master/net/miniupnpd)). **[M]**
  - miniupnpd is built with `--igd2` and with `--ipv6` when IPv6 is enabled.
  - It isn't installed by default. When it is installed, the shipped `upnpd.config` has `enabled 0` and `igdv1 1`.
  - The init script maps `igdv1` to `force_igd_desc_v1`, which hides the UPnP IPv6 service.
  - So on stock OpenWrt, PCP is the working path once miniupnpd is installed and enabled. **[I]**
- **pfSense:**
  - The build sets miniupnpd's `IPV6` option but not `UPNP_IGDV2` ([make.conf](https://github.com/pfsense/pfsense/blob/master/tools/conf/pfPorts/make.conf)). **[M]** So it has no UPnP IPv6 service, and PCP IPv6 pinholes are compiled in. **[I]**
  - Netgate's docs say "IPv6-capable client implementations are still rare" ([docs](https://docs.netgate.com/pfsense/en/latest/services/configuring-upnp-and-nat-pmp.html)). **[S]**
- **OPNsense 26.7** builds miniupnpd with `UPNP_IGDV2` and without `IPV6`, so it offers no IPv6 pinholes by either protocol ([make.conf](https://github.com/opnsense/tools/blob/master/config/26.7/make.conf)). **[M]**
- **AVM FRITZ!Box:**
  - AVM's IGD document lists `WANIPv6FirewallControl:1` with GetFirewallStatus, AddPinhole, UpdatePinhole and DeletePinhole. It also has AVM extensions that return the IPv6 prefix and its lifetimes ([IGD_v2.pdf](https://fritz.support/resources/IGD_v2.pdf)). **[S]**
  - The knowledge base says "By default, the FRITZ!Box's firewall blocks all incoming connections from the internet". Apps can open ports through UPnP IGD or PCP per RFC 6887, but only once "Permit independent port sharing" is ticked for that device ([AVM KB 894](https://uk.fritz.com/service/knowledge-base/dok/FRITZ-Box-7682/894_Setting-up-automatic-port-sharing-in-the-FRITZ-Box/)). **[S]**
- **Google Nest Wifi** has manual "port opening (for IPv6)" ([help](https://support.google.com/googlenest/answer/6274503)). **[S]** The help also mentions automatic setup over UPnP, but doesn't say which protocol carries IPv6.
- **eero** lists "IPv6 Firewall Rules" in a support article; the page failed to load, so this rests on a search snippet ([article](https://eero.com/support/hc/en-us/articles/207908443)). **[S, snippet]**
- **Xfinity gateways:** "Port forwarding only works for devices using IPv4 addresses and DHCP" ([Xfinity](https://www.xfinity.com/support/articles/port-forwarding-xfinity-wireless-gateway)). **[S]** No official doc covers opening IPv6.
- **Starlink routers** "do not support port forwarding or firewall rules for IPv4 or IPv6", and Starlink recommends a third-party router for inbound traffic ([Starlink: What IP address does Starlink provide?](https://starlink.com/support/article/1192f3ef-2a17-31d9-261a-a59d215629f4)). **[S]**, confirmed through the official page's search snippet because the page renders by JavaScript. Whether Starlink's router blocks or passes unsolicited inbound IPv6 by default is not documented.
- **No primary source** was found for UniFi, AT&T residential gateways, Apple, Netgear or ASUS.

### Client libraries (Go)

- **huin/goupnp** has a generated `WANIPv6FirewallControl1` client. Its SSDP discovery is IPv4-only (`ssdpUDP4Addr = "239.255.255.250:1900"`) ([goupnp](https://github.com/huin/goupnp/blob/main/dcps/internetgateway2/internetgateway2.go)). **[M]**
- **Tailscale's portmapper refuses IPv6 gateways.** It returns `ErrGatewayIPv6`, "no IPv6 support for portmapping" (`net/portmapper/portmapper.go:564`, [tailscale@3bb8a90](https://github.com/tailscale/tailscale)). So the portmapper that the [library research](oss-libraries/B-net-security.md) suggests borrowing from has no IPv6 pinhole code. **[M]**
- **Syncthing's `lib/upnp` is a working Go precedent.** **[M]**
  - It discovers IGDv2 over `udp6` at `[FF05::C]:1900` and skips ULAs, with a comment that IsGlobalUnicast allows them.
  - It requests a pinhole for each global address, binding the SOAP request's source to that address.
  - It notes that FRITZ!Box breaks if the IPv4 and IPv6 discoveries run at the same time.
  - It does not support IPv6 UPnP on Windows (golang/go#63529).
- **Bitcoin Core 29.0** replaced libnatpmp with built-in PCP and NAT-PMP that "supports automatic IPv4 port forwarding as well as IPv6 pinholing" ([release notes](https://bitcoincore.org/en/releases/29.0/)). **[S]**
- **No maintained Go PCP client library** was found. **[M]** (GitHub search)

### What the default firewall blocks

- **RFC 6092 (Informational)** ([RFC 6092](https://www.rfc-editor.org/rfc/rfc6092)). **[S]**
  - State is created by traffic from inside, and "All other traffic is expected to be discarded or rejected" (§2.3).
  - REC-34: an unsolicited inbound TCP SYN gets ICMPv6 "administratively prohibited" by default.
  - REC-11 and REC-33: filtering should be endpoint-independent by default.
  - REC-48: the router SHOULD offer apps a protocol to request inbound traffic.
  - REC-49: a "transparent mode" MUST exist and MAY be the default.
- **RFC 7084 S-1** says the router SHOULD support RFC 6092 and "takes no position on whether such functionality is enabled by default" ([RFC 7084](https://www.rfc-editor.org/rfc/rfc7084)). **[S]**
- **RFC 4864** says IPv4 NAT "was not developed as a security mechanism" ([RFC 4864](https://www.rfc-editor.org/rfc/rfc4864)). **[S]**
- **RFC 4890 §4.3.1** lists ICMPv6 types that must pass, including Packet Too Big and echo ([RFC 4890](https://www.rfc-editor.org/rfc/rfc4890)). **[S]**
- **OpenWrt firewall4 defaults:** the wan zone has input REJECT and forward DROP. `Allow-ICMPv6-Forward` forwards echo request and reply to the LAN. So a ping to a LAN host from outside succeeds while TCP to it is dropped. **[M]**
- **FRITZ!Box** blocks all inbound by default (AVM KB 894 above). **[S]**
- **Many gateways don't block in practice.**
  - Rye, Levin and Beverly reached 14,901,892 IPv6 addresses inside residential networks across 4,412 networks. They conclude that NAT has been the de facto firewall ([arXiv 2509.04792](https://arxiv.org/abs/2509.04792), CCS 2026). **[S]**
  - They cite a 2022 study in which most of ten gateways tested had no IPv6 firewall on by default. That is cited secondhand; I did not read the study.
- **Host firewalls also block.** Windows blocks inbound connections until a rule exists, and its rules cover both IPv4 and IPv6. **[S]** ([New-NetFirewallRule](https://learn.microsoft.com/en-us/powershell/module/netsecurity/new-netfirewallrule)) A router pinhole doesn't help until the host firewall also allows the port. **[I]**

## 4. How does a Node learn its global IPv6 address and whether it is reachable?

### Interface addresses

- **The address kinds.** **[S]**
  - SLAAC ([RFC 4862](https://www.rfc-editor.org/rfc/rfc4862)), stable-privacy ([RFC 7217](https://www.rfc-editor.org/rfc/rfc7217)) and DHCPv6 ([RFC 8415](https://www.rfc-editor.org/rfc/rfc8415)) give an interface its stable addresses.
  - Temporary addresses ([RFC 8981](https://www.rfc-editor.org/rfc/rfc8981)) default to a 1-day preferred and 2-day valid lifetime (§3.8). RFC 8981 says the stable address is the one "used to accept incoming connection requests" (§2.2).
- **Go's classifier accepts ULAs.**
  - `netip.Addr.IsGlobalUnicast` "returns true even if ip is in… IPv6 unique local address space" (`go doc net/netip Addr.IsGlobalUnicast`, Go 1.25.5). **[M]**
  - `IsPrivate` covers `fc00::/7`. **[M]**
  - So a Node needs `Is6() && IsGlobalUnicast() && !IsPrivate()`, and should also exclude the documentation, NAT64 and v4-mapped ranges. **[I]**
- **Go's standard library drops the address flags.**
  - `net.Interfaces` and `Addrs` return only `*net.IPNet` ([golang/go#42694](https://github.com/golang/go/issues/42694), open). **[S]**
  - Telling temporary from stable addresses is per-OS work. **[M]**
    - **Linux:** netlink `IFA_F_TEMPORARY` (0x1), `IFA_F_DEPRECATED` (0x20) and `IFA_F_STABLE_PRIVACY` (0x800), from x/sys.
    - **macOS:** `IN6_IFF_TEMPORARY` (0x80) and `IN6_IFF_DEPRECATED` (0x10), read with `SIOCGIFAFLAG_IN6` ([xnu `in6_var.h`](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/netinet6/in6_var.h)). x/sys doesn't define them, so this needs a raw ioctl.
    - **Windows:** `SuffixOrigin == IpSuffixOriginRandom` marks an anonymous address, and lifetimes are exposed too ([MIB_UNICASTIPADDRESS_ROW](https://learn.microsoft.com/en-us/windows/win32/api/netioapi/ns-netioapi-mib_unicastipaddress_row)). **[S]**
- **Defaults.**
  - **Linux kernel:** `use_tempaddr=0` for most devices, `temp_prefered_lft=86400` and `temp_valid_lft=172800`, with EUI-64 addresses by default ([ip-sysctl](https://docs.kernel.org/networking/ip-sysctl.html)). **[S]**
  - **Ubuntu** ships `use_tempaddr=2`, which prefers temporary addresses (procps `55-ipv6-privacy.conf`). **[M]**
  - **NetworkManager** generates stable-privacy addresses by default ([nm-settings](https://networkmanager.dev/docs/api/latest/nm-settings-nmcli.html)). **[S]**
  - **macOS** (this Mac): `use_tempaddr=1` and `prefer_tempaddr=1`. Temporary addresses stay valid for 7 days, not RFC 8981's 2. **[M]**
  - **Windows:** temporary addresses being on by default is stated only in secondary sources.
- **Source selection.**
  - RFC 6724 Rule 7 prefers a temporary source address. Implementations MAY reverse this and MUST let apps override it ([RFC 6724 §5](https://www.rfc-editor.org/rfc/rfc6724)). **[S]**
  - The override is `IPV6_ADDR_PREFERENCES` on Linux and `IPV6_PREFER_TEMPADDR` on macOS. **[M]**
  - A server bound to `::` answers on whatever address the client dialed. Source selection only matters for traffic the Node starts: STUN, pinhole requests, and any registration of its address. **[I]**

### STUN, NPTv6 and containers

- **STUN supports IPv6.** XOR-MAPPED-ADDRESS has an IPv6 family, XORed with the magic cookie and transaction ID ([RFC 8489 §14.2](https://www.rfc-editor.org/rfc/rfc8489)). **[S]**
  - Public STUN servers with AAAA records include `stun.l.google.com`, `stun.cloudflare.com` and `stun.nextcloud.com`. `global.stun.twilio.io` has none. **[M]** (`dig`)
- **Without translation, STUN's answer is the source address.** That is usually the temporary address, and it says nothing about inbound filtering. **[I]**
- **NPTv6 changes the picture.** NPTv6 ([RFC 6296](https://www.rfc-editor.org/rfc/rfc6296), Experimental) maps prefixes one to one. A host behind it "is unable to determine which external address it is using" without something like STUN (§2.4). **[S]**
  - So a STUN answer that differs from the interface address points to NPTv6 or NAT66, and the pinhole and DNS need the external form. **[I]**
  - OpenWrt leaves `masq6` off by default. **[M]**
- **Docker.**
  - IPv6 is opt-in per network, and only on Linux hosts ([Docker IPv6](https://docs.docker.com/engine/daemon/ipv6/)). **[S]**
  - Published ports listen on `[::]`. With an IPv4-only bridge and the userland proxy, "Ports on the host's IPv6 addresses will map to the container's IPv4 address" ([port publishing](https://docs.docker.com/engine/network/port-publishing/)). **[S]**
  - So a containerised Node can be reachable over the host's IPv6 while seeing no global address on its own interfaces. With IPv6 on, the default is a ULA behind NAT66. **[I]**
  - A Node must not decide "no IPv6" from its own interface list when it runs in a container. Host networking avoids the problem. **[I]**

### Prefix changes

- **A new prefix can arrive at any reboot.** RFC 8978 describes a router reboot that brings a new delegated prefix while hosts keep stale addresses. It also cites a UK survey (2018) in which "37% of the responding ISPs employ dynamic IPv6 prefixes" ([RFC 8978](https://www.rfc-editor.org/rfc/rfc8978)). **[S]**
- **RIPE-690** "strongly recommended" persistent prefixes, partly because they "allow DNS names to be assigned". It gives no rotation data ([RIPE-690](https://www.ripe.net/publications/docs/ripe-690/)). **[S]**
- **Some ISPs rotate daily.** The DynamIPs study (CoNEXT 2020) found Deutsche Telekom, Versatel, NetCologne, Telefónica DE and M-net renumbering IPv6 prefixes every 24 hours, and most other ISPs keeping prefixes for months ([pdf](https://www.caida.org/catalog/papers/2020_dynamips/dynamips.pdf)). **[S]**
  - No current primary source from Deutsche Telekom states its policy.
- **So an AAAA record for a home Node can go stale daily, and the diagnostic must re-check after a prefix change.** **[I]**
  - #16 already re-runs the diagnostic when the public address changes.
  - A FRITZ!Box reports the prefix lifetime through its IGD extension (section 3).

### Proving reachability

- **Only a connection from outside to the stable address and port proves reachability.** **[I]** Each local signal falls short:
  - Interface addresses and STUN show only the address.
  - A PCP or UPnP success is the router's policy, not a test.
  - miniupnpd's status is hard-wired to 1/1.
  - Host and upstream firewalls are invisible to the router.
- **Ping is not a test.** OpenWrt forwards ICMPv6 echo while dropping TCP (section 3). **[M]/[I]**
- **A probe can pass by accident.** With endpoint-independent filtering (RFC 6092 REC-11), recent outbound traffic from the service's port can open it to any peer. So a probe should target the service port, from a host that has not just been contacted. **[I]**
- **The probe should target the stable address, not the temporary one that STUN sees.** **[I]**

## 5. How common is CGNAT with native IPv6, and how often does a viewer lack IPv6?

### Two kinds of metric

- **Google** counts the share of its users whose connection to Google used IPv6 ([Google IPv6 statistics](https://www.google.com/intl/en/ipv6/statistics.html)). **[S]**
- **APNIC Labs** measures with ads. "Capable" is the share of users who can fetch an IPv6-only object; "preferred" is the share who use IPv6 when both are offered ([APNIC IPv6](https://stats.labs.apnic.net/ipv6)). **[S]**
- **Cloudflare Radar** counts requests, not users, so it doesn't compare with the other two. **[S]**

### Adoption (measured 2026-10-11)

- **Google, worldwide:** 48.70% on 2026-10-09. The all-time high was 51.69% on Saturday 2026-10-03. Over eight weeks, weekends averaged about 50% and weekdays about 46.8%. **[M]** (`adoption.js`)
- **APNIC, worldwide:** 43.6% capable and 41.1% preferred. **[M]** So about 56% of users worldwide are not IPv6-capable. **[I]**
- **Cloudflare Radar 2025:** 29% of requests for content offered over both protocols used IPv6 ([Year in Review](https://blog.cloudflare.com/radar-2025-year-in-review/)). **[S]**
- **By country** (Google % / APNIC capable %). **[M]**

| Country | Google | APNIC capable |
|---|---|---|
| France | 87.3 | 85.6 |
| Germany | 75.7 | 73.1 |
| India | 68.3 | 76.8 |
| US | 54.4 | 61.7 |
| UK | 53.6 | 54.4 |
| Japan | 56.6 | 61.8 |
| Brazil | 56.6 | 59.4 |
| Canada | 42.1 | 45.1 |
| Australia | 39.3 | 47.3 |
| Italy | 18.7 | 19.1 |
| Spain | 13.3 | 10.6 |

- **By network** (APNIC % IPv6-capable, 30-day window to 2026-10-10). **[M]**
  - **US mobile:** T-Mobile 94.4, Verizon Wireless 94.7, AT&T Mobility 87.1.
  - **US fixed:**
    - Comcast 85.3, AT&T 85.8, Cox 77.2, Starlink 87.5, Charter 65.8.
    - Frontier **23.8**.
    - CenturyLink/Qwest **1.6** and Optimum **1.4**. Viasat is about **0.1**.
  - **UK:**
    - BT 89.9, Sky 93.5, Vodafone 90.3.
    - Virgin Media **0.56**, TalkTalk **0.42**, Plusnet **0.65**, O2 broadband **0.89**.
  - **Germany:** Telekom 81.4, Vodafone 72.5, Telefónica 88.6.
  - **India:** Jio 96.2, Airtel mobile 89.4, BSNL 22.9, Hathway 1.0.
  - **Japan:** KDDI 89.9, SoftBank 62.8, J:COM 22.4.

### CGNAT, and CGNAT with IPv6

No source gives the share of *users* behind CGNAT. Every source counts networks or traffic.

- **Richter et al., IMC 2016** (data 2014–16) ([arXiv 1605.05606](https://arxiv.org/abs/1605.05606)). **[S]**
  - CGN was found in 17–18% of the non-cellular networks that serve end users, and in over 92% of cellular networks.
  - About 40% of ISPs surveyed ran CGN.
  - The authors note that "Some ISPs also provide customers with an internal (CGN) IPv4 address and a publicly reachable IPv6 address."
- **Livadariu et al. (CAIDA), 2018:** 39.5% of the networks with CGN announced IPv6, against 22% of all networks ([pdf](https://www.caida.org/catalog/papers/2018_inferring_carrier_grade_nat/inferring_carrier_grade_nat.pdf)). **[S]**
- **Hsu, Pearce, Li, Berger and Richter (Akamai, Georgia Tech and MIT), August 2026, a preprint not yet peer-reviewed** ([arXiv 2608.06517](https://arxiv.org/abs/2608.06517), title and date verified). **[S]**
  - 41.1% of IPv4 CDN traffic comes from massively shared addresses, which are only 1.6% of active IPv4 addresses. This is a lower bound on CGNAT, since it misses lightly shared addresses.
  - About half of that traffic comes from blocks that also have IPv6. In Africa it is under 20%.
  - 52% of the top providers studied run shared IPv4 alongside unshared IPv6.
  - Sharing of IPv6 /64s is rare: 5.6% of IPv6 traffic.
- **Documented lines with CGNAT IPv4 and native IPv6:**
  - **Starlink.**
    - Default IPv4 is CGNAT in `100.64.0.0/10` and "does not allow inbound traffic". Every plan gets native IPv6 with a /56 by DHCPv6-PD.
    - Public IPv4 comes only with Priority plans, and the router has no IPv4 or IPv6 firewall rules (Starlink article above).
    - **[S]**. The page renders by JavaScript, so its text was read through a live render and a search snippet of the official page.
  - **DS-Lite** carries IPv4 inside IPv6 to a carrier NAT, while IPv6 is native ([RFC 6333](https://www.rfc-editor.org/rfc/rfc6333)). **[S]** AVM says that on DS-Lite inbound access works only over IPv6, and only "if both parties have an IPv6 internet connection", which is "not the case in all mobile networks and Wi-Fi hotspots" ([AVM 1611](https://uk.fritz.com/service/knowledge-base/dok/FRITZ-Box-7430/1611_What-is-DS-Lite-and-how-does-it-work/)). **[S]**
    - Vodafone Germany cable is DS-Lite, but only Vodafone's forum says so. No official doc was found.
    - Japan's IPoE services use DS-Lite (transix) and MAP-E (v6プラス) ([transix](https://www.mfeed.ad.jp/transix/dslite/)). **[S]**
  - **T-Mobile US.**
    - Its mobile network is IPv6-only with 464XLAT ([RFC 6877](https://www.rfc-editor.org/rfc/rfc6877); [ISOC case study](https://www.internetsociety.org/resources/deploy360/2014/case-study-t-mobile-us-goes-ipv6-only-using-464xlat/)). **[S]**
    - Its Home Internet gateways "do not have the ability to enable and disable NAT/port forwarding" ([T-Mobile](https://www.t-mobile.com/support/home-internet/connect)). **[S]**
    - T-Mobile reported 8.45M 5G broadband customers at the end of 2025 ([SEC 8-K](https://www.sec.gov/Archives/edgar/data/1283699/000128369926000007/tmus12312025ex991.htm)). **[S]**
    - *User reports* say inbound IPv6 is filtered and there is no prefix delegation.

### Viewers without IPv6

- **Viewers on mobile data mostly have IPv6:** 87–95% on the big US carriers and 96% on Jio. **[M]**
- **Fixed-line viewers on some ISPs almost never have it:** Virgin Media, TalkTalk, Frontier, CenturyLink, Optimum and Viasat (above). **[M]**
- **Offices lag.**
  - RIPE Labs says corporate networks "tend to lag" ([2021](https://labs.ripe.net/author/stephen_strowes/ipv6-adoption-in-2021/)). **[S]**
  - Google's weekday dip of about 3 points fits that. **[M]** dip, **[I]** cause
  - Zscaler's network (AS22616), which carries corporate traffic, is 11.3% capable and 1.2% preferred. **[M]**
- **Hotels and public Wi-Fi.** No measurement exists. Wifirst, a French hotel and student Wi-Fi operator, is 1.5% capable. **[M]**, a proxy only.
- **The device matters as much as the network.** A Roku viewer can't use IPv6 for channel traffic even on an IPv6 network (section 2). **[S]**

## 6. Could the Relay Node reach a home Node over IPv6 directly, and would it matter?

### Facts it rests on

- **VPS providers offer IPv6, and none charges for it.** **[S]**
  - **Hetzner Cloud:** "We do not charge for our Primary IPv6s". IPv4 costs €0.50 a month, and servers can run IPv6-only ([Hetzner](https://docs.hetzner.com/cloud/servers/primary-ips/overview/)).
  - **DigitalOcean:** opt-in, "at no additional cost" ([DO](https://docs.digitalocean.com/products/networking/ipv6/)).
  - **Linode/Akamai:** every Linode gets one IPv6 address ([Akamai](https://techdocs.akamai.com/cloud-computing/docs/an-overview-of-ipv6-on-linode)).
  - **AWS Lightsail:** IPv6 is on by default for resources created since 2021 ([AWS](https://docs.aws.amazon.com/lightsail/latest/userguide/understanding-public-ip-and-private-ip-addresses-in-amazon-lightsail.html)). AWS has charged for public IPv4 since 2024-02-01 ([AWS](https://aws.amazon.com/blogs/aws/new-aws-public-ipv4-address-charge-public-ip-insights/)).
  - **Vultr and Oracle Cloud:** opt-in. Neither page states a price.
- **An outbound connection needs no pinhole.** A stateful router passes return traffic for flows started from inside (RFC 6092 REC-31) and rejects unsolicited inbound by default (REC-34). **[S]**
- **Idle timers set the keepalive interval.** **[S]**
  - **The standards' floors:**
    - Idle TCP state must last at least 2 h 4 min ([RFC 5382](https://www.rfc-editor.org/rfc/rfc5382) REQ-5; RFC 6092 REC-35).
    - UDP state must last at least 2 min, with 5 min recommended ([RFC 4787](https://www.rfc-editor.org/rfc/rfc4787) REQ-5).
  - **What real gateways do:**
    - Richter et al. measured a median CGN UDP timeout of 35 s (non-cellular) and 65 s (cellular).
    - Neubert, Winter and Winkler (ITC 36, 2025) found only 4 of 23 home gateways meeting the TCP floor. One dropped idle TCP after 1 minute ([pdf](https://thomasez.folk.ntnu.no/itc36/itc2025-final4.pdf)).
- **Head-of-line blocking.** HTTP/2 does not fix TCP head-of-line blocking ([RFC 9113 §1](https://www.rfc-editor.org/rfc/rfc9113)), and QUIC avoids it across streams ([RFC 9000](https://www.rfc-editor.org/rfc/rfc9000)). **[S]**
  - The Node link is one TLS connection multiplexed by yamux (ADR 0013).
  - The library research already flags its per-stream window cap and TCP head-of-line blocking for relayed traffic ([B-net-security §2](oss-libraries/B-net-security.md)).
- **An IPv4-only host can't reach an IPv6-only host** without a translator ([RFC 4213](https://www.rfc-editor.org/rfc/rfc4213)). **[S]**
- **A proxy can serve IPv4 clients from an IPv6 origin.** Tailscale's DERP servers are dual-stack and bridge IPv4-only and IPv6-only peers ([Tailscale](https://tailscale.com/kb/1232/derp-servers)). **[S]**
  - Cloudflare's docs cover only its "IPv6 compatibility" and "Pseudo IPv4" features ([Cloudflare](https://developers.cloudflare.com/network/ipv6-compatibility/)). **[S]**

### What follows

All of this is **[I]**, built on the facts above.

- **The relay could dial home over IPv6.** A dual-stack relay can accept a viewer's IPv4 connection and open a fresh TCP connection to the home Node's global IPv6 address. That requires:
  - **a pinhole.** It could be limited to the relay's source address, which is narrower than opening the port to everyone.
  - **a current home address.** The Node would report it over the Node link anyway.
  - **a router that can open pinholes at all.** Starlink's router and T-Mobile's gateways can't.
  - **acceptance that the home Node is exposed to the internet.** The Node link exposes nothing inbound.
- **It buys little.**
  - Latency is the same, because both designs run viewer → relay → home over the same route.
  - The gains are escaping the single TCP connection's head-of-line blocking and the yamux window cap, and simpler per-stream forwarding.
  - A QUIC Node link, several TCP connections, or a larger window would get the same gains without IPv6. ADR 0013 and the library research already weigh those.
- **The reverse direction matters more.** If the relay has an AAAA record, the home Node can dial the Node link over IPv6.
  - The connection passes the home firewall as outbound traffic, with no pinhole.
  - It also skips the carrier's IPv4 translation state (CGN, the DS-Lite AFTR, 464XLAT or MAP-E port sets), so the link is no longer hostage to the short CGN UDP and TCP timeouts measured above.
  - Happy Eyeballs on the Node's dialer would pick IPv6 automatically.
  - This changes the Node link's resilience, not whether a relay is needed.
- **IPv6 never removes the need for a relay for viewers without IPv6.** That is about 56% of users worldwide, nearly all users of some large ISPs, and every Roku channel.
- **IPv6 can remove the relay hop for viewers who do have it.** That needs a pinhole and DNS that publishes both the home AAAA and the relay's A record. Whether to do that is a split-horizon question for #115.

## Surprises

- **Plex IPv6 is months old, Plex Pass only "temporarily", and its connect-back status over IPv6 is undocumented.** The support articles still describe the old checkbox.
- **Roku staff confirm that ordinary channels can't make IPv6 requests**, even on IPv6-capable Roku OS.
- **Neither pinhole protocol works out of the box on the common open firmwares:**
  - miniupnpd always reports the IPv6 firewall as enabled with pinholes allowed.
  - OpenWrt's default config hides the UPnP IPv6 service.
  - pfSense lacks IGDv2.
  - OPNsense builds without IPv6 pinholes.
  - Tailscale's portmapper refuses IPv6 outright.
- **Both pinhole protocols open only the requester's source address**, which by default is the temporary one.
- **Go's `IsGlobalUnicast` returns true for ULAs.**
- **OpenWrt forwards ICMPv6 echo to the LAN while it drops TCP**, so ping proves nothing.
- **Large UK and US fixed ISPs measure under 2% IPv6-capable**: Virgin Media, TalkTalk, CenturyLink and Optimum.
- **German ISPs renumbered IPv6 prefixes daily** in the 2020 data.
- **Starlink's router supports no IPv6 firewall rules.**
- **Most home gateways tested break RFC 5382's TCP idle floor**, and one dropped idle TCP after a minute.

## Gaps

No primary source was found for any of these:

- How plex.tv's reachability worker treats IPv6, and the default value of Plex's Client Network setting.
- Whether PMS binds dual-stack.
- How the official Plex apps order their connection tests.
- IPv6 behaviour in Plex, Jellyfin and Emby clients on Samsung Tizen, LG webOS, PlayStation and iOS.
- How Emby sets up its listening socket, and whether it has any reachability test.
- A deployment survey of PCP or `WANIPv6FirewallControl` on routers.
- Docs for UniFi, AT&T, Apple, Netgear and ASUS.
- Whether Google Nest's automatic IPv6 port opening is UPnP.
- Starlink's and T-Mobile's default inbound IPv6 policy.
- BBF TR-124, which was not reachable.
- The share of users, as opposed to networks, behind CGNAT, and the share of DS-Lite and MAP-E lines.
- Measured IPv6 coverage on hotel and public Wi-Fi.
- Deutsche Telekom's current prefix-rotation policy.
- Whether Windows enables temporary addresses by default.
- Idle timeouts on IPv6 home gateways.
- A live STUN-over-IPv6 test. The machine used has no global IPv6 address, only link-local and a ULA.
