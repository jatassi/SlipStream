# A four-digit PIN signs in anywhere, and only Admins recover Accounts

Everyone, Admins and Members alike, has one Account signed in by its email. A 4-digit PIN or a passkey each signs in alone, from any address, behind per-IP rate limiting. Passkeys are optional, and each address SlipStream is reached at is its own relying party. PIN failures are stored per Account: five bring a 15-minute wait, and ten consecutive disable the PIN until the person signs in by passkey or reset link and sets a new one. Recovery is a reset link that an Admin issues. There is no self-service path, even though SMTP is available. The last Admin recovers through a host command that prints a reset link.

The PIN is the everyday credential, quick to type on a TV, and the maintainer chose that ease over binding it to known devices. The stored hard stop bounds the guessing odds instead. A stranger gets ten guesses at an Account before its PIN is disabled, about a 0.1% chance. Decided in [Identity, PIN and passkey, admin permission, API key](https://github.com/jatassi/SlipStream/issues/23#issuecomment-6028813763), Q2, Q5, Q6, Q9, Q15 and Q17.

## Considered Options

- **PIN only on a device that has signed in before, with passkey or emailed link for new devices.** Rejected. It takes the PIN away exactly when a person is on a new screen.
- **Passkey and PIN on every sign-in, or passkey only.** Rejected. A passkey already verifies the person, and requiring one shuts out every plain-HTTP address.
- **v1's in-memory lockout of 5 fails, then waits doubling to an hour.** Rejected. A restart clears it, and it allows about 120 guesses a day, a 50% chance at a PIN in about 42 days.
- **A self-service emailed reset link.** Rejected. Whoever controls a person's inbox would control their Account, and household installs have an Admin close at hand.
- **Counting only failures from unknown devices toward the hard stop.** Rejected. It adds device tracking to soften an attack on small installs, and a passkey already sidesteps it.
- **One relying party ID for the whole install.** Rejected. The maintainer preferred a passkey at every HTTPS address a person uses over passkeys at one domain only.

## Consequences

- Anyone who knows an email can disable that Account's PIN with ten wrong guesses. The owner is emailed, passkey sign-in still works, and an Admin issues a reset link.
- A person who loses both credentials depends on an Admin, and the last Admin depends on shell access to the host.
- A person who uses several addresses holds several passkeys, and the Account's passkey list shows which address each belongs to.
- Admin actions still need a re-confirm by PIN or passkey within 15 minutes, and that re-confirm counts toward lockout.
- WebAuthn needs a domain as the relying party ID, so no IP address can hold a passkey, over HTTPS or not. SlipStream never offers a passkey at an IP address, and Signup and Reset links always use the Controller's configured name. Decided in [verify-slipstream skill design](https://github.com/jatassi/SlipStream/issues/27), Q37.
- The Controller's configured name defaults to the relay's domain when Remote access has one, else the hostname the Admin confirms at first run. No Signup link can be issued until it is set, and it travels with a Backup and a Handover. Decided in the [coherence pass](https://github.com/jatassi/SlipStream/pull/112), Q15.
