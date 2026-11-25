# plex.tv response shapes from the 2026-11-25 recorded run (read from the Mac's redacted export)

Every call sent Accept: application/json; v1 endpoints answer XML anyway. "id?" = id redacted to a label (JSON numbers elsewhere in the run). XML attributes are strings.
SharingSettings = { allowCameraUpload, allowChannels, allowSubtitleAdmin, allowSync: bool; allowTuners: number; filterAll, filterMovies, filterMusic, filterPhotos, filterTelevision: null }

## 1. GET /api/v2/shared_servers/owned/accepted and /api/v2/shared_servers/owned/pending
Path is `shared_servers`, not `sharings`. 200 JSON on clients.plex.tv and plex.tv ([] when none). Filled lists read on clients.plex.tv.
[ { id, accepted: bool, acceptedAt: string|null, allLibraries: bool, deletedAt: null, inviteToken: string, invitedEmail: null,
    invitedId, lastSeenAt: string, leftAt: null, machineIdentifier: string, name: string, numLibraries: number, owned: true, ownerId,
    searchEnabled: bool, serverId: number, libraries: [ { id: number, key: number, title, type } ], sharingSettings,
    invited: { id, uuid, username, title, thumb, friendlyName: null, home: bool, restricted: bool, status: "pending"|"accepted", sharingSettings, [accepted only] email, subscription: null },
    owner: { id, uuid, username, title, thumb, friendlyName: null, restricted: bool, subscription: null, [accepted only] email } } ]

## 2. GET clients.plex.tv/api/v2/shared_servers/invites/owned/pending — 200 JSON
[ { type: "owned", friendshipStatus: "pending", invitedId, ownerId, invited: {id, uuid, username, title, thumb, friendlyName, restricted}, owner: {same},
    sharedServers: [ { id (= share id), accepted: false, ...same fields as item 1 minus invited/owner } ] } ]   (outer object has no id)

## 3. POST https://plex.tv/api/servers/{machine}/shared_servers — 200 XML
Request JSON: {server_id: string, shared_server: {invited_id: number, library_section_ids: [number (plex.tv section id, not local key)]}, sharing_settings: {}}
<MediaContainer friendlyName identifier machineIdentifier size="1"><SharedServer id userID username email name owned="0" invitedAt acceptedAt="0" accessToken="" allow*...><Section id key shared="1" title type/></SharedServer></MediaContainer>
GET same path lists only accepted shares (empty before accept, both hosts).

## 4. GET /api/invites/requested — 200 XML, both hosts
<MediaContainer ...><Invite id(=invitee account id) username email friendlyName thumb createdAt friend="1" home="0" server="1|0"><Server name numLibraries/></Invite></MediaContainer>
After a pending share is deleted the Invite stays with server="0" and no Server child.

## 5. GET /api/home/users — 200 XML, both hosts
<MediaContainer friendlyName guestUserID identifier machineIdentifier size totalSize><User id uuid username email title thumb admin guest hasPassword protected restricted restrictionProfile/></MediaContainer>

## 6. Member (own token)
GET clients.plex.tv/api/v2/shared_servers/invites/received/pending — 200 JSON (plex.tv too); shape as item 2 with type "received", sharedServers[].owned false, no inviteToken.
POST clients.plex.tv/api/v2/shared_servers/{shareId}/accept — 204, no bodies. shareId = sharedServers[].id = POST share id.

## 7. Deletes (Admin token)
Before accept: DELETE clients.plex.tv/api/servers/{machine}/shared_servers/{shareId} — 200 empty (application/xml). plex.tv not tried.
After accept: DELETE plex.tv/api/servers/{machine}/shared_servers/{shareId} — 200 empty. clients not tried.
Invite cancel: DELETE plex.tv/api/invites/requested/{inviteeAccountId}?friend=1&home=0&server=0 (flags echo the Invite's attributes; server=1 while a share is pending) — 200 XML <Response code status="Canceled invitation to ..."/>. One cancel with server=1 removes the pending share's invite.
Friend removal (not needed): DELETE plex.tv/api/v2/sharings/{accountId} — 204.

## 8. Errors (application/xml)
Second share: 400 <Response code="400" status="You're already sharing this server with <user>. Please edit your existing share."/>
Home invite without Plex Pass: POST plex.tv/api/home/users?invitedEmail=<username or email> — 403 <Response code="403" status="Invite a user requires a Plex Pass"/>
Home full: 403 same shape, status "You have reached the 15 person Home limit". Tell them apart by status text.

Note: the export holds share inviteToken values in clear on 4 lines (redactor gap). Shares are deleted, but don't publish the file.
