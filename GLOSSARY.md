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
A set of qualities a Target ranks as equal, so neither replaces the other as an upgrade. Files of one tier upgrade only by Revision or Score.
_Avoid_: Group, quality group

**Fills**:
A file fills a Target for an Edition when it is the one file that Target holds for that Edition. Under any cut, a Target holds one file of any Edition. A file can meet a Target without filling it, when another file ranks higher there.
_Avoid_: Holds, occupies, assigned to

**Kept file**:
A library video file that fills no wanted Target and stays, flagged, such as one whose Edition stopped being wanted or one a Library scan found. It is named with the label of the first Target it meets, or the label last recorded for it when it meets none, plus ` (extra N)`, a counter, so two Kept files never clash. A Library scan records the label and counter without renaming the file.
_Avoid_: Extra, extras

**Gone file**:
A library file a Library scan didn't find on its Ready root. Its record stays and its cells read missing until it returns or the Admin forgets it on its Work's page. A gone file never starts a search by itself.
_Avoid_: Missing file (missing is a cell state), vanished, deleted

**Companion**:
A file in a video's folder named after the video, as the video's name without its extension, then `.` and a tail, such as `Heat (1995) - 4K HDR.en.srt` or `Heat (1995) - 4K HDR.nfo`. It is never a video or a file SlipStream tracks, and it renames and moves with its video.
_Avoid_: Related file, extra file

**Media info**:
What the bundled ffprobe read from a file's streams: the video codec, resolution, bit depth and HDR format, each audio track's codec, channels and language, the subtitle languages and the duration. Targets judge a library file by it.
_Avoid_: MediaInfo, probe data

**Impact**:
What a change to a Target, a Profile, a cut or one Work's Profile or Editions would do, shown before it saves: the Works it changes, the files it leaves filling nothing, the files it renames and the cells it makes wanted. The Admin chooses whether the files that fill nothing are deleted or stay as Kept files.
_Avoid_: Dry run, conflict check, impact check

**Rename plan**:
The list of library files whose path differs from what the naming templates render, each with its old and new path, and the step that renames them when the Admin confirms. It covers one Work, one Content Type or the whole library, and every Impact shows one in place of a count of renames.
_Avoid_: Mass rename, rename preview

**Cell**:
One Edition at one Target of one Item. Under any cut, an Item has one cell per Target. A cell's state is failed, missing, downloading, upgradable, unreleased or met. A final file makes it met. Otherwise it is downloading from the moment a Grab for it exists until no unfinished Grab holds it, fetching included, and failed when its latest Grab failed in a way only an Admin can resolve. A downloading or failed cell keeps the file its Grab would replace.
_Avoid_: Slot, square

**Availability date**:
The day an Item counts as released, chosen by its Content Type's rule from the release dates its metadata source lists. A movie counts as released on its first home release in the metadata region, then in any country, then a settable delay after its cinema release; an Admin can let the first cinema release count instead. An episode counts as released on its TVDB air date plus TV's "Delay after an air date", 0 days unless the Admin sets one, on the Controller's calendar, with no air time or network time zone. An Item with no date stays unreleased.
_Avoid_: Release date (for the deciding day), available date, out date

**Item status**:
An Item's state, rolled up from its cells with what needs attention first: failed, missing, downloading, upgradable, unreleased, met. An Item reads unreleased only when nothing else applies, and a Work's status rolls up its Items the same way. Cells of a listed cut not yet seen for the Item stay out of it.
_Avoid_: Available (for an Item or cell), complete

**Needs you**:
The list of what only a person can resolve, such as failed cells, upgrades available, pending and Delayed Requests, and jobs left waiting, each with its cause and recovery.
_Avoid_: Alerts, attention, inbox

**Monitoring preset**:
A rule that sets which of a Work's Items are monitored, from level order, the aside mark, each Item's availability date and whether a file fills any of its cells: All, Future, Missing, Existing, First (level), Latest (level) or None. Each also sets Monitor new Items, and Include adds the Items of aside levels, such as Specials, to any of them. The add, the Work page and bulk edit apply the same presets, and bulk Monitor and Unmonitor are All and None.
_Avoid_: Monitor mode, monitor option, preset (alone, which reads as a Target's tiers)

**Monitor new Items**:
A Work's switch for whether Items its metadata source adds later arrive monitored. An Item in an aside level arrives monitored only when the Work also includes its asides. A preset sets it, and an Admin can change it on its own. Its label uses the Item noun, such as "Monitor new episodes".
_Avoid_: Monitor future, auto-monitor

**Episode order**:
The numbering a series follows, chosen per Work in Edit: Aired (TVDB's default order), DVD or Absolute, Aired unless the Admin picks another. Each episode keeps its identity across orders, so changing the order moves episodes between seasons and keeps their files, Grabs and monitoring. Edit lists only the orders TVDB has for the series, and nothing ever reads TMDB's numbering.
_Avoid_: Numbering scheme, season type, episode group

**Aside**:
A code word for an outermost level outside a Work's main run, such as season 0. Copy never says it: it says the Content Type's noun for it, such as Specials.
_Avoid_: Extras, special (as the generic term)

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
Releases known to be bad, keyed by torrent info hash, that no search grabs again for any Work until an Admin removes them. A release fault, Mark failed, Not this Release on a waiting download and a canceled Grab the Admin chose to blocklist each add one.
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
The state every restored Controller starts in, including after a manual rollback, until it matches the Backup to what Nodes hold. It cancels in-flight jobs from the Backup, adopts torrents it doesn't know and sends unmatched files to Pending imports, and nothing searches, blocklists, or removes or renames a torrent or file until it finishes.
_Avoid_: Resync, recovery mode

**Role**:
A job the admin assigns to one or more Nodes. The Roles are Searching, Downloading, Storage, Converting, Subtitles, Metadata, Media Server link, Notifying, Relaying and Serving. Work that reads the library database on every call stays on the Controller and is never a Role. Every Role but Relaying can sit on any number of Nodes. Searching, Downloading, Storage, Metadata and Notifying always have a holder, falling back to the Controller. The Controller always holds Serving and Media Server link, whatever the admin ticks.
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
A one-way copy of files from one Node to another for a job, such as a completed download going to the storage Node that imports it. The source keeps its copy, and the receiving Node is the destination. The Controller records each Transfer, and the job that asked for it places its files once verified.
_Avoid_: Sync, mirror, push; target for the destination

**Lane**:
One TLS connection, or one stream on the Node link, that carries a Transfer's chunks. A Transfer has one or more.
_Avoid_: Channel, pipe

**Send limit**:
A Node's cap on the bandwidth its Transfers send, across all of them.
_Avoid_: Throttle, upload limit (that is the torrent client's)

**Staging folder**:
The folder on the destination's filesystem where the receiving Node writes a Transfer's files until the job that asked for the Transfer places them.
_Avoid_: Temp folder, incoming, holding (that is import's)

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

**Library scan**:
The Scheduled task that compares each library root with SlipStream's records, matches found files to Works, sends the rest to Pending imports and reads media info. It runs daily in the Maintenance window with one unit per root. A new root's first Ready report, a finished re-home and Scan now each scan one root at once. A scan that would mark more of a root's files gone than the share the Admin set, 25% by default, marks none until the Admin confirms them with Mark them gone.
_Avoid_: Rescan, library sync, refresh

**Import**:
The work of bringing a set of video files into one Work from a download, a folder in a library root or a folder the Admin picks. Each file fills the cells it beats, waits for the Admin or is left out with a reason.
_Avoid_: Import job, processing, post-processing

**Pending import**:
An Import with a file that waits for the Admin, such as a download whose file isn't better than the one the Work has, or a folder a Library scan can't match to a Work with confidence.
_Avoid_: Unmatched, manual import queue, orphan

**Force into a Target**:
Fill a Target with a file that meets it but ranks lower than the file it holds.
_Avoid_: Force import, downgrade

**Clone**:
A copy that shares its data on disk with the original until either file changes. Btrfs, XFS and APFS can clone.
_Avoid_: Reflink, CoW copy

**Metadata refresh**:
The Scheduled task that keeps every Work's record current from its metadata source. Each run refreshes the Works the source reports changed, then every Work its tier makes due: Works with an unreleased Item, Works that can still gain Items, and every other Work, each on its own cadence. Refreshing a Work never searches for it again.
_Avoid_: Metadata sync, rescan, update metadata

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
A torrent client that a downloading Node sends Grabs to, either its built-in one or an external one such as qBittorrent, Transmission, Deluge or rTorrent. An external client runs on its own, and its Node reaches it at one address.
_Avoid_: Downloader, torrent app

**Client folder**:
A folder as a Download Client reports it, paired with the path of the same folder on the client's Node.
_Avoid_: Path mapping, remote path mapping

**Attached torrent**:
A torrent that was already in an external Download Client when a Grab sent the same info hash. SlipStream tracks it without having added it, and removing it in SlipStream stops tracking it and leaves it in the client.
_Avoid_: Adopted torrent, existing torrent

**New Grabs go to**:
A downloading Node's setting that names the one Download Client its Grabs go to: its built-in client or one of its external clients that is on. It starts at the built-in client. Tracker routing, where a Tracker has one, comes before it.
_Avoid_: Default client, grabs to

**Tracker**:
A site a torrent comes from and seeds to. It carries the site's seeding rule, the admin's seeding goals over it, hit and run state and its Tracker routing. A Tracker outlives every Indexer that reaches it.
_Avoid_: Site, indexer (for the seeding side), tracker (for an announce URL)

**Seeding rule**:
What a Tracker requires before a torrent may stop seeding: a ratio, a seeding time, and which of them clears it. SlipStream reads it from the minimum ratio and seeding time the Tracker's Releases state, and the Admin can set it.
_Avoid_: floor, minimums, H&R rule

**Seeding goal**:
What the built-in client seeds a torrent to: a ratio, a time, how they combine (Whichever first, Both or Forever) and what happens when met. A default goal applies unless a Tracker has its own. It never ends seeding before the Tracker's seeding rule is met. External clients keep their own rules.
_Avoid_: Seed ratio target, cleanup mode, seeding rule (for the admin's goal)

**Hit and run**:
Stopping or removing a torrent before its Tracker's seeding rule is met. Each seeding torrent counts down to when removing it no longer counts as one.
_Avoid_: H&R, HnR

**Hit and run window**:
The time a private torrent owes its Tracker's seeding rule, from its first downloaded byte until the rule is met. It ends met, missed or released.
_Avoid_: H&R window

**Indexer**:
One configured way to search one or more Trackers, such as a catalog definition, a ported site indexer, a Torznab feed or an RSS feed. Each Indexer searches from one home Node. An Indexer that signs in to its site holds a login session and doesn't fail over unless the Admin allows it.
_Avoid_: Tracker (for the search side), provider, source

**Indexer catalog**:
The list of site definitions the Controller downloads at runtime, from Prowlarr's catalog unless the Admin changes the address.
_Avoid_: Definitions repo, indexer list

**Definition**:
One catalog or custom file describing how to sign in to, search and download from one site.
_Avoid_: Cardigann file, indexer YAML

**Home Node**:
The one Searching Node an Indexer runs every query, login, RSS fetch, Test and .torrent fetch from. It defaults to the Controller when the Controller holds Searching, and otherwise to the first Searching Node. The Controller still decides every Grab and keeps the Indexer's cookies, limits and health.
_Avoid_: Search node, proxy

**Challenge solver**:
A service speaking FlareSolverr's API, such as Byparr, that clears a site's browser challenge for a Searching Node. Each Searching Node has its own, and none is bundled.
_Avoid_: FlareSolverr (as a generic term), captcha solver

**Tracker routing**:
Which Download Client a Tracker's Grabs go to, set per Node an Indexer of the Tracker searches from. By default it is the built-in client on that Node.
_Avoid_: Client mapping, routing rule

**Release**:
What an Indexer lists for download: a title with the facts it states about itself and one info hash.
_Avoid_: Result, torrent (for the listing), NZB

**Magnet link**:
A link that names a Release by its info hash, with an optional name and trackers, in place of its .torrent. While the Grab is fetching, the Downloading Node gets the torrent's metadata from peers, and the Grab fails if no peer sends it within the metadata wait.
_Avoid_: Magnet URI, magnet (alone, in UI)

**Want**:
One Edition at one Target of a monitored, released Item, still listed by its source, that no file fills, or whose filling file is below the Target's cutoff or below its upgrade-until score, and that no unfinished Grab holds. Under any cut, a Want is one Target of the Item, which a file of any Edition can fill. Every search and RSS sync tries to satisfy Wants.
_Avoid_: Missing item, wanted cell, slot

**Grab**:
The decision to download one Release for a Work and Edition, recorded with the Items it covers and the Targets it is meant to fill, and sent to one Download Client. Once the download completes, an Import takes its files.
_Avoid_: Snatch, download (for the decision)

**Hold**:
How long a Target waits after a Release is published before grabbing it automatically, so a better one can arrive first.
_Avoid_: Delay, delay profile

**Backlog search**:
The scheduled search that works through Wants nothing has found yet, spacing each Want's searches further apart while it stays unfound. At its longest gap, it also looks for listed cuts not yet seen for an Item.
_Avoid_: Sweep, scheduled search, auto search

**Manual search**:
A search an Admin opens to see the Releases for part of a Work and choose what to grab. It asks every Indexer that is on, leads with a Plan and grabs nothing until the Admin presses Grab.
_Avoid_: Interactive search, search by hand

**Plan**:
In Manual search, the fewest Grabs that fill the missing, failed and upgradable cells in scope, one Release per row, chosen as automatic search would choose them.
_Avoid_: Bulk grab, cover, suggestion

**Custom Format**:
A named set of conditions on what a release says about itself, such as its title, Release group or source, that each Target scores. Its conditions have Radarr's and Sonarr's shape, so a format from TRaSH Guides, Radarr or Sonarr pastes in and runs as written.
_Avoid_: Preferred word, release profile, tag

**Score**:
The sum of a Target's scores for the Custom Formats a Release matches. Each Target also holds a minimum score, below which a Release isn't grabbed automatically, an upgrade-until score, which a file at the cutoff keeps upgrading towards, and a minimum increment, by which a Release must beat the file's score to replace it.
_Avoid_: Weight, points

**Revision**:
How many times a Release was re-released to fix it, read from PROPER, REPACK, RERIP, vN and REAL. A higher Revision of the same Quality replaces the file it fixes, even past the cutoff, as each Target's propers and repacks setting allows. A repack replaces only a file from its own Release group.
_Avoid_: Version (for this)

**Release group**:
The group a Release names as its maker, such as FLUX in a title ending -FLUX.
_Avoid_: Uploader

**Refusal**:
Why a Release can't fill a Want, such as a title that doesn't match the Work or a Score below the Target's minimum. A hard refusal, such as an id that names another Work, ends judging. A soft refusal, such as a title or year that doesn't match, leaves the Release judged at every Target, and Manual search can grab past it.
_Avoid_: Rejection, block

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

**First setup**:
The guided steps a new Controller's first Admin walks through after the claim, open until that Admin finishes them.
_Avoid_: Onboarding, setup wizard, first run (for the steps after the claim)

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
