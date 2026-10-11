# The relay is a paired Worker forwarding streams over the Node link

A home network behind CGNAT or double NAT can't accept inbound connections. In that case the admin rents a VPS, installs SlipStream on it, pairs it with a Join Code, and gives it the Relaying Role. Home Nodes already dial out to the relay over the Node link. The relay listens on public ports and forwards each incoming TCP or UDP stream down that link to the Node and port it maps to: the Serving Node for the UI and portal, the Media Server, or any other port the admin names. The admin brings their own domain and points it at the relay. TLS for the UI and portal passes through by SNI, so the certificate and key stay on the home Serving Node, which gets them from ACME with TLS-ALPN-01 through the same path. Forwarded Media Server traffic is not changed: SlipStream doesn't try to make remote viewers look remote or local. A SlipStream has at most one Relay Node, and assigning Relaying to a second Node is blocked. Replacing the relay means pairing a new Node and moving Relaying onto it. The forwards come back from the Controller, the certificate stays at home, and the admin points the domain at the new relay by hand. A home Node with a public IP can be the relay. Decided in [Relay Role and reachability diagnostic](https://github.com/jatassi/SlipStream/issues/16) and [More than one Relay Node](https://github.com/jatassi/SlipStream/issues/92). Research in [CGNAT, remote access and relay techniques](https://github.com/jatassi/SlipStream/issues/15).

## Considered Options

- **A userspace WireGuard tunnel to the VPS.** Rejected. It is a layer 3 network with its own keys and addressing, alongside the Node link that already authenticates both ends with the keys pinned at Pairing.
- **A separate frp-style forwarder.** Rejected. It is a second tunnel stack to pair, run and fake in offline mode.
- **End TLS on the relay.** Rejected. It puts keys and plaintext on rented hardware.
- **More than one Relay Node, as a failover standby or one per region.** Rejected. Every stream starts at home, so a regional relay lengthens one leg by about what it saves on the other, and a second relay adds no home upload. Fast failover would need SlipStream to rewrite DNS on the admin's domain, which is the automatic failover that ADR 0003 rejects. One relay already forwards to Nodes at any number of sites.
- **SlipStream editing DNS through a provider's API.** Rejected. It needs provider credentials and one integration and offline fake per provider, all to speed up a rare manual swap.
- **Hosted tunnels or a hosted subdomain.** Rejected. They limit bandwidth, ports or video, and they break self-hosting and offline mode.

## Consequences

- The relay holds no keys for the UI and portal, and it sees only SNI and ciphertext for them.
- The relay's fake in offline mode is a loopback Relay Node over the faked Node link.
- Media Servers see relayed viewers as coming from the home network.
- A dead relay turns every exposure through it Fixable, not Needs a relay, and the admin is notified after a grace period.
- Renewing the certificate over TLS-ALPN-01 fails until DNS points at a new relay, but the current certificate keeps serving until it expires.
- A planned move lets the old relay drain while the Controller hands every Worker the new relay's address and key. A Worker that reaches the Controller only through a relay that dies unplanned has to re-pair, which loses nothing.
