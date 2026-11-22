# A restore is staged, then written once, and keeps what the Nodes hold

This amends ADR 0018 and ADR 0019 where building Backups and restore showed them wrong or silent. Every restore, from the first-run page, `slipstream restore` or the Docker variables, is staged first. The Backup is decrypted into `.restore` files beside the live ones, checked, and summarised for the Admin, and nothing live changes until they confirm. Confirming writes one record, `lifecycle.json`, and the next start swaps the files and runs the restore's Write once, in one transaction that the restore's id stamps. A start stopped anywhere finishes at the next one, and never writes twice. Decided in [Backups and restore](https://github.com/jatassi/SlipStream/issues/174), building on [Config backup and restore](https://github.com/jatassi/SlipStream/issues/91).

What it amends in ADR 0018:

- **The archive** holds library.db, library.key, `backup.key` and `backups.json`. Nothing else goes in.
- **The passphrase** lives in `backup.key` beside library.key, sealed with it, never in the database. SlipStream generates it and never takes one typed. It is shown again to a session that confirmed its PIN, never to the API key, so a leaked API key can't read it.
- **A newer version backs up too**, before its migrations, as the pre-migration Backup of ADR 0015.

What it amends in ADR 0019:

- **A new machine keeps one Controller row.** The restored Controller row takes this machine's key, and the old key is recorded as superseded with a proof minted from the Backup's fence secret. If this machine's key already holds a row, as when a Worker the Backup lists is restored as the Controller, that row is removed. The new machine then draws a fresh fence secret, so a leaked Backup of the old Controller can't fence the new one.
- **Fencing is cleared.** A restored library starts unfenced. A library that was fenced also gets a fresh fence secret, since its Workers carry proofs naming the old one.
- **Every Node keeps every torrent** until the Controller adopts its Holdings. A torrent moves only once the Node that holds it reports it lacks it.
- **In-flight work pauses and resumes** instead of being cancelled. Cancelling a Grab at downloading removes its torrent and risks a hit and run. A queue search that the Backup caught open is ended and runs again once Reconciling ends. A queued move is dropped, and the scan after the restore finds each file where it is.
- **A restore in place keeps credentials and Backups.** It takes a before-restore Backup, keeps this machine's Backups index, and carries over Accounts, PINs, passkeys and the API key by id, so a credential revoked after the Backup never returns. Workers removed after the Backup are removed again. A manual rollback replaces the API key instead. Every session is signed out.
- **The first-run page takes an upload only.** A path typed behind the Setup code alone would read any file on the machine.
- **Nothing is sent twice.** Messages due before the Backup was taken are given up, since the old Controller most likely sent them.
- **"force-drains" reads "writes off".**

## Considered Options

- **Restore straight into the live files.** Rejected. A wrong passphrase, a damaged file or a full disk would leave a half-written library, and nothing could show the Admin what they are about to restore.
- **Run the Write at confirm, on the running Node.** Rejected. The live library is open, and a crash between the swap and the Write would leave a library that never began Reconciling.
- **Let every Write step be idempotent instead of stamping the restore.** Rejected. Giving the Controller row the new key can't run twice, and every new step would need its own proof.
- **Keep the restored Backup's credentials in place.** Rejected. A key or passkey revoked after the Backup, perhaps because it leaked, would work again.

## Consequences

- The lifecycle record is the one lock for a restore, an update and a Handover. A claim and a join refuse with `lifecycle_busy` while it exists.
- A Backup the first-run page restores is kept, listed as restored, in the Backups folder.
- On a new machine the restored index lists no Backup, since none of the old machine's files is here. Its copies on other Nodes are found again by the Destinations work.
- Each Reconciling asks every Node for a Holdings round it was never asked for, because a Worker sends each round once while it runs.
- History records Backup restored, and Controller replaced on a new machine, once per restore, and Reconciling finished when it ends. A fenced Controller records nothing. The Superseded page is the signal.
