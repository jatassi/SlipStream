# Workers hold no durable state and run in lockstep

The Controller holds every durable thing the user configures or sees, including the intent for each Role, such as which torrents a Worker seeds and to what goals. A Worker holds only its key, its pairing record, Role-local runtime state and an outbox of results. It can be wiped and re-paired with nothing lost. A rebuilt downloading Worker re-adds its torrents from the Controller's intent and rechecks the files. Every Worker runs the Controller's exact version. A Worker on another version only updates, and the handshake and update path stay frozen across versions so it always can. The freeze binds from the first release someone other than the maintainer installs; before that, a handshake change costs only a manual re-pair. Until it updates, it behaves as if the Controller were unreachable. It finishes in-flight jobs and keeps seeding from its last copy of its Roles' intent, but starts nothing new. So lockstep costs no ratio on private trackers.

Nodes trust each other by public keys pinned at Pairing, not by certificates. Either side may open the connection, chosen at Pairing from what each can reach, and one channel then carries calls both ways. This lets a Controller behind CGNAT dial out to its seedbox. Every subsystem whose placement changes the outcome is a Role, and any Role can sit on any number of Nodes, except Relaying, which [ADR 0012](0012-the-relay-is-a-paired-worker-forwarding-streams-over-the-node-link.md) limits to one. Work that reads the library database on every call stays on the Controller. Decided in [Node topology](https://github.com/jatassi/SlipStream/issues/12#issuecomment-5988811643), Q2, Q3, Q7, Q8, Q11, Q16 and Q21.

## Considered Options

- **Workers with a replica of the library database.** Rejected. Replication and conflict handling buy autonomy that only matters for new decisions, and new decisions can wait for the Controller.
- **A compatibility window or negotiated protocol versions.** Rejected. Every version pair needs its own offline test run, and a Worker without durable state has nothing to protect by staying on an old version.
- **Mutual TLS with the Controller as a certificate authority.** Rejected. Certificates add lifetimes and rotation, and removing a pinned key already revokes a Node at once.
- **The Controller as a Role with automatic failover.** Rejected. Failover needs consensus and replicated databases. The Controller moves only by an explicit Handover instead.

## Consequences

- A Worker can be rebuilt from scratch. Backup and restore covers only the Controller.
- Changing the handshake or update path breaks every older Worker in the field, so those two stay fixed across releases.
- The offline suite fakes the Node link twice. An in-memory link with fault injection drives simulation tests, and real Node processes on loopback behind a fault proxy drive the end-to-end suite.
- A mismatched Relay Node or Serving Worker keeps forwarding and proxying during an update, because forwarding isn't a job. Decided in the [coherence pass](https://github.com/jatassi/SlipStream/pull/112), Q14.
- A Role leaves a Node by draining or by a write off. A draining Node finishes the work it holds for the Role and takes nothing new. A write off gives that work up on a live or an offline Node and keeps its files. Decided in [Nodes, Pairing and Roles](https://github.com/jatassi/SlipStream/issues/140).
