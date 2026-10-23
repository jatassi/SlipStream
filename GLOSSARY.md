# SlipStream

SlipStream finds, downloads, processes and files media into a library that media servers play, across one or more machines.

## Media

**Content Type**:
A pluggable kind of media with its own shape and metadata sources, such as movies or TV.
_Avoid_: Module, media type

**Work**:
What a user adds and monitors, such as a movie or a series. A Work holds Items, grouped under any levels its Content Type declares, such as seasons.
_Avoid_: Title, show, media

**Item**:
The part of a Work that is acquired and holds files, such as a movie itself or one episode.
_Avoid_: Unit, entry, episode (as a generic term)

**Coverage**:
The Work and the set of its Items that a release or a file holds, such as one episode, a whole season or one album.
_Avoid_: Match, scope

**Edition**:
A cut of an Item, such as Standard, Theatrical, Extended or Director's. Every Item has at least one Edition. A Work's list is either any cut or named cuts, never both. Under any cut, an Item wants one file at each Target, of whatever Edition, and the file's own Edition sets only its name. A list naming Standard always wants it, and wants each other Edition on the list only once a release or file of it has been seen for the Item. A list without Standard wants its Editions outright. Each wanted Edition is wanted at every Target in its Work's Profile. A release with no cut named holds Standard.
_Avoid_: Version

**Cut**:
A named kind of Edition, such as Extended, that an Edition list names and that SlipStream reads from release titles and file names by its name and aliases. A cut is built in, added by a Content Type or added by an Admin, and Standard is fixed.
_Avoid_: Edition type, version

**Target**:
A library-wide, named encoding goal, such as 1080p low bitrate or 2160p HDR, with the qualities it accepts ranked in tiers, a cutoff, hard filters, sizes and Custom Format scores. A Profile sources each of its Targets by download, by Conversion from another Target, or both. One file meets every Target whose rules it passes.
_Avoid_: Rendition, slot, version, quality version, variant

**Quality**:
One of the 18 fixed kinds of release or file a Target can accept, such as WEBDL-1080p, set by its source and resolution. CAM, for cinema recordings, is one of them.
_Avoid_: Resolution (for the whole kind), format, source

**Tier**:
A set of qualities a Target ranks as equal, so neither replaces the other as an upgrade.
_Avoid_: Group, quality group

**Fills**:
A file fills a Target for an Edition when it is the one file that Target holds for that Edition. Under any cut, a Target holds one file of any Edition. A file can meet a Target without filling it, when another file ranks higher there.
_Avoid_: Holds, occupies, assigned to

**Kept file**:
A library video file that fills no wanted Target and stays, flagged, such as one whose Edition stopped being wanted or one a rescan found. It is named with the label of the first Target it meets, or its current label when it meets none, plus ` (extra N)`, a counter, so two Kept files never clash.
_Avoid_: Extra, extras

**Impact**:
What a change to a Target, a Profile, a cut or one Work's Profile or Editions would do, shown before it saves: the Works it changes, the files it leaves filling nothing, the files it renames and the cells it makes wanted. The Admin chooses whether the files that fill nothing are deleted or stay as Kept files.
_Avoid_: Dry run, conflict check, impact check

**Cell**:
One Edition at one Target of one Item. Under any cut, an Item has one cell per Target. A cell's state is unreleased, held, missing, downloading, converting, failed, upgradable or met, and a missing cell is either Not found or waiting on its source Target.
_Avoid_: Slot, square

**Item status**:
An Item's state, rolled up from its cells in this order: unreleased, held, missing, downloading, converting, failed, upgradable, met. Cells of a listed cut not yet seen for the Item stay out of it.
_Avoid_: Available (for an Item or cell), complete

**Needs you**:
The list of what only a person can resolve, such as failed cells, upgrades available, Not found cells, pending and Delayed Requests, and jobs left waiting, each with its cause and recovery.
_Avoid_: Alerts, attention, inbox

**Profile**:
The ordered Targets a Work wants, each with how it is sourced: downloaded, derived from another Target, or downloaded first and derived after a grace period. It also sets the default list of Editions wanted, which a Work can replace with its own list, and the Wanted Languages for subtitles. Every Work has one, defaulting per Content Type.
_Avoid_: Rendition profile, quality profile, slot

**Conversion**:
One run that turns one source file into one file for one Target, following that Target's Conversion settings, whether or not the video is re-encoded. Making a 1080p file from a 2160p one is a Conversion, and so is dropping extra audio from a download and adding a stereo track.
_Avoid_: Transcode, transcode job, pipeline job, encode, task; never a Media Server's live transcoding

**Conversion settings**:
What a Target's file should come out as when SlipStream makes or adjusts it: video, dynamic range, audio tracks and container. They belong to the Target, and every Profile that derives the Target uses them.
_Avoid_: Recipe, encode settings, preset, transcode profile

**Blocklist**:
Releases known to be bad, keyed by torrent info hash, that no Item ever grabs again until removed by hand.
_Avoid_: Blacklist, ignore list

## Topology

**Node**:
One SlipStream install running on one machine.
_Avoid_: Instance, server, peer

**Controller**:
The one Node that owns the library database and serves the UI. A single-machine setup is a Controller holding every Role. The Controller moves to another Node only by Handover.
_Avoid_: Primary, master, hub

**Worker**:
Any Node other than the Controller. A Worker joins by Pairing and does the work of the Roles assigned to it.
_Avoid_: Agent, client, slave

**Unclaimed**:
A fresh install that is neither a Controller nor a Worker yet. From its first-run page, gated by the Setup code, it starts a new SlipStream, joins one with a Join Code, or restores a Backup.
_Avoid_: Unpaired, uninitialized

**Node link**:
The one connection between two paired Nodes, authenticated by each Node's pinned key, which carries calls both ways whichever side dialed. On a single machine it is an in-process link.
_Avoid_: Channel, tunnel

**Pairing**:
Joining a new Node to a SlipStream with a Join Code from the Controller, after which the two Nodes trust each other's keys. A Node the Controller already knows by its key keeps its Roles when it pairs again.
_Avoid_: Registration, enrollment, adoption

**Join Code**:
A short-lived, single-use code the Controller issues so one new Node can pair with it.
_Avoid_: Invite, token, pairing key

**Node address**:
A short text naming one Node's key and the addresses it listens on, which an Admin pastes on the Controller so the Controller dials that Node when the Node can't reach the Controller.
_Avoid_: Dial code, Node code, Join Code (for the Node's side)

**Handover**:
Moving the Controller to another paired Node, both running, by shipping it a Backup over the Node link. Workers follow a notice signed with the old Controller's key, without a Join Code, and nothing in flight is cancelled. The old Controller becomes a Worker that keeps its Roles, library roots and torrents and redirects Workers that reconnect late.
_Avoid_: Failover, promotion, migration

**Backup**:
An encrypted copy of the Controller's library database, from which a Node can be restored as Controller. It never holds the Controller's key.
_Avoid_: Config backup, snapshot, dump, export

**Backup destination**:
A folder on a Node that receives a copy of every Backup.
_Avoid_: Backup target, backup location, remote

**Reconciling**:
The state every restored Controller starts in, including after a manual rollback, until it matches the Backup to what Nodes hold. It cancels in-flight jobs from the Backup, adopts torrents it doesn't know and sends unmatched files to pending imports, and nothing searches, blocklists, or removes or renames a torrent or file until it finishes.
_Avoid_: Resync, recovery mode

**Role**:
A job the admin assigns to one or more Nodes. The Roles are Searching, Downloading, Storage, Converting, Subtitles, Metadata, Media Server link, Notifying, Relaying and Serving. Work that reads the library database on every call stays on the Controller and is never a Role. Every Role but Relaying can sit on any number of Nodes. Searching, Downloading, Storage, Metadata and Notifying always have a holder, falling back to the Controller.
_Avoid_: Service, capability, purpose

**Role Group**:
A named bundle of Roles that the admin ticks together when assigning: Acquire (Searching, Downloading), Store (Storage), Process (Converting, Subtitles), Connect (Metadata, Media Server link, Notifying) and Reach (Relaying, Serving). A Node holds Roles, never Role Groups.
_Avoid_: Preset, profile, tier

**Roster**:
Every Node with the Roles it holds and each Role's state, which answers who may hold a Role right now. The Nodes page shows it as one row per Node.
_Avoid_: Node list, fleet, inventory

**Draining**:
The state of a Role unticked on a Node that still holds work for it. The Node finishes that work and takes nothing new, and ticking the Role again ends the drain.
_Avoid_: Decommissioning, winding down

**Write off**:
Giving up the work a Node holds for a Role, on a live or an offline Node. Its Grabs not yet imported fail and their Items are wanted again, and the Node's files stay on disk.
_Avoid_: Force, abandon, discard

**Covering**:
The Controller holding a Role whose holders have all been offline past the offline grace, without being assigned it. It applies only to Roles whose work can move to the Controller, and ends once a holder is back.
_Avoid_: Failover, fallback, standby

**Serving**:
The Role that serves the UI and portal on a Node, passing every request to the Controller. The Controller always serves too.
_Avoid_: Hosting, web; never the Media Server's serving of the library

**Relaying**:
The Role that accepts connections from the internet on a Node, usually a rented VPS, and forwards each one to the Node and port it maps to, for a home network that can't accept them itself. At most one Node relays, and that Node is the Relay Node.
_Avoid_: Tunnel, proxy, gateway

**Exposure**:
One thing the admin makes reachable from the internet: the UI and portal, one Media Server, or one named port. The Reachability diagnostic gives each Exposure one verdict, and Remote access draws it as one route.
_Avoid_: Service, endpoint, forward

**Reachability diagnostic**:
The check that tells the admin whether each Exposure can be reached from the internet, over IPv4 and over any IPv6 address DNS or the Media Server offers, and what to do if not: open or map a port, stop offering an address that doesn't work, or turn on Relaying.
_Avoid_: Port check, firewall check, connectivity test

**Transfer**:
A one-way copy of files from one Node to another for a job, such as a completed download going to the storage Node that imports it. The source keeps its copy, and the receiving Node is the destination.
_Avoid_: Sync, mirror, push; target for the destination

**Library root**:
A folder on one Storage Node that holds Works of one Content Type, recorded on the Controller. A Work's folder is stored relative to its root, so editing the root's Node and path re-homes every Work in it.
_Avoid_: Root folder, library folder

**Volume**:
The filesystem a library root sits on, as its Node sees it, such as a drive, a partition, a network share or a dataset. Two roots on one volume share its free space.
_Avoid_: Disk, drive, mount (when the volume is meant)

**Warning level**:
The free space below which a volume shows a Warning on Health, set as an amount or a percentage of the volume, once globally and optionally per volume.
_Avoid_: Threshold, low-space limit

**Error level**:
The free space below which a volume shows an Error on Health, set like the Warning level. A volume below both shows only the Error.
_Avoid_: Critical threshold, hard limit

**Move**:
An admin-started job that relocates one Work's files to another library root, or one torrent to another Download Client, then deletes the source. Files that cross Nodes go as Transfers, and the whole Work or torrent switches over at once before the source is deleted.
_Avoid_: Migration, relocation, sync, transfer (for the whole job), Handover

**Media Server**:
An external player-facing server that serves the library to players, such as Plex, Jellyfin or Emby. The admin adds each one SlipStream keeps up to date, and one Node holding the Media Server link Role reaches it. The Media Server, never SlipStream, picks which file a player gets.
_Avoid_: Plex (as a generic term), link, connection

**Path mapping**:
How one Media Server sees a library root, such as `/mnt/media` on the storage Node being `/data/media` to Plex.
_Avoid_: Path substitution, remote path mapping

**Playback report**:
What SlipStream shows about recent plays on each Media Server: which players transcoded, why, and which Target would have played without transcoding.
_Avoid_: Stats, analytics, player profile

**Scheduled task**:
A library-wide job the Controller starts on a cadence, which the Tasks page lists with its last run, result and next run. A due time on one object, a job an event queues, and watching a Node keeps up continuously are not Scheduled tasks.
_Avoid_: Job, cron job, background task, timer

**Maintenance window**:
The daily stretch of the Controller's local time when updates install and the nightly Scheduled tasks, such as Backup and Library scan, start.
_Avoid_: Update window, quiet hours

**Health**:
The list of what is wrong with SlipStream's own machinery right now, by Health category, each Health item at Warning or Error with what to do. Decisions about Works stay in Needs you.
_Avoid_: Alerts, problems, issues, system status

**Health category**:
One feature's group of Health items, such as Storage, Nodes or Scheduled tasks, which that feature derives from its own state each time Health asks. A settings section names the categories whose dot it carries.
_Avoid_: Health check, section

**Health item**:
One thing wrong in a Health category, keyed by its subject's own id, at Warning when something will go wrong if nobody acts or Error when something is wrong now, with what to do first and since when.
_Avoid_: Alert, issue, warning (for an item at either level)

## Acquisition

**Download Client**:
A torrent client that a downloading Node sends grabs to, either the built-in one or an external one such as qBittorrent.
_Avoid_: Downloader, torrent app

**Tracker**:
A site a torrent comes from and seeds to. It carries the site's seeding rule, the admin's seeding goals over it, hit and run state and its Tracker routing. A Tracker outlives every Indexer that reaches it.
_Avoid_: Site, indexer (for the seeding side)

**Seeding goal**:
What the built-in client seeds a torrent to: a ratio, a time, how they combine (Whichever first, Both or Forever) and what happens when met. A default goal applies unless a Tracker has its own, and the Tracker's seeding rule is a floor under both. External clients keep their own rules.
_Avoid_: Seed ratio target, cleanup mode, seeding rule (for the admin's goal)

**Hit and run**:
Stopping or removing a torrent before its Tracker's seeding rule is met. Each seeding torrent counts down to when removing it no longer counts as one.
_Avoid_: H&R, HnR

**Indexer**:
One configured way to search one or more Trackers, such as a catalog definition, a ported site indexer, a Torznab feed or an RSS feed. Each Indexer searches from one home Node.
_Avoid_: Tracker (for the search side), provider, source

**Home Node**:
The one Searching Node an Indexer runs every query, login, RSS fetch, Test and .torrent fetch from. It defaults to the Controller when the Controller holds Searching, and otherwise to the first Searching Node. The Controller still decides every Grab and keeps the Indexer's cookies, limits and health.
_Avoid_: Search node, proxy

**Challenge solver**:
A service speaking FlareSolverr's API, such as Byparr, that clears a site's browser challenge for a Searching Node. Each Searching Node has its own, and none is bundled.
_Avoid_: FlareSolverr (as a generic term), captcha solver

**Tracker routing**:
Which Download Client a Tracker's Grabs go to.
_Avoid_: Client mapping, routing rule

**Release**:
What an Indexer lists for download: a title with the facts it states about itself and one info hash.
_Avoid_: Result, torrent (for the listing), NZB

**Want**:
One Edition at one Target of a monitored, released Item that no file fills, or whose filling file is below the Target's cutoff. Under any cut, a Want is one Target of the Item, which a file of any Edition can fill. Every search and RSS sync tries to satisfy Wants.
_Avoid_: Missing item, wanted cell, slot

**Grab**:
The decision to download one Release for a Work and Edition, recorded with the Targets it is meant to fill and sent to one Download Client.
_Avoid_: Snatch, download (for the decision)

**Hold**:
How long a Target waits after a Release is published before grabbing it automatically, so a better one can arrive first.
_Avoid_: Delay, delay profile

**Backlog search**:
The scheduled search that works through Wants nothing has found yet, spacing each Want's searches further apart while it stays unfound. At its longest gap, it also looks for listed cuts not yet seen for an Item.
_Avoid_: Sweep, scheduled search, auto search

**Custom Format**:
A named set of conditions on what a release says about itself, such as its title, release group or source, that each Target scores. A release's score for a Target is the sum of the scores of the Custom Formats it matches.
_Avoid_: Preferred word, release profile, tag

**TRaSH Guides**:
The community guide whose Custom Formats, scores, quality sizes and naming SlipStream imports and keeps tracking.
_Avoid_: TRaSH profile (for a Target), Recyclarr

## Subtitles

**Subtitle Provider**:
An external service SlipStream searches and downloads subtitles from, such as SubDL.
_Avoid_: Subtitle source, subtitle indexer

**Wanted Language**:
One entry in a Profile's subtitle list: a language, whether full or forced only, a hearing-impaired preference, and whether it is wanted always or only when no audio track is in that language.
_Avoid_: Language profile, subtitle profile

**Sidecar**:
A subtitle file stored beside one library file. A Sidecar belongs to that file, not to its Edition.
_Avoid_: External subtitle, subtitle file

**Speech Map**:
Where speech falls in one library file's audio, built when the file is imported and used for Timing sync.
_Avoid_: VAD map, audio fingerprint

**Timing sync**:
Shifting and stretching a subtitle's cue times to match a library file's Speech Map. It never changes the cue text.
_Avoid_: Alignment, resync

**Mark bad**:
Removing a Sidecar as wrong, which blocks that candidate for that file and searches again. An Admin marks one bad from a file's subtitles, and a Subtitle report does it at once.
_Avoid_: Reject, Blocklist (for a subtitle)

**Subtitle search**:
The scheduled search for wanted subtitles that no Sidecar meets yet, spacing each one's searches further apart while nothing passes Timing sync.
_Avoid_: Subtitle sweep, subtitle auto search

## Identity

**Account**:
One person's identity in SlipStream, signed in by its email with a PIN or a passkey. Admins and Members are both Accounts.
_Avoid_: User, portal user, login

**Admin**:
An Account holding the admin permission. At least one Admin always exists.
_Avoid_: Administrator, owner, superuser

**Member**:
An Account without the admin permission.
_Avoid_: Portal user, user, guest

**Setup code**:
A one-time code a fresh install prints to its console and log, which unlocks its first-run page to create the first Admin, join a SlipStream or restore a Backup.
_Avoid_: Bootstrap token, install key

**Signup link**:
A single-use link an Admin issues so one person can create their Account.
_Avoid_: Invite, invitation, Join Code

**Reset link**:
A single-use link an Admin issues so a person who lost their credentials can set a new PIN and passkey.
_Avoid_: Recovery email, password reset

## Requests

**Request**:
An Account's ask for a Work, or some of its levels or Items, to be added and monitored, which an Admin approves or auto-approval admits. Its stage is Pending, Approved, Searching, Downloading, Processing, Available or Denied, and Delayed flags it. A Request is Available once every requested Item is Playable.
_Avoid_: Order, ticket, portal request; Requested, Coming, Ready (for a stage)

**Watcher**:
An Account following another Account's Request, who gets its notices without spending a Request and inherits it if the requester cancels.
_Avoid_: Subscriber, follower

**Request limit**:
How many of a Member's Requests per week, per Content Type, are approved automatically. A Request over the limit waits for an Admin.
_Avoid_: Quota, allowance

**Playable**:
An Item with any library file a Media Server can play, Kept files included.
_Avoid_: Downloaded, complete, available (for an Item)

**Delayed**:
A flag on a Request whose released Items have stayed unplayable too long or hold a failed cell. Search keeps running underneath it.
_Avoid_: Failed, stuck

**Subtitle report**:
A Member's report that one language's subtitles on an Item are wrong, which SlipStream acts on with Mark bad.
_Avoid_: Complaint, issue, flag

## Development

**Offline mode**:
Running SlipStream against the Fake world instead of the real services it talks to, for developers, agents and CI.
_Avoid_: Dev mode, developer mode, mock mode

**Fake world**:
The set of fakes standing in for every external boundary SlipStream crosses, such as metadata providers, trackers and Media Servers, together with the Nodes started against them.
_Avoid_: Mocks, sandbox, simulation

**Scenario**:
One declared state and script for the Fake world: the Works, releases, swarms, servers and providers it holds, and the faults it plays on cue.
_Avoid_: Fixture, seed data, test case
