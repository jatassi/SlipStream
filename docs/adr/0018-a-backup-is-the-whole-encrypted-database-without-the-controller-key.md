# A Backup is the whole encrypted database without the Controller's key

A Backup is the Controller's whole library database: Works, history, file records, each Role's intent, settings, Accounts and every secret, PIN hashes and passkey credentials included. Nothing else goes in. Certificates, caches, artwork, Speech Maps, logs and the Controller's own torrent resume data are rebuilt after a restore. The file is one tar holding a manifest and a `VACUUM INTO` snapshot, compressed with zstd and always encrypted with an age passphrase. The Controller generates that passphrase at first run, shows it once for the admin to save, and keeps it so scheduled Backups run unattended. A Backup never holds the Controller's private key. A binary restores Backups from its own version or any older one and migrates them forward, and refuses newer ones. Handover ships the same archive over the Node link into the same restore code. Decided in [Config backup and restore](https://github.com/jatassi/SlipStream/issues/91), Q1 to Q9 and Q19.

## Considered Options

- **A config-only backup** of settings, Targets and Profiles. Rejected. It can't become a Controller, and without each Role's intent nobody knows a torrent's seeding goals, which risks hit and runs.
- **Plaintext Backups**, as Sonarr and Radarr write. Rejected. Copies leave the Controller as downloads and to other Nodes, possibly a rented relay VPS, and the global API key alone is full admin access.
- **Secrets left out.** Rejected. Every integration would break after a restore, and every Member would re-enroll.
- **An age public key**, so the Controller can't read its own Backups. Rejected. The Controller then can't verify what it wrote, and a restore needs a long pasted key.
- **Carrying the Controller's key** so Workers reconnect without re-pairing. Rejected. A leaked Backup could steer Workers, and two machines would share one identity, which ADR 0019 needs to avoid.
- **Restoring only the same version.** Rejected. A dead machine is usually rebuilt from the latest installer.

## Consequences

- The Controller keeps its Role-local runtime state in a separate file it can lose, like a Worker (ADR 0003), never in the library database. Any setting that isn't specific to one machine lives in the database.
- Speech Maps are stored outside the database so they don't bloat every Backup.
- If credential encryption at rest (P-350) uses a key file, that key goes inside the Backup's encrypted layer.
- A lost passphrase makes every Backup useless, so a health warning stays until the admin confirms they saved it.
- A Backup runs before every migration at startup, which covers updates the self-updater never sees and satisfies P-270. Rollback is the old binary plus that version's pre-migration Backup (ADR 0015).
- Backup destinations are folders on paired Nodes, reached by Transfer (ADR 0004), so Backups add no external boundary to offline mode (ADR 0014).
- CI restores a fixture Backup from every released version.
