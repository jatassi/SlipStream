# Reachability recorded run, 2026-11-24 (S51 (Reachability diagnostic), #176)

Copied from Jackson's media server (C:\Users\jacks\Git\SlipStream\recorded-runs-out\reachability\reachability-run.md and its .jsonl, which hold no token). Plex Media Server 1.43.5.11079 on Windows 11 build 26300, 14:14 to 14:16 UTC, read only: nothing was written to Plex, the firewall, the router, DNS or services. The recorder ran unelevated and read Plex with the registry's PlexOnlineToken. The home WAN address reads as 203.0.113.9 and the server's ULA as fd72::5 here; every other value is as measured.

`plexkind`'s `plexExposed` holds the pref ids below, and `fakes/plex` answers the `/myplex/account` and `/:/prefs` shapes. Not recorded: whether the server's own resources token, which SlipStream holds, reads `/myplex/account` and `/:/prefs`, and any `mappingState` but `mapped`.

## Results by step

| Step | Key values |
| --- | --- |
| 1. `GET /` | `myPlex` true, `myPlexMappingState` `mapped`, `myPlexSigninState` `ok`, `myPlexSubscription` true, `platform` Windows, `platformVersion` `10.0 (Build 26300)`, `claimed` true (from `/identity`) |
| 1. `GET /myplex/account` | JSON root is `MyPlex`. `mappingState` `mapped`, `mappingError` empty string, `signInState` `ok`, `publicAddress` 203.0.113.9, `publicPort` 32400, `privateAddress` 10.0.0.5, `privatePort` 32400, `subscriptionActive` true, `subscriptionState` `Active`. No `mappingErrorMessage` attribute. `reachabilityPending` true at 14:14:36 and false at 14:16:08. XML form has the same attributes, booleans as `0`/`1` |
| 2. `GET /:/prefs` | 187 settings under `MediaContainer.Setting[]` |
| 3. plex.tv resources | One resource. `publicAddress` 203.0.113.9, `publicAddressMatches` true, `httpsRequired` false, `relay` false, `dnsRebindingProtection` false, `natLoopbackSupported` true, `presence` true, five connections |
| 4. Interfaces | WireGuard is `VPS`, type 53, no default route. Only IPv4 default route is `Ethernet` through 10.0.0.1. No IPv6 default route |
| 5. STUN | Both servers return 203.0.113.9. Local address 10.0.0.5 (`Ethernet`), port preserved. Cloudflare timed out once in three attempts |
| 6. Gateway | NETGEAR RAX50 at 10.0.0.1. No NAT-PMP answer in 2 s, twice. UPnP answers IGD:1 only, WAN 203.0.113.9, five port mappings, none to this machine |
| 7. Listeners | Plex on `[::]:32400` and `127.0.0.1:32401`. SlipStream v1 on `[::]:8080`. Nothing on 8484, 443, 80, 3000 |
| 8. Firewall | `INetFwPolicy2` works for this user. `CurrentProfileTypes` 6 (Private and Public). Each profile on, blocks inbound by default, not block-all. No rule names port 32400 or 8484 |
| 9. DNS | Both names resolve to 5.161.67.8 from the home resolver (10.0.0.1) and 1.1.1.1. No AAAA |
| 10. Dial | Both names connect to 5.161.67.8:443 from 10.0.0.5, TLS 1.3, valid chain, HTTP 200 |
| 11. SlipStream Address | No v2 install. v1 config.yaml empty |

## Plex mapping state

- `/` uses `myPlexMappingState`, `myPlexSigninState`. `/myplex/account` uses `mappingState`, `mappingError`, `reachabilityPending`, `signInState`, `publicAddress`, `publicPort`, `privateAddress`, `privatePort`.
- `mappingState` was `mapped` in both passes, including while `reachabilityPending` was true. No failing value seen.
- `mapped` is not evidence UPnP made the mapping: `ManualPortMappingMode` is true and the router's UPnP table has no entry for 32400. Nothing tested 203.0.113.9:32400 from the internet.

## Plex pref ids

| id | type | default | value | hidden |
| --- | --- | --- | --- | --- |
| `customConnections` | text | empty | `http://5.161.67.8:32400,https://plex.atassi.org:443` | no |
| `PublishServerOnPlexOnlineKey` | bool | false | true | yes |
| `ManualPortMappingMode` | bool | false | true | yes |
| `ManualPortMappingPort` | int | 32400 | 32400 | yes |
| `LastAutomaticMappedPort` | int | 0 | 18529 | yes |
| `secureConnections` | int | 1 | 1 | no |
| `EnableIPv6` | bool | true | true | yes |
| `IPNetworkType` | text | `dualstack` | `dualstack` | no |
| `RelayEnabled` | bool | true | false | no |
| `TreatWanIpAsLocal` | bool | true | false | no |
| `allowedNetworks` | text | empty | empty | no |
| `LanNetworksBandwidth` | text | empty | empty | no |
| `PreferredNetworkInterface` | text | empty | empty | no |
| `customCertificateDomain`/`Path`/`Key` | text | empty | empty | no |
| `WanPerStreamMaxUploadRate` | int | 0 | 0 | yes |
| `WanTotalMaxUploadRate` | int | 0 | 0 | yes |
| `WanPerUserStreamCount` | int | 0 | 0 | no |

## plex.tv connections

`publicAddress` is the home WAN (203.0.113.9), not the VPS (5.161.67.8).

| protocol | address | port | uri | local | relay | IPv6 |
| --- | --- | --- | --- | --- | --- | --- |
| https | 5.161.67.8 | 32400 | `https://5-161-67-8.<hash>.plex.direct:32400` | false | false | false |
| https | plex.atassi.org | 443 | `https://plex.atassi.org:443` | false | false | false |
| https | 10.0.0.5 | 32400 | `https://10-0-0-5.<hash>.plex.direct:32400` | true | false | false |
| https | fd72::5 | 32400 | `https://fd72-0-0-0-0-0-0-5.<hash>.plex.direct:32400` | true | false | true |
| https | 203.0.113.9 | 32400 | `https://203-0-113-9.<hash>.plex.direct:32400` | false | false | false |

- The custom URL connection's uri is `https://plex.atassi.org:443` with the explicit port.
- Custom URL `http://5.161.67.8:32400` comes back as `https` on a plex.direct name. Match custom URLs to connections by address and port, not by string.
- The one IPv6 connection is a ULA marked `local`. No global IPv6 connection.

## Interfaces

| Name | Description | InterfaceType | ComponentID | Go flags | Addresses |
| --- | --- | --- | --- | --- | --- |
| VPS | WireGuard Tunnel | 53 | `WireGuard` | up, running | 10.8.0.2/24, link-local |
| Tailscale | Tailscale Tunnel | 53 | `Wintun` | up, running | 169.254.83.107/16, link-local |
| Ethernet | Intel I225-V | 6 | PCI id | up, broadcast, multicast, running | 10.0.0.5/24, fd72::/64 ULAs, link-local |
| vEthernet (Default Switch) | Hyper-V | 6 | empty | up, broadcast, multicast, running | 172.23.224.1/20 |
| vEthernet (FSE HostVnic) | Hyper-V container | 6 | empty | not listed by Go | none |
| Wi-Fi | Intel AX200 | 71 | PCI id | broadcast, multicast | 169.254.193.193/16 |

- WireGuard is not the default route (routes only 10.8.0.0/24). So the class is not ThroughVPN.
- Type 53 alone doesn't identify WireGuard; Tailscale reports it too. Go doesn't set `pointtopoint` on either tunnel. Tunnel detection on Windows must use IfType 53 (IF_TYPE_PROP_VIRTUAL) via GetAdaptersAddresses, and the egress must come from the STUN socket's local address, not interface metric (both tunnels have metric 5, Ethernet 25).

## Router

- NAT-PMP: no answer. UPnP IGD:1 only, at `http://10.0.0.1:56688/Public_UPNP_gatedesc.xml`, server `Linux/2.6.12, UPnP/1.0, NETGEAR-UPNP/1.0`. IGD:2 no answer.
- WAN 203.0.113.9, `Connected`, matches STUN: class Mappable (UPnP answers and WAN == STUN). Inferred no double NAT.
- Exposes both `WANIPConnection:1` and `WANPPPConnection:1` with the same data. A reader querying both sees each mapping twice: dedupe.
- Mapping entries 0-4 are UDP Parsec forwards to 10.0.0.15. Index 5 ends the list with HTTP 500 / UPnP error 713.
- A manual forward set in router pages may not appear in the UPnP table (inferred). Empty does not prove no forward. (Here Plex's 32400 manual forward is invisible to UPnP.)
- A Hue bridge at 10.0.0.18 answers IGD searches with `upnp:rootdevice`. A reader must check `ST` and that the response comes from the gateway address.

## Firewall (INetFwPolicy2, unelevated)

- Works without elevation. 1,013 rules.
- Plex: four enabled allow rules by program path (`Plex Media Server.exe` etc.), protocol 256 (any), all profiles (2147483647), any remote address, EdgeTraversal true, LocalPorts empty.
- SlipStream v1: four enabled allow rules, profiles 6, any remote, LocalPorts `*`, program paths `C:\users\jacks\appdata\local\slipstream\slipstream.exe` and `C:\program files\slipstream\slipstream.exe`. EdgeTraversal false.
- A v2 binary at another path has no allow rule; inbound blocks by default (inferred).
- 32 enabled rules have no program, service or port limit ("allow everything" on paper): 9 carry `LocalAppPackageId`, 23 have `LocalUserOwner`. They are Store/system app rules; a reader must not treat them as open ports.
- Profile differs per interface: `VPS` Public; `Ethernet`, `Tailscale` Private.
- `Get-NetFirewallPortFilter` unelevated: "Access is denied" partway; 268 rules lack port filters including all Plex/SlipStream rules. Use COM, not cmdlets.

## DNS

No split horizon: both names return 5.161.67.8 (TTL 300) from router and 1.1.1.1. `fec0:0:0:ffff::1..3` site-local placeholders appear as IPv6 DNS servers on six interfaces (Windows default); don't report them as configured.

## Certificates

| Name | Issuer | Expires (UTC) |
| --- | --- | --- |
| slipstream.atassi.org | Let's Encrypt YE1 | 2027-01-14 11:10:20 |
| plex.atassi.org | Let's Encrypt YE2 | 2027-01-14 13:10:21 |

Both answers carry `Via: 1.1 Caddy`, no `Server` header. `/identity` on plex.atassi.org returns this server's `machineIdentifier`, so the VPS reaches this Plex.
