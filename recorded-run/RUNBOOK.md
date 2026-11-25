# Recorded Plex run for S43: Member sharing and Plex Home

This run checks what plex.tv does with the Admin's Plex JWT when the Admin shares a server with Members and invites them to a Plex Home. It answers questions 1 to 8 of "The recorded Plex run" in `member-media-access/spec.md`. It uses throwaway accounts A (Admin) and B to E (Members) and a throwaway Plex Media Server in Colima on Jackson's Mac. It never touches Jackson's own server, libraries, friends or Plex account.

A Claude Remote Control session on the Mac runs every command. Jackson does three things: he approves two sign-ins in private windows, he does the two manual pre-steps, and at the end he deletes the accounts.

## What's already on the Mac

Everything is in `~/.slipstream-recorded-runs/plex-members/` (dir 700):

- `accounts.json` (600) has one entry per label A to E: email, username, client identifier, and a legacy token from a strong PIN with no JWK on clients.plex.tv. Plex assigned the usernames, so always read them from this file.
- `signin.jsonl` logs those sign-ins. `export` redacts it and puts it at the top of the exported log.

The recorder adds `state.json` (600: A's device key, tokens, ids, labels), `plex-members-run.jsonl` (the redacted log), `media/clips/` and `out/`.

## Setup

```sh
mkdir -p ~/.slipstream-recorded-runs/plex-members/tool
cp <this folder>/recorder/main.go ~/.slipstream-recorded-runs/plex-members/tool/main.go
cd ~/.slipstream-recorded-runs/plex-members/tool
go build -o rec main.go        # standard library only, no go.mod needed
colima status || colima start
docker version
```

Every command below is `./rec <command>` from that folder. `./rec status` shows progress at any time. The default state dir is the one above, and `-state DIR` before the command changes it.

**For the driver:**

- Commands print each call as `step METHOD url -> status` and each finding as `finding name: {...}`, already redacted.
- `signin` polls for up to 30 minutes, so run it with `run_in_background`. Read the `>>> Ask Jackson` line from its output and pass that URL to Jackson.
- Most commands take seconds. `server` can take up to about 6 minutes.
- Every command adds to the log, so a rerun is safe but leaves extra lines. Only the share probes in q5 and q6 make new state on plex.tv when rerun.
- No request names SlipStream. Requests send `X-Plex-Product` and `X-Plex-Device-Name` as "Recorded run", and the container and server are named `recorded-run-pms`.

## The run, in order

| # | Command | Jackson does |
|---|---|---|
| 1 | `./rec init` | Nothing. This makes A's Ed25519 device key and a fresh client identifier for the JWT sign-in. |
| 2 | `./rec import` | Nothing. Reads B to E, plus A's legacy token as slot `AL`, from `accounts.json`, then reads `/api/v2/user` with each token on clients.plex.tv. If a label is skipped, check its field names. If the file has no issue time, pass `-token-at 2026-11-24T18:00:00Z`, since q8 counts from it. |
| 3 | `./rec signin A` (background) | **Approval 1.** Close every private window, open a new one, open the printed URL and sign in as **A** (A's email in `accounts.json`). Approve. The command finishes once plex.tv issues the JWT. |
| 4 | Manual pre-steps, below | Uses private windows for A and C. |
| 5 | `./rec media` | Nothing. With ffmpeg it makes two 6-second clips. Without it, it downloads two Blender trailers (CC BY 3.0). |
| 6 | `./rec claim`, then **at once** the printed `docker run` | Nothing. A claim token lasts 4 minutes. Wait for `docker logs -f recorded-run-pms` to show the server started, then press Ctrl-C. |
| 7 | `./rec server` | Nothing. This finds the server in A's resources and checks it is 1.43 or later. Then it makes the "Member access test" section (Other Videos, `/data`), scans it, and waits for plex.tv to list the section's plex.tv id. **The stop rule can fire here.** |
| 8 | `./rec signin B -host plex` (background) | **Approval 2.** Open a new private window and sign in as **B**. This is the plex.tv half of question 3, stored as slot `B2`. If plex.tv refuses the PIN, record the refusal (the finding is already logged) and go on. |
| 9 | `./rec precheck` | Nothing. C must show `matched: true` in `users` or `friends-v2`. D must show it in `invites-requested` or `home-users`. If not, redo the pre-step. |
| 10 | `./rec q1` | Nothing. **Stop rule.** |
| 11 | `./rec q2`, `q3`, `q4`, `q5`, `q6`, `q6-remove`, `q6-full`, `q7` | Nothing. |
| 12 | `./rec cleanup`, then the printed docker commands | Nothing. B's and B2's tokens stay for q8. |
| 13 | **On or after the date `./rec status` prints:** `./rec q8`, then `./rec cleanup -all` | Nothing. |
| 14 | `./rec export -date 2026-11-XX`, then the write-up below | Deletes accounts A, C, D and E after the export, and B after step 13. |

### Manual pre-steps (step 4)

Do these by hand before `precheck`. They set up C as an existing friend and D as an existing Home invitee.

1. **C becomes a plain friend of A, with no library shared.** In a private window signed in as A at app.plex.tv, send a friend request to C's username (from `accounts.json`). Close it, open a new private window signed in as C, and accept the request. The exact Plex Web path is unverified. Look for Friends or Add friend, never "Share libraries" or "Grant library access". If Plex Web offers only a library share, stop and record it with `./rec note precheck "..."` instead of sharing.
2. **D gets A's Plex Home invite and leaves it pending.** In a private window signed in as A, go to Settings > Plex Home (it may be called Manage Home) > Invite user, and enter D's username or email. **D must not accept it.** If Plex Web says Plex Home needs Plex Pass, record the exact text with `./rec note precheck "Home invite by hand: <text>"` and go on. q6 then records what the API answers.

## Stop rule

If plex.tv refuses A's JWT (401 or 403) on the section route (`GET plex.tv/api/servers/{machineId}`, in `server` or `q1`) or on the share routes (the `shared_servers` list or POST, in `q1`), the command writes a `stop-rule` finding and exits with code 3 and a `*** STOP RULE` banner. Then:

1. Run nothing else from q2 on.
2. Run `./rec cleanup -all` and the docker commands it prints.
3. Run `./rec export -date <today>` and write it up with the stop as the first result.

The spec then ships Share by hand instead and Mark as added, and the API share waits for a decision. A 404 or 5xx is not the stop rule. A 404 on a clients.plex.tv route is itself an answer to question 1.

## What each question reads

| Spec question | Command | Findings that answer it |
|---|---|---|
| 1. JWT on servers, `shared_servers` list, POST and DELETE, accepted shares, Home users, `invites/requested`. Does clients.plex.tv serve them? | `server`, `q1`, `q7` | `jwt-route` (each route on both hosts), `share` (POST on plex.tv), `delete-before-accept` (DELETE on clients.plex.tv first, then plex.tv), `plex-section` |
| 2. Which call lists a pending share with its id? | `q2` | `listing` entries where `has_share_id: true` and `matches[].tag` names the element |
| 3. Legacy PIN without JWK on clients.plex.tv, and on plex.tv | `import`, `signin B -host plex`, `q3` | `imported` (clients.plex.tv half, from the Mac's sign-in), `legacy-pin-create` and `token` for B2, `legacy-token`, `b-on-both-hosts` |
| 4. Is the share's id the pending invite's shared server id? | `q4` | `pending-id.equal`, `accept` (`is_share_id`, status 204), `pending-after-accept`, `member-sees-server` |
| 5. Second share, share to an existing friend, share to a Home user | `q5`, `q6` | `share` for `q5:second-share-B`, `q5:share-C-friend-no-share`, `q5:share-D-pending-home-invitee` and `q6:share-E-home` |
| 6. Home invite with the JWT, in the pending list, same accept, Plex Pass, Home full, removal ends shares | `q6`, `q6-remove`, `q6-full` | `admin-plex-pass`, `home-invite`, the `listing`s for `pending-v2 on clients(E)` and `(D)`, `home-share-pending`, `accept`, `accept-home-v1`, `home-remove`, `member-sees-server` after removal, `home-full`, `home-invite` when full |
| 7. Does DELETE keep the friendship, and does it work before accept? Does cancelling the invite remove a pending share? | `q7` | `delete-before-accept`, the `listing`s after it, `cancel-invite` (only if the share survived), `C-still-friend` and `B-still-friend` listings, `delete-after-accept` |
| 8. Does B's legacy token work eight days later, and does sign-out revoke only it? | `q8` | `token-after-days` (B and B2), `signout`, `signout-scope` (`B_after` 401 and `B2_after` 200 means only B was revoked) |

`q6-full` makes managed Home users ("Member access filler N") until plex.tv refuses one, then tries to invite C, then deletes the managed users. If the first managed user is refused, the Home could not be filled, and the refusal is the answer (likely Plex Pass). Say so in the write-up.

## Where each call comes from

Tags: **[M]** measured by an earlier recorded run (#74 on 2026-10-11 with legacy tokens, the JWT run on 2026-11-14, or the Mac's `signin.jsonl`). **[R]** read in python-plexapi or Wizarr source, per `how/04-protocols.md`, but never called by us. **[G]** guessed from memory with no source read for this run. Fix [G] lines live if they miss.

| Call | Host | Body or parameters | Tag |
|---|---|---|---|
| JWT PIN `POST /api/v2/pins`, poll `GET /api/v2/pins/{id}?deviceJWT=` | clients.plex.tv | `{jwk, strong: true}`, device JWT with `aud iss iat exp` | [M] JWT run |
| Legacy PIN `POST /api/v2/pins?strong=true`, poll without deviceJWT | clients.plex.tv | no body, no JWK | [M] Mac `signin.jsonl` |
| Same on plex.tv | plex.tv | same | [M] #74, a month old |
| `GET /api/v2/user`, `GET /api/v2/resources` | clients.plex.tv (q3 and q8 also try plex.tv) | | [M] |
| `DELETE /api/v2/users/signout` | clients.plex.tv, then plex.tv on 404 | | [M] #74 |
| `GET /api/claim/token.json` with A's JWT | plex.tv | answers `{token}` | [G]. Fallback: https://plex.tv/claim in A's window |
| PMS `POST /library/sections?name&type=movie&agent=tv.plex.agents.none&scanner=Plex Video Files Scanner&language=xn&location=/data` | 127.0.0.1:32400, with the `resources` token | | [G]. Fallback: make it in http://localhost:32400/web, then rerun `server` |
| `GET /api/servers/{machineId}` (XML `Section id key title`) | plex.tv, plus clients.plex.tv in q1 | | [M] #74 with a legacy token, never with a JWT |
| `GET /api/v2/servers/{machineId}` | both | | [R] Wizarr |
| `POST /api/servers/{machineId}/shared_servers` | plex.tv | `{server_id, shared_server: {library_section_ids: [plex.tv ids], invited_id}, sharing_settings: {}}` | [M] #74 (legacy token) |
| `GET /api/servers/{machineId}/shared_servers` | both | XML `SharedServer id userID …` | [R] plexapi |
| `DELETE /api/servers/{machineId}/shared_servers/{id}` | clients.plex.tv first in q7, then plex.tv | | [R] plexapi |
| `GET /api/users/`, `GET /api/invites/requested` | both | XML `User`/`Server`, `Invite friend home server` | [R] plexapi |
| `DELETE /api/invites/requested/{id}?friend=&server=&home=` | plex.tv | values copied from the `Invite` element | [R] plexapi |
| `GET /api/v2/shared_servers/owned/accepted` | both | | [R] Wizarr |
| `GET /api/v2/shared_servers/owned/pending` | both | | [G] |
| `GET /api/v2/friends` | both | | [G] |
| Member `GET /api/v2/shared_servers/invites/received/pending`, `POST /api/v2/shared_servers/{id}/accept` | clients.plex.tv (q4 also lists on plex.tv) | pending ids read from `sharedServers[].id` | [M] #74 |
| Member `GET /api/invites/requests`, `PUT /api/invites/requests/{id}?friend&server&home` | plex.tv | the old accept, tried only for a Home invite | [R] plexapi, which Wizarr calls outdated |
| `GET /api/home/users`, `POST /api/home/users?invitedEmail=<username>` (then `<email>` on refusal), `POST ?title=` (managed), `DELETE /api/home/users/{id}` | plex.tv | | [R] plexapi. Username before email is [G] |
| `DELETE /api/v2/sharings/{id}`, then `DELETE /api/friends/{id}` | plex.tv | removes the friend in cleanup | [R] plexapi (old path [S]) |
| `GET /devices.xml`, `DELETE /devices/{id}.xml` | plex.tv | removes the server from A in cleanup | [G] |

Other assumptions worth checking live:

- Every call sends `Accept: application/json`. The v1 routes may still answer XML. The recorder parses both and logs XML as `response_xml` (element maps with `_tag`, attributes and `_children`).
- A share's id is read from a `SharedServer` element's `id`, or from an object that carries `invitedId`, `userID` or `acceptedAt`. If `share` finds no `id`, read the logged body and adjust `shareID`.
- `items()` treats a listing's entries as XML children or JSON arrays. A listing shaped differently shows `matched: false` even when the Member is in it, so read `response_*` before trusting a `false`.
- `server` finds the server by name `recorded-run-pms` with `owned` true. An older claimed container with the same name is passed over if it isn't present.
- `PLEX_MEMBERS_WEB`, `PLEX_MEMBERS_API` and `PLEX_MEMBERS_PMS` override the hosts. They exist for a local smoke test and are never needed on the run.

## Redaction

Every line is redacted when it is written. These values become stable labels: tokens (`<token-A>`, `<token-B>`, `<token-AL>`), PIN codes, client identifiers, account ids, uuids, usernames, titles and emails (`<id-B>`, `<username-B>`, `<email-B>`), the machine id (`<machine-id>`), the server tokens (`<server-token>`, `<server-token-B>`), the public address, the claim token and share ids (`<share-B>`). The same value gets the same label on every line, so equal ids stay visibly equal.

On top of the labels:

- Strings under personal keys (`email`, `username`, `thumb`, `uuid`, `accessToken`, `address`, `uri`, `code` and others) become `<string>`.
- `location` becomes `<removed>`.
- Any other JWT becomes `<unlabelled-jwt>`.
- Other emails become `<email>`, plex.direct hosts `<plex-direct-host>`, and IPv4 addresses other than 127.0.0.1 `<ipv4>`.
- `/api/v2/user` bodies are kept as field shapes, with `subscription` kept as returned, as in the JWT run.

`export` audits the result for every raw secret in `state.json` and for those patterns, and refuses to copy a log that isn't clean.

## The write-up

1. `./rec export -date 2026-11-XX` (the date of steps 1 to 12) writes `~/.slipstream-recorded-runs/plex-members/out/plex-members-run-2026-11-XX.jsonl`. It holds the redacted `signin.jsonl` lines, then the recorder's own lines. If the audit fails, it names the label and the pattern. Find the line, add the value with a `note` only if it is safe, or fix the redaction and export again.
2. Write `plex-members-run-2026-11-XX.md` beside it, in the shape of `plex-jwt-run-2026-11-14.md`:
   - **Title:** `# Plex Member sharing: recorded run, 2026-11-XX`.
   - **Opening paragraph:** that this is the recorded run S43's spec puts first; the plexinc/pms-docker version from the `server` finding; that A is a throwaway account, with Plex Pass on or off from `admin-plex-pass`; that B to E are throwaway Members whose legacy tokens came from the Mac's sign-in; the log's name and what it labels.
   - **Recorder paragraph:** the tool is `~/.slipstream-recorded-runs/plex-members/tool/main.go` on Jackson's Mac, single-file and standard library only, with its state there for q8.
   - **`## Results`:** a table with columns `Spec question | Result | Evidence`. One row per question 1 to 8, and a row per sub-question where the answers differ (for example JWT on each route, and clients.plex.tv on each route). Result is Pass, Fail, Refused (status) or a short answer. Evidence quotes the status codes, the finding names and the redacted bodies or error text. Until q8 runs, its row says "Not run yet" with the command and the earliest date, as the JWT run did for renewal after expiry.
   - **`## What this means for the spec`:** bullets that settle each "the run decides" in `member-media-access/spec.md`: the hosts (`Endpoints.Web` or `API` alone), the listing for "Look up, then share", accept by share id or by owner and machine id, Remove access before accept and the cancel fallback, Home invites in the same pending list, the Plex Pass gate on Add to Plex Home, Plex's "Home full" refusal text, whether removing a Home user ends its shares, and the legacy-token lifetime for the ADR. Name every finding the spec should change.
3. Hand both files back to the project thread, which saves them beside `plex-jwt-run-2026-11-14.jsonl` in project files.

## Cleanup

`cleanup` does these, in order:

1. Deletes every share on the server (from both the v1 list and `owned/accepted`).
2. Cancels every sent invite.
3. Deletes the Home users the run knows (B to E and the managed fillers, never A).
4. Removes each of B to E as a friend.
5. Removes the server device from A.
6. Lists everything again into a `left` finding, which should be all zeros.
7. Signs out C, D, E and AL, and A unless you pass `-keep-admin`. B and B2 wait for q8.

It ends by printing the commands that delete the container, its volumes and the clips.

`cleanup -all` after q8 also signs out B and B2. A 404 counts as done, so rerunning cleanup is safe.

Jackson then deletes accounts A, C, D and E at app.plex.tv, and B after q8.
