# Releases are signed file manifests the Controller rolls out and rolls back

Each release carries a `manifest.json` listing every file for every OS and arch (the binary, ffmpeg, ffprobe and `dovi_tool`) with its size and SHA-256, plus the version, channel and release notes. Release CI signs it with minisign (Ed25519), and the public keys are compiled into the binary. The Controller alone finds releases, on GitHub Releases through the release feed entry in `Endpoints`. It trusts the channel written inside the signed manifest, never GitHub's prerelease flag. It fetches the files for itself and for every platform its Workers run, keeps them for the current and previous version, and relays the signed manifest bytes and files to Workers unchanged. Each Worker checks them against its own embedded keys, so a compromised Controller can't push a build release CI didn't sign. Workers converge to the Controller's exact version in either direction. The handshake, the update service, the manifest format and this verification rule are the frozen update path from ADR 0003, so the manifest only ever gains fields. Every install except Docker replaces itself in place, and Docker installs show the tag to pull.

An update runs as stored, resumable phases: stage the files, check free disk, drain the Controller for up to ten minutes, swap and pass a health trial, update every Worker in parallel, then done. The new binary takes the migration-time Backup before migrating. If its health trial fails twice, it restores that Backup and puts the old binary back. The old binary returns on the untouched database only when the Backup itself failed, since nothing then migrated. A manual rollback, from the dashboard or a host command, restores the old binary plus the Backup taken before that update, losing changes made since, and starts in Reconciling like any restore (ADR 0019). Decided in [Packaging, the updater and the update dashboard](https://github.com/jatassi/SlipStream/issues/93), Q7, Q9 to Q13.

## Considered Options

- **cosign keyless (Sigstore).** Rejected. Verifying needs Fulcio roots and a certificate identity, which is heavy for Workers with no internet.
- **TUF.** Rejected. Its roles and metadata defend against freeze attacks, which matter little for a self-hosted app.
- **One archive per platform.** Rejected. Once frozen, every Worker would download about 60 MB of unchanged ffmpeg on every release.
- **Workers fetching from GitHub.** Rejected. It contradicts ADR 0003, and an outage at the release host after the Controller swaps would strand Workers.
- **Updating through apt, yum or the MSI.** Rejected. Workers would update outside the Controller, and it needs signed package repositories.
- **goose down migrations for rollback.** Rejected. They double the migration work and would never be tested.
- **A test-only build that trusts a test key.** Rejected for ADR 0014's harness, which runs the release binary. A Node instead accepts one extra release key from its own local config or env, never from the UI or the Node link.

## Consequences

- A compromised signing key has no in-band recovery. Keys rotate by shipping the next key in an earlier release. Every retired public key stays embedded for verification, so a Worker offline across a rotation can still update, and a key is dropped only after a compromise. Admins reinstall by hand after a compromise. Decided in the [coherence pass](COHERENCE_PR), Q19.
- CI parses each new manifest with the oldest supported parser, and an end-to-end suite updates the first release and the previous one to the current build on loopback. `buf breaking` alone checks only the proto schema.
- Breaking the frozen path by mistake strands every older Worker, which must then be reinstalled and re-paired.
- Rollback depends on the migration-time Backup, so a Backup failure blocks the update instead of risking the database.
- The fake release source is a loopback server behind the `Endpoints` entry, signed with the harness's extra key, serving real builds of the current commit with a bumped version.
