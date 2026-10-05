# CGNAT, remote access and relay techniques

Research for [#15](https://github.com/jatassi/SlipStream/issues/15). It feeds charting decision Q10 in [#2](https://github.com/jatassi/SlipStream/issues/2): the forwarder carries Media Server traffic, the SlipStream UI and the portal, with TLS, a domain, and ports the user names. Terms follow `GLOSSARY.md` on `docs/v2-glossary`. "Relay Node" below means a Node assigned the relaying Role.

Sources were read on 2026-10-04. Every claim links its source. Anything marked **inferred** is reasoning from those sources, not something a source states. This file lays out options and tradeoffs. It does not pick one.

## Answer in brief

- All three Media Servers expect an inbound port on the home public IPv4 address. Plex and Emby try UPnP or NAT-PMP first. Jellyfin removed automatic port mapping in January 2025. Only Plex has a relay of its own, and Plex caps it at 2 Mbps.
- Plex decides reachability from outside. plex.tv sends a worker to connect back to the address and port the server reports.
- A Node can classify its network without outside help. It compares the gateway's WAN address (from UPnP IGD, NAT-PMP or PCP) with the public address a STUN server reports. A gateway WAN address in `100.64.0.0/10` means CGNAT. Proving that a port is reachable from outside still needs a vantage point outside the home network.
- The main relay designs are a WireGuard tunnel from the home Node to a VPS (userspace via `wireguard-go/tun/netstack`, the design Pangolin's Newt uses), a multiplexed TCP, QUIC or WebSocket tunnel (frp, rathole, `ssh -R`), and hosted tunnels (Cloudflare Tunnel, Tailscale Funnel, Plex Relay). Hosted tunnels limit bandwidth or video.
- TLS can terminate on the Relay Node, with ACME HTTP-01 or TLS-ALPN-01 answered there. The Relay Node can also pass TLS through by SNI, so certificates and keys stay on the home Node.
- No Media Server's terms forbid relaying in so many words. Plex documents reverse proxies and VPNs as supported setups. Plex's terms do forbid circumventing its mechanisms. A relay that makes remote viewers look local to Plex therefore carries risk, because Plex now charges for remote playback. **Inferred.**

## 1. How each Media Server does remote access

### Plex

**Port mapping.** Remote Access requires the server to be signed in to a Plex account. "The server will attempt to automatically configure a connection through your router using UPnP or NAT-PMP first" ([Plex: Remote Access](https://support.plex.tv/articles/200289506-remote-access/)). A manual forward maps any external port to internal TCP 32400. The user must also tick "Manually specify public port", or the server "will continue to try and automatically map the port and will almost certainly fail" (same page). The internal port "must always be 32400 (TCP)" ([Plex: Troubleshooting Remote Access](https://support.plex.tv/articles/200931138-troubleshooting-remote-access/)).

**How Plex decides the server is reachable.** The server makes an HTTP request to plex.tv whenever the Remote Access settings page opens. "That service then has a worker try to reach out to your server, using the supplied remote connection information, to see whether it's accessible." The workers run on a cluster whose source IPs change, and Plex publishes the current list ([Plex: Troubleshooting Remote Access](https://support.plex.tv/articles/200931138-troubleshooting-remote-access/)). The page also shows a "Server Mapping" view of the private address, the public address, and the link to the internet ([Plex: Remote Access](https://support.plex.tv/articles/200289506-remote-access/)). The server exposes this state at `/myplex/account`, with the fields `mappingState`, `mappingError`, `publicAddress` and `privateAddress`. Plex's own API reference does not document that endpoint ([developer.plex.tv/pms](https://developer.plex.tv/pms/)). The field names come from the third-party python-plexapi client ([`server.py`](https://github.com/pkkid/python-plexapi/blob/master/plexapi/server.py)).

**Double NAT and CGNAT.** Plex says double NAT "will interfere with the server attempting to create automatic connections". It names CGNAT as a case of it. Plex's detection test is manual. Compare the router's WAN IPv4 address with the address a "what is my IP" site shows. A mismatch means double NAT. A router WAN address in 100.64.0.0 to 100.127.255.255 means CGNAT. Plex's fix for CGNAT is to buy a static public IP from the ISP ([Plex: Troubleshooting Remote Access](https://support.plex.tv/articles/200931138-troubleshooting-remote-access/)).

**How clients find the server.** plex.tv publishes a list of connections for each server. Each entry carries `local` and `relay` flags. Clients fetch the list from `https://plex.tv/api/v2/resources?includeHttps=1&includeRelay=1`. python-plexapi tries local connections first, then remote, then relay (`DEFAULT_LOCATION_ORDER = ['local', 'remote', 'relay']` in [`myplex.py`](https://github.com/pkkid/python-plexapi/blob/master/plexapi/myplex.py)). Plex does not document this endpoint. The order is that client's choice, and official apps may differ. **Inferred.**

**Custom server access URLs.** This setting takes "a comma-separated list of URLs (either HTTP or HTTPS), which will be published to plex.tv for server discovery". Plex names the uses: "if you're using a VPN to get back home, if you're using a reverse proxy in front of the media server" ([Plex: Network settings](https://support.plex.tv/articles/200430283-network/)). This is the documented hook for publishing a Relay Node's address to Plex clients. Plex does not say whether its worker check also tests these URLs. Unknown.

**Plex Relay.** When an app cannot connect directly, Plex relays the stream. Relay is on by default. "Connections are limited to 2 Mbps maximum for streams (this applies to both Plex Pass and Remote Watch Pass subscribers)." Downloads do not work over Relay, and not every app supports it. The stream stays encrypted if secure connections are enabled ([Plex: Accessing a Server through Relay](https://support.plex.tv/articles/216766168-accessing-a-server-through-relay/)).

**TLS.** Plex issues certificates for each server, now from Let's Encrypt ([Plex: Secure Server Connections](https://support.plex.tv/articles/206225077-how-to-use-secure-server-connections/)). The scheme uses hostnames such as `1-2-3-4.<hash>.plex.direct`, where "any `IP.*.plex.direct` domain resolves to `IP`". Each user gets a wildcard certificate for `*.<hash>.plex.direct` ([Filippo Valsorda, How Plex is doing HTTPS for all its users](https://words.filippo.io/how-plex-is-doing-https-for-all-its-users/)). A connection to a custom domain or bare IP does not match that certificate ([Plex: Secure Server Connections](https://support.plex.tv/articles/206225077-how-to-use-secure-server-connections/)). For a custom domain, Plex accepts a PKCS #12 file and a "Custom certificate domain". That domain is "published to plex.tv using the port you currently have mapped" ([Plex: Network settings](https://support.plex.tv/articles/200430283-network/)).

**Local or remote.** "LAN Networks" sets which addresses count as local. Remote bandwidth limits do not apply to local playback. By default only the server's own subnet is local ([Plex: Network settings](https://support.plex.tv/articles/200430283-network/)). Since 2025-04-29, remote playback of personal video needs a Plex Pass held by the server admin, or a Plex Pass or Remote Watch Pass held by the viewer. "Remote" means "any time that the player app cannot make a connection to the Plex Media Server on the same subnet of the same local network". Plex lists VPNs and containers without host networking as causes of misdetection ([Plex: Requirements for Remote Playback of Personal Media](https://support.plex.tv/articles/requirements-for-remote-playback-of-personal-media/)). Plex does not document how it handles `X-Forwarded-For`. One user report, with no staff reply, says Plex reads it but ignores private-range entries ([Plex forum: X-Forwarded-For Trust Configuration](https://forums.plex.tv/t/x-forwarded-for-trust-configuration/889105)).

### Jellyfin

- Ports are HTTP 8096, HTTPS 8920, and UDP 7359 for client discovery on the local network ([Jellyfin: Networking](https://jellyfin.org/docs/general/post-install/networking/)).
- Jellyfin lists four remote-access routes. They are a direct port forward ("not recommended!"), a reverse proxy, a VPN, and a VPS reverse proxy to the home network. It suggests terminating HTTPS on a reverse proxy (same page).
- Jellyfin has no relay or cloud service. It is self-hosted only (same page) and licensed GPL-2.0 ([jellyfin/jellyfin](https://github.com/jellyfin/jellyfin)).
- Automatic port mapping is gone. PR [#13222](https://github.com/jellyfin/jellyfin/pull/13222), "Remove the ability to auto port forward", merged on 2025-01-25. Its reason was that "the package we're using to do this has multiple dependencies with vulnerabilities". `EnableUPnP` now carries `[Obsolete("No longer supported")]` in [`NetworkConfiguration.cs`](https://github.com/jellyfin/jellyfin/blob/master/MediaBrowser.Common/Net/NetworkConfiguration.cs).
- Local and remote are configured in the same file. `LocalNetworkSubnets`, `KnownProxies`, `EnableRemoteAccess` (default true), `RemoteIPFilter`, and `EnableIPv6` (default false) all live there.
- Behind a reverse proxy, the proxy's IP must be listed in Known Proxies. Otherwise "Jellyfin sees the proxy's IP address rather than the client's". Jellyfin discards forwarded-for headers that do not come from a known proxy. Websockets must pass through ([Jellyfin: Reverse Proxy](https://jellyfin.org/docs/general/post-install/networking/reverse-proxy/)). The docs do not mention the PROXY protocol.

### Emby

- Remote access is a server toggle plus a toggle for each user. "Enable automatic port mapping" uses UPnP. If the option is missing, the user installs the Port Mapper plugin. Default public ports are 8096 and 8920 ([Emby: Connecting from Client Apps](https://emby.media/support/articles/Connectivity.html)).
- The reachability check is manual. The user visits canyouseeme.org, and "the IP you see listed here must match what Emby shows on the dashboard. If it doesn't, remote access will not work and likely your ISP is using cgNAT or you are using a VPN." The page also gives the `100.64.0.0 - 100.127.255.255` range test and a traceroute test that looks for private-range hops (same page).
- The page notes that testing the external address from inside the home may fail, because "some internet routers disable loopback capabilities" (same page). **Inferred:** a diagnostic that dials the public address from the LAN can fail on hairpin NAT even when the forward works.
- Emby Connect is a directory, not a relay. "No, all connections are direct from your devices to your Emby Server" ([Emby Connect](https://emby.media/support/articles/Emby-Connect.html)).
- For VPN subnets, Emby says to add them to "LAN Networks" so the server treats them as local ([Emby: Connecting from Client Apps](https://emby.media/support/articles/Connectivity.html)). Emby's TLS guide notes that Let's Encrypt certificates "may have compatibility issues with LG and Samsung Smart TVs" ([Emby: Secure Your Server](https://support.emby.media/support/articles/Secure-Your-Server.html)).

### Comparison

| | Plex | Jellyfin | Emby |
|---|---|---|---|
| Automatic port mapping | UPnP, NAT-PMP | Removed in Jan 2025 | UPnP (built in or plugin) |
| Reachability check | plex.tv worker connects back | None | Manual (canyouseeme.org) |
| Vendor relay | Plex Relay, 2 Mbps | None | None (Emby Connect is a directory) |
| Publish an alternate URL | Custom server access URLs | Published server URIs | Not reviewed |
| Built-in TLS | plex.direct, or a custom PKCS #12 | PKCS #12, HTTPS off by default | PKCS #12 |
| Local or remote by | Source IP vs LAN Networks; XFF undocumented | Source IP, or XFF from Known Proxies | Source IP vs LAN Networks |
| Remote playback paywall | Yes, since 2025-04-29 | No | Not reviewed |

## 2. Detecting CGNAT, double NAT and port-mapping support from inside

### Signals a Node can gather

| Signal | How | What it gives | Source |
|---|---|---|---|
| Default gateway and own address | OS routing table | The first-hop router to query | n/a |
| UPnP IGD present | SSDP `M-SEARCH` to `239.255.255.250:1900` | Device description and control URL | [UPnP Device Architecture 2.0](https://openconnectivity.org/upnp-specs/UPnP-arch-DeviceArchitecture-v2.0-20200417.pdf) |
| Gateway WAN address via UPnP | `WANIPConnection` `GetExternalIPAddress()` returns `NewExternalIPAddress` | The address on the gateway's outside interface | [WANIPConnection:2](https://upnp.org/specs/gw/UPnP-gw-WANIPConnection-v2-Service.pdf), §2.5.20 |
| Gateway WAN address via NAT-PMP | UDP to gateway port 5351, opcode 0 | "the external IPv4 address of the NAT gateway" | [RFC 6886](https://www.rfc-editor.org/rfc/rfc6886) §3.2 |
| PCP support and mapped address | UDP 5351, ANNOUNCE or MAP | "Assigned External IP Address" on a MAP success | [RFC 6887](https://www.rfc-editor.org/rfc/rfc6887) §11 |
| Public address as the internet sees it | STUN Binding request, default port 3478, read `XOR-MAPPED-ADDRESS` | The server-reflexive address | [RFC 8489](https://www.rfc-editor.org/rfc/rfc8489) |
| Endpoint-dependent mapping ("hard NAT") | STUN to two or more servers and compare | Whether the mapping varies by destination | [RFC 5780](https://www.rfc-editor.org/rfc/rfc5780); Tailscale `MappingVariesByDestIP` in [`netcheck.go`](https://github.com/tailscale/tailscale/blob/main/net/netcheck/netcheck.go) |
| Private hops upstream | Traceroute, counting RFC 1918 or `100.64/10` hops before the first public hop | A second NAT layer | [Emby: Connecting from Client Apps](https://emby.media/support/articles/Connectivity.html) |
| Inbound reachability | A host outside connects back to the public address and port | Proof that the port is open | Plex worker model, [Plex: Troubleshooting](https://support.plex.tv/articles/200931138-troubleshooting-remote-access/) |

### Classification rules (inferred from the sources above)

1. The gateway reports a WAN address in `100.64.0.0/10`. That means CGNAT. [RFC 6598](https://www.rfc-editor.org/rfc/rfc6598) reserves the block to number "the interfaces that connect CGN devices to Customer Premises Equipment".
2. The gateway reports a WAN address in RFC 1918 space. That means double NAT. The second NAT may be the user's own router, or an ISP CGN that numbers with RFC 1918 space, which RFC 6598 §3 allows under some conditions.
3. The gateway reports a public WAN address that differs from the STUN address. That means another NAT sits upstream. RFC 6598 §5.1 warns that a CPE outside address can be globally unique yet "not advertised to the global Internet as when it is behind a CGN". A range check alone misses this case.
4. The gateway WAN address equals the STUN address. Then there is a single NAT, and a port forward can work. Only an inbound test proves it.
5. No gateway answers UPnP, NAT-PMP or PCP. Then the WAN address is unknown. Fall back to the STUN address, traceroute, and an inbound test.

### Limits

- NAT-PMP talks only to the default gateway. It "is not appropriate" where the NAT is some other device ([RFC 6886](https://www.rfc-editor.org/rfc/rfc6886) §3). It does not chain through nested NATs (§4.3.1). A mapping that succeeds on the inner router therefore does not make the port reachable under double NAT. **Inferred** from §4.3.1.
- [RFC 6888](https://www.rfc-editor.org/rfc/rfc6888) REQ-9 says a CGN "MUST implement a protocol giving subscribers explicit control over NAT mappings" and that it "SHOULD be the Port Control Protocol". By default a PCP client sends to its default router ([RFC 6887](https://www.rfc-editor.org/rfc/rfc6887) §8.1), which is the home router, not the CGN. Whether any given ISP's CGN accepts PCP is unknown and must be probed.
- UPnP IGD:2 recommends access control. Unauthenticated control points may be limited to mappings for their own IP and to ports 1024 and above. Devices may choose another policy ([WANIPConnection:2](https://upnp.org/specs/gw/UPnP-gw-WANIPConnection-v2-Service.pdf), §1.2). `GetExternalIPAddress` itself may return error 606 under such a policy (§2.5.20.3).
- NAT behaviour tests give "no absolute determination". "NAT devices do not behave consistently enough to predict future behavior" ([RFC 5780](https://www.rfc-editor.org/rfc/rfc5780) §1).
- STUN needs a public STUN server. A Relay Node can serve this role, and it can also run the inbound connect-back test. Without any outside host, the diagnostic can classify the NAT but cannot prove reachability. **Inferred.**

### Prior art

Tailscale's `portmapper.Probe` probes NAT-PMP, PCP and UPnP in parallel with a 250 ms budget. It sends SSDP both unicast to the gateway and multicast, because some routers answer only one ([`portmapper.go`](https://github.com/tailscale/tailscale/blob/main/net/portmapper/portmapper.go), BSD-3-Clause). Its `netcheck` report records UPnP, PMP, PCP, and whether the mapping varies by destination ([`netcheck.go`](https://github.com/tailscale/tailscale/blob/main/net/netcheck/netcheck.go)). Both are Go code under a permissive license.

## 3. Relay designs

**Constraint (inferred).** Media Server clients are stock apps such as Plex apps, Jellyfin apps, and browsers. They cannot run a tunnel agent. So any relay must expose a publicly reachable endpoint, and peer-to-peer hole punching between the viewer and home does not apply. The ports such a relay opens are the ones Q10 has the user name.

### A. WireGuard tunnel from the home Node to a VPS

- The home Node dials out to the Relay Node over UDP. Outbound UDP works through CGNAT, but the mapping must be kept alive. [RFC 4787](https://www.rfc-editor.org/rfc/rfc4787) REQ-5 forbids UDP mapping timers under two minutes. The `wg(8)` man page suggests "a persistent keepalive interval of 25 seconds" behind NAT ([wg.8](https://git.zx2c4.com/wireguard-tools/about/src/man/wg.8)).
- **Userspace.** `golang.zx2c4.com/wireguard/tun/netstack` runs WireGuard plus gVisor's TCP/IP stack inside the process. It exposes `ListenTCP` and `DialContext` on the tunnel without a TUN device or root ([`http_server.go` example](https://github.com/WireGuard/wireguard-go/blob/master/tun/netstack/examples/http_server.go), MIT). For Linux, wireguard-go's README says to prefer the kernel module, "which is faster" ([README](https://github.com/WireGuard/wireguard-go)). Netstack throughput for video is unmeasured here.
- **Scope of the tunnel (inferred).** A netstack tunnel carries only the sockets the SlipStream process opens. It cannot route a separate Media Server process's traffic. Inbound viewer traffic therefore arrives at the home Node, which dials the Media Server on the LAN. That last hop has a LAN source address, which matters for local-or-remote detection (see below).
- **Prior art.** Pangolin's Newt is "a fully user space WireGuard tunnel client and TCP/UDP proxy". It uses wireguard-go netstack to reach a VPS-side peer (Gerbil) and proxies TCP and UDP to LAN targets ([fosrl/newt](https://github.com/fosrl/newt)). Newt is AGPL-3.0 with a commercial dual license. Pangolin is AGPL-3 plus a "Fossorial Commercial License" on some files ([Pangolin LICENSE](https://github.com/fosrl/pangolin/blob/main/LICENSE)). Copying its code would bring those terms into SlipStream. **Inferred.**
- **Tailscale tsnet** embeds Tailscale in a Go program with "a userspace TCP/IP networking stack" ([Tailscale: tsnet](https://tailscale.com/kb/1244/tsnet)). It depends on a Tailscale or Headscale control server ([juanfont/headscale](https://github.com/juanfont/headscale), BSD-3-Clause).
- **Transport limit.** WireGuard runs over UDP. Networks that block UDP need a fallback. Tailscale falls back to DERP relays, which forward "already-encrypted traffic" ([Tailscale: DERP servers](https://tailscale.com/kb/1232/derp-servers)).

### B. Multiplexed reverse tunnel over TCP, QUIC or WebSocket

- **frp** (Apache-2.0) exposes TCP, UDP, HTTP and HTTPS. Its transports include TCP, KCP and QUIC, and it supports the PROXY protocol ([fatedier/frp](https://github.com/fatedier/frp)).
- **rathole** (Apache-2.0, Rust) forwards TCP and UDP. Its transports are TCP, TLS, Noise and WebSocket ([rathole-org/rathole](https://github.com/rathole-org/rathole)).
- **OpenSSH `ssh -R`** uses the same pattern. It needs `sshd` on the VPS, with `GatewayPorts` set to bind public addresses (inferred from OpenSSH configuration conventions, not read this session).
- **Tradeoff (inferred).** TCP-in-TCP suffers head-of-line blocking under loss. QUIC and WireGuard avoid it. The TCP and WebSocket transports get through networks that allow only HTTPS.

### C. Hosted tunnels

- **Cloudflare Tunnel** (cloudflared, Apache-2.0) routes through Cloudflare's CDN. Cloudflare's terms say non-Enterprise customers must use paid services "to serve video and other large files via the CDN". Cloudflare may limit use "to serve video or a disproportionate percentage of pictures, audio files, or other large files" ([Cloudflare Service-Specific Terms](https://www.cloudflare.com/service-specific-terms-application-services/), updated 2026-09-28).
- **Tailscale Funnel** relays a TCP proxy with TLS terminated on the device. The relay "cannot decrypt data". It allows only ports 443, 8443 and 10000 and only `*.ts.net` names. Traffic "is subject to non-configurable bandwidth limits" ([Tailscale: Funnel](https://tailscale.com/kb/1223/funnel)). That fails Q10's user-named ports and own domain. **Inferred.**
- **Plex Relay** is Plex-only and capped at 2 Mbps (section 1).

### D. No relay

- **IPv6.** Plex and Jellyfin both have IPv6 switches, and Jellyfin's defaults to off ([Plex: Network](https://support.plex.tv/articles/200430283-network/); [`NetworkConfiguration.cs`](https://github.com/jellyfin/jellyfin/blob/master/MediaBrowser.Common/Net/NetworkConfiguration.cs)). It helps only when the viewer's network also has IPv6. **Inferred.**
- **PCP to the CGN** (RFC 6888 REQ-9). ISP support is unknown.
- **A static public IPv4 from the ISP.** This is Plex's and Emby's documented fix.

### Cross-cutting: client IP and local-or-remote detection

All three Media Servers classify viewers by source address (section 1). Each relay design changes the source address the Media Server sees.

| Last hop to the Media Server | What the Media Server sees | Consequence |
|---|---|---|
| Home Node dials it over the LAN, raw TCP | The home Node's LAN IP | Every remote viewer looks local. Plex skips remote bandwidth limits, and may skip the remote-playback check. **Inferred** from Plex's definition of remote. |
| Home Node proxies HTTP and adds `X-Forwarded-For` | Jellyfin: the real client, if the home Node is in Known Proxies. Plex: undocumented; one user reports public XFF addresses are used. Emby: not documented in what was read. | Correct classification on Jellyfin. Unverified on Plex and Emby. |
| PROXY protocol header | Only if the receiver supports it ([PROXY protocol spec](https://www.haproxy.org/download/3.0/doc/proxy-protocol.txt): the receiver "MUST be configured to only receive the protocol") | None of the three Media Servers document support. |
| Relay Node's tunnel IP placed in LAN Networks | Local | This is what Plex's and Emby's VPN guidance does for subnets inside the home. Applying it to remote viewers is the terms risk in section 5. |

### Comparison

| Design | Runs on the home side | Needs a VPS | UDP-blocked networks | Bandwidth ceiling | Own domain and ports | License of prior art |
|---|---|---|---|---|---|---|
| A. WireGuard netstack | In the SlipStream process, no root | Yes | Needs a fallback | The VPS link | Yes | MIT (wireguard-go) |
| A'. tsnet + Headscale | In process | Yes, plus a control server | DERP fallback | The VPS link | Yes | BSD-3 |
| B. frp or rathole style | In process (reimplemented) or a sidecar | Yes | Yes, over TCP or WebSocket | The VPS link | Yes | Apache-2.0 |
| C. Cloudflare Tunnel | Sidecar | No | Yes | Terms restrict video | Domain yes, ports limited | Apache-2.0 client |
| C. Tailscale Funnel | Sidecar or tsnet | No | Yes | Non-configurable limits | No (`ts.net`, 3 ports) | BSD-3 |
| C. Plex Relay | Built into Plex | No | Yes | 2 Mbps | No, Plex only | Proprietary |

## 4. TLS and ACME on the relay

### Where TLS terminates

- **On the Relay Node.** The Relay Node holds certificates and keys and sees plaintext. It can route by Host header, add `X-Forwarded-For`, and serve the SlipStream UI and portal. Media Server traffic is then decrypted on a rented VPS. **Inferred.**
- **On the home Node (SNI passthrough).** The Relay Node reads the SNI in the ClientHello and forwards raw bytes. Keys stay home. The client IP survives only through the PROXY protocol or not at all. Funnel's split works this way ([Tailscale: Funnel](https://tailscale.com/kb/1223/funnel)). **Inferred** for SlipStream.
- **Plex in either mode.** A client that reaches Plex through a custom domain needs a certificate for that domain (section 1). Either the Relay Node terminates TLS for that domain, or Plex is loaded with a custom PKCS #12 for it. Plex's own plex.direct certificate does not cover a custom domain ([Plex: Secure Server Connections](https://support.plex.tv/articles/206225077-how-to-use-secure-server-connections/)).

### ACME challenges

| Challenge | Port on the validated host | Wildcards | IP identifiers | Notes |
|---|---|---|---|---|
| HTTP-01 | TCP 80, `/.well-known/acme-challenge/` ([RFC 8555](https://www.rfc-editor.org/rfc/rfc8555) §8.3) | No | Yes ([RFC 8738](https://www.rfc-editor.org/rfc/rfc8738)) | Fails if port 80 is blocked ([Let's Encrypt: challenge types](https://letsencrypt.org/docs/challenge-types/)) |
| TLS-ALPN-01 | TCP 443, ALPN `acme-tls/1` ([RFC 8737](https://www.rfc-editor.org/rfc/rfc8737)) | No | Yes (RFC 8738) | Answered at the TLS layer |
| DNS-01 | None | Yes | No ("MUST NOT", RFC 8738) | Needs DNS provider API credentials ([Let's Encrypt](https://letsencrypt.org/docs/challenge-types/)) |
| DNS-PERSIST-01 | None | n/a | n/a | Being standardized. The TXT record need not change at each renewal ([Let's Encrypt, 2025-12-02](https://letsencrypt.org/2025/12/02/from-90-to-45)) |

**Inferred placement.** With TLS terminating on the Relay Node, it owns the public 80 and 443 and answers HTTP-01 or TLS-ALPN-01 directly. With SNI passthrough, the Relay Node can forward port 80 challenge paths, or the `acme-tls/1` handshake on 443, to the home Node. DNS-01 works from either side.

### Libraries (Go)

- `golang.org/x/crypto/acme/autocert` does TLS-ALPN-01, plus HTTP-01 when `HTTPHandler` serves port 80. It has no DNS-01 ([pkg.go.dev](https://pkg.go.dev/golang.org/x/crypto/acme/autocert)).
- CertMagic (Apache-2.0) solves HTTP, TLS-ALPN and DNS through libdns providers. It supports ARI (RFC 9773) and shared storage for clusters. Its On-Demand TLS needs an allow-policy to avoid abuse ([caddyserver/certmagic](https://github.com/caddyserver/certmagic)).

### Let's Encrypt constraints

- Lifetimes are shrinking. The opt-in `tlsserver` profile issued 45-day certificates from 2026-05-13. The default `classic` profile moves to 64 days on 2027-02-10 and to 45 days on 2028-02-16. Authorization reuse drops to 7 hours by then ([Let's Encrypt, 2025-12-02](https://letsencrypt.org/2025/12/02/from-90-to-45)).
- Rate limits: 50 certificates per registered domain per 7 days, and 5 per identical identifier set per 7 days. ARI renewals are "exempt from all rate limits" ([Let's Encrypt: rate limits](https://letsencrypt.org/docs/rate-limits/), updated 2026-08-05).
- IP address certificates are generally available on the `shortlived` profile, valid 160 hours ([Let's Encrypt, 2026-01-15](https://letsencrypt.org/2026/01/15/6day-and-ip-general-availability)). This allows HTTPS on a bare Relay Node IP without a domain. **Inferred.**
- Some TV clients have trouble with Let's Encrypt chains, according to Emby ([Emby: Secure Your Server](https://support.emby.media/support/articles/Secure-Your-Server.html)) and Plex's note about LG root stores ([Plex: Secure Server Connections](https://support.plex.tv/articles/206225077-how-to-use-secure-server-connections/)).

## 5. Terms

**Plex** ([Terms of Service](https://www.plex.tv/about/privacy-legal/plex-terms-of-service/), revised 2025-10-30):
- The license is personal and non-commercial. Users may share access with "members of your immediate family" as Authorized Users.
- Users may not "disable, circumvent, or otherwise avoid or undermine any security device, mechanism, protocol, or procedure implemented in the Plex Solution". They may not use it "for the benefit of any third party" in a service-bureau arrangement.
- "Enforcing Security" forbids "circumventing any Plex security or authentication measures" and "using any false, misleading, or deceptive TCP-IP packet header information".
- The terms say nothing about reverse proxies or relays. Plex's support docs present reverse proxies and VPNs as supported ([Plex: Network settings](https://support.plex.tv/articles/200430283-network/)).
- **Inferred risk.** A relay that makes remote viewers look local, through a LAN source IP or a relay subnet in LAN Networks, avoids the remote-playback paywall. That plausibly falls under "circumvent... any... mechanism". A relay that keeps viewers classified as remote is ordinary reverse-proxy use. Plex states no position.

**Jellyfin.** GPL-2.0, no account or service terms ([jellyfin/jellyfin](https://github.com/jellyfin/jellyfin)). Jellyfin's docs list a VPS reverse proxy as a supported route ([Jellyfin: Networking](https://jellyfin.org/docs/general/post-install/networking/)).

**Emby** ([Terms](https://emby.media/terms.html), updated 2026-08-16):
- The software license is "personal, non-commercial". It forbids modifying the software and using it for "timesharing or service bureau purposes".
- Users must use the Access Services "solely for non-commercial, private and personal use".
- Nothing addresses relays or proxies. Emby's own docs describe reverse proxy and VPN setups ([Emby: Connecting from Client Apps](https://emby.media/support/articles/Connectivity.html)).

**Relay carriers.**
- Cloudflare restricts video over its CDN without paid services. Tunnel traffic is served through the CDN, so this plausibly applies (**inferred**; quote in section 3C).
- Tailscale Funnel has non-configurable bandwidth limits.
- VPS providers' acceptable-use and egress terms vary and were not reviewed.

## Open questions for prototypes

1. Does Plex's worker check, or the client's connection choice, treat a Custom server access URL on a Relay Node as remote? Does the green Remote Access status matter if clients connect through it anyway?
2. What source address and `X-Forwarded-For` behaviour make Plex and Emby classify a relayed viewer correctly as remote?
3. What throughput does wireguard-go netstack sustain for a 4K remux stream, against kernel WireGuard and a TCP or QUIC tunnel?
4. How often do home networks block outbound UDP, so that a TCP or WebSocket fallback is needed?
5. Do real CGNAT ISPs answer PCP from the home side?
