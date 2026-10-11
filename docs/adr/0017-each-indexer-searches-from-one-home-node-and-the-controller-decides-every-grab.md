# Each Indexer searches from one home Node and the Controller decides every Grab

An Indexer is one way to search, and a Tracker is the site a torrent seeds to. They are separate, because a Tracker's seeding goals, hit and run state and Tracker routing outlive any Indexer that reaches it. Every query, login, RSS fetch, Test and .torrent fetch for an Indexer runs on one Searching Node, its home, which defaults to the Controller. The home Node only speaks the site's protocol and returns normalized Releases. The Controller parses them, judges Coverage, checks the Blocklist, ranks them per Target and decides each Grab, for automatic search, RSS sync and manual search alike. Credentials, cookies, rate-limit counters and health live on the Controller, and each cookie is tagged with the Node that earned it. When the home Node is offline past the grace period, an Indexer reached by an API key or passkey fails over to another Searching Node. One reached through a login session waits unless the admin allows failover. Decided in [Search, RSS sync and grab across Nodes](https://github.com/jatassi/SlipStream/issues/87#issuecomment-6110599137), Q1, Q4, Q5 and Q7.

## Considered Options

- **Fan each query out to every Searching Node.** Rejected. Private-tracker sessions and Cloudflare clearance are tied to the IP that earned them, so the other Nodes would trip logins and captchas, and per-site rate limits would multiply.
- **Let each query float to any free Searching Node.** Rejected for the same reason. It also scatters a site's request spacing across Nodes.
- **Decide on the Searching Node.** Rejected. Decisions read the library database, which stays on the Controller (ADR 0003).
- **One object for a Tracker and its Indexer.** Rejected. A generic Torznab feed reaches many Trackers, and deleting or swapping an Indexer would orphan live seeding obligations.

## Consequences

- A Worker wipe loses no session, limit or health state.
- A home Node's public IP is part of an Indexer's identity. Moving it to another Node may force a new login or challenge.
- A Challenge solver is configured per Searching Node, not per Indexer.
- A new Tracker routes by default to the built-in client on its Indexer's home Node when that Node downloads.
- Re-pairing a machine after the single Relay Node dies needs a rule for Indexers homed on its old Node record.
