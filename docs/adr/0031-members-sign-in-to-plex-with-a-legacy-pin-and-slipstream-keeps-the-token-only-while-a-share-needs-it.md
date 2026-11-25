# Members sign in to Plex with a legacy PIN under their own client identifier, and SlipStream keeps the token sealed only while a share needs it

A Member offered a Plex server signs in to Plex once from SlipStream. SlipStream then knows which Plex account to share with, and it can accept the share for the Member when the Admin presses Share. That sign-in is plex.tv's legacy PIN: a strong PIN with no device key, polled by the Controller under a client identifier that SlipStream makes at the Member's first PIN and reuses for every later one. It is never the Admin's client identifier, so the Admin's device never appears on a Member's Plex account. The token is sealed per Member in library.db and opened only on the Controller, to accept a share. The Write that leaves the Member no access waiting for a share clears it, and SlipStream signs it out after that Write commits. The Admin's own sign-in stays the JWT PIN with the device key that S35 (Media Servers) built. Decided in [Member access to Media Servers](https://github.com/jatassi/SlipStream/issues/168).

## Considered Options

- **The JWT flow for each Member.** Rejected. A Plex JWT lasts 7 days and renews only with the device key that made its PIN. The Admin may press Share days after the Member signed in, so each Member would need a device key and a renewal task of their own.
- **The Admin's client identifier for the Member's PIN.** Rejected. The Admin's device would then appear on every Member's Plex account, and a Member's sign-in would run under the device that holds the Admin's key.

## Consequences

- SlipStream keeps a Member's token as long as a share waits for them, with no limit, because no share can be accepted without it.
- How long plex.tv honours a legacy token, and whether signing it out revokes only that token, is measured by the recorded Plex run on 3 December 2026. Its answer lands here.
