# An Exposure has one verdict and IPv6 counts only where it is offered

The Reachability diagnostic gives each Exposure one verdict: Reachable, Fixable or Needs a relay. Address family is a dimension of the checks, not of the verdict. IPv4 is the required family by default. The verdict is the worse of the required family's verdict and the result of every other family that something offers, such as an AAAA record or a Media Server publishing an IPv6 address. An offered family that fails makes the Exposure Fixable, because "make it work or stop offering it" always works. A family nobody offers never changes the verdict. So an Exposure reachable over IPv6 alone reads Needs a relay, with a neutral fact that IPv6 viewers reach it directly. IPv6 replaces the relay only when the admin turns on that Exposure's "IPv6 is enough" setting, which is off by default and makes IPv6 the required family. The Relay Node never dials a home Node over either family. Decided in [IPv6 reachability verdicts](https://github.com/jatassi/SlipStream/issues/115), Q1, Q2, Q4, Q10 and Q12. Research in [IPv6 remote access facts](https://github.com/jatassi/SlipStream/issues/114).

## Considered Options

- **One verdict per address family.** Rejected. It doubles the verdicts and the Remote access status sentence, and "IPv6: Needs a relay" reads as a problem when nothing offers IPv6.
- **Judging IPv4 alone.** Rejected. A broken AAAA record stalls or fails viewers whose clients prefer IPv6, and an IPv4-only diagnostic would still call that Exposure Reachable.
- **Treating working IPv6 as enough on its own.** Rejected. About 56% of users can't use IPv6, Roku apps can't at all, and Plex apps ignore IPv6 unless a per-app toggle is on. Inferring it would strand those viewers without telling the admin.
- **The relay dialing home over IPv6.** Rejected. It saves no latency, needs a home pinhole that many routers can't open, and exposes the home Node. The Node link stays the relay's only path home.
- **Shipping all of IPv6 with the diagnostic.** Rejected. Opening pinholes over PCP and UPnP IGDv2 has no maintained Go client, router support is patchy, and telling temporary from stable addresses needs per-OS code. That work serves only admins who turn on "IPv6 is enough".

## Consequences

- The Reachability diagnostic judges offered IPv6 from its first release: bind per family, the OS firewall per family, the Node's global addresses, AAAA records and IPv6 connect-back from the Relay Node. Where "unproven" lands follows the same rule for both families.
- A later spec, IPv6 direct reach, adds the "IPv6 is enough" setting, opening IPv6 pinholes under the same per-Node setting as IPv4 port mappings, STUN over IPv6 with NPTv6 detection, and per-OS temporary address detection. It blocks nothing.
- Remote access draws each Exposure as one route with a family tag ("IPv4 and IPv6", "IPv4" or "IPv6 only"). Each family's checks sit in the hop they belong to, and a check for a family nobody offers never colours a hop.
- For a relayed route, every AAAA record must point at the relay or be absent. An AAAA pointing at home fails even if it works, because Let's Encrypt validates over IPv6 first with no fallback for TLS-ALPN-01, and the route would bypass the relay. SlipStream still never edits DNS (ADR 0012).
- The Relay Node probes offered IPv6 addresses when it has IPv6 itself, and warns when a home service meant to be relayed answers directly over IPv6.
- A change in a Node's published IPv6 address or its IPv6 class reruns that Node's diagnostic. Temporary address changes never do.
- With "IPv6 is enough" on, the Player hop lists who is cut off: Roku, IPv4-only networks, Plex apps without the IPv6 toggle, and Plex servers without Plex Pass.
