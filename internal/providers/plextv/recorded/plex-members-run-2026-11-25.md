# Plex Member sharing: recorded run, 2026-11-25

The recorded run that S43 (Member access to Media Servers) (jatassi/SlipStream#168, "The recorded Plex run") puts first in its build. It ran from Jackson's Mac against real plex.tv with five throwaway accounts (A Admin, B to E Members, plus-addresses on Jackson's Gmail, none with Plex Pass) and a throwaway Plex Media Server 1.43.5.11079 in Colima (container `recorded-run-pms`, claimed by A, one section "Member access test" of two generated clips). Jackson's own Plex account and server were not touched. Requests sent `X-Plex-Product: Recorded run`.

The redacted log (468 lines) stays on the Mac at `~/.slipstream-recorded-runs/plex-members/out/plex-members-run-2026-11-25.jsonl`; it was not published. The recorder is `~/.slipstream-recorded-runs/plex-members/tool/main.go` (source and runbook in /mnt/project-files/member-media-access/plex-run/, live edits on the Mac: section agent `com.plexapp.agents.none`, a `prestep` command, URL-encoded redaction, q2 reads `invites/owned/pending`).

## Results

| Spec question | Result | Evidence |
|---|---|---|
| 1. Admin JWT on the sharing routes, and clients.plex.tv | Pass on both hosts | `/api/servers/{machineId}`, `/api/v2/servers`, `shared_servers`, `/api/users`, `/api/home/users`, `invites/requested`, sharings `owned/accepted` and `owned/pending`: 200 on plex.tv and clients.plex.tv. Share POST on plex.tv: 200. `/api/v2/friends`: 410 on both. Stop rule did not fire |
| 2. Which call lists a pending share with its id | `owned/pending` | clients.plex.tv `owned/pending` has the share id on the top-level entry; `invites/owned/pending` also lists it. `invites/requested` lists the invite without the share id. `shared_servers` and `users` don't list pending shares |
| 3. Legacy PIN without JWK | Works on both hosts | clients.plex.tv and plex.tv PINs (strong, no JWK) each issued a 20-character legacy token. Two tokens for one account coexist and both work on both hosts |
| 4. Share id = pending shared server id | Yes | Member accepts by that id with its own token: 204. Then it's in `owned/accepted` and `shared_servers`, and the Member sees the server |
| 5. Second share, share to a friend, share to others | Second refused | Second share to the same Member: 400 "You're already sharing this server with B. Please edit your existing share." Share to a plain friend with no share, and to a non-friend: 200 |
| 6. Plex Home invite | Needs Plex Pass | Invite of an existing account by the Admin, by username or email: 403 "Invite a user requires a Plex Pass". Managed users without Plex Pass are allowed up to the cap: 403 "You have reached the 15 person Home limit" (Admin plus 14). Share to a Home user, its accept, and whether leaving the Home ends shares were NOT measured (blocked by Plex Pass) |
| 7. DELETE shared_servers/{id} | Works before and after accept | Pending: DELETE 200, share gone, invite still listed; cancel invite 200 removes it. Accepted: DELETE 200, Member loses the server. Friendship survives the DELETE |
| 8. Legacy token eight days later, sign-out revokes only it | Due 2026-12-03 | `./rec q8` on or after 3 Dec 12:16 EST, then `./rec cleanup -all` |

Plain friends are made through community.plex.tv (`addFriend`, `acceptFriend`), not plex.tv.

## What this means for the build

- The API share ships. Share and accept use the routes above; look up `owned/pending` and `owned/accepted` before sharing, and treat the "already sharing" 400 as a share that exists.
- Remove of a waiting share is DELETE plus cancelling the invite.
- Add to Plex Home needs Plex Pass on the Admin's account; SlipStream shows Plex's refusal. Home-user share cases stay on Plex's documentation, marked unverified.

## Cleanup

Done: 0 shares, invites, Home users and friends on A; A's JWT and legacy token and C, D, E signed out; container, volumes and clips deleted, Colima stopped. B's two tokens kept for q8. After q8, Jackson deletes the five accounts.
