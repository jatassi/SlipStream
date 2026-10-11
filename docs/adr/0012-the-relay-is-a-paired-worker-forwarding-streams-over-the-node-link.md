# The relay is a paired Worker forwarding streams over the Node link

A home network behind CGNAT or double NAT can't accept inbound connections. In that case the admin rents a VPS, installs SlipStream on it, pairs it with a Join Code, and gives it the Relaying Role. Home Nodes already dial out to the relay over the Node link. The relay listens on public ports and forwards each incoming TCP or UDP stream down that link to the Node and port it maps to: the Serving Node for the UI and portal, the Media Server, or any other port the admin names. The admin brings their own domain and points it at the relay. TLS for the UI and portal passes through by SNI, so the certificate and key stay on the home Serving Node, which gets them from ACME with TLS-ALPN-01 through the same path. Forwarded Media Server traffic is not changed: SlipStream doesn't try to make remote viewers look remote or local. A SlipStream has one Relay Node. Decided in [Relay Role and reachability diagnostic](https://github.com/jatassi/SlipStream/issues/16). Research in [CGNAT, remote access and relay techniques](https://github.com/jatassi/SlipStream/issues/15).

## Considered Options

- **A userspace WireGuard tunnel to the VPS.** Rejected. It is a layer 3 network with its own keys and addressing, alongside the Node link that already authenticates both ends with the keys pinned at Pairing.
- **A separate frp-style forwarder.** Rejected. It is a second tunnel stack to pair, run and fake in offline mode.
- **End TLS on the relay.** Rejected. It puts keys and plaintext on rented hardware.
- **Hosted tunnels or a hosted subdomain.** Rejected. They limit bandwidth, ports or video, and they break self-hosting and offline mode.

## Consequences

- The relay holds no keys for the UI and portal, and it sees only SNI and ciphertext for them.
- The relay's fake in offline mode is a loopback Relay Node over the faked Node link.
- Media Servers see relayed viewers as coming from the home network.
- More than one Relay Node waits for a later design.
