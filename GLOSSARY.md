# SlipStream

SlipStream finds, downloads, processes and files media into a library that media servers play, across one or more machines.

This glossary names SlipStream's concepts for code, specs, issues and docs. It isn't the UI's word list. A term reaches a screen only through its `_In UI_` line, which gives the words a person sees there, and a line of none keeps it out of user-facing strings. Copy then says what the concept means for the reader. [WRITING.md](WRITING.md) sets how. A definition says what the thing is in one or two sentences. The rules for how it behaves belong in its spec or ADR, or in [docs/domain-rules.md](docs/domain-rules.md) when neither records them.

## Media

**Content Type**:
A pluggable kind of media with its own shape and metadata sources, such as movies or TV.
_In UI_: The plural noun of each type, such as "Movies" and "TV". "Kind of media" where the types are listed together.
_Avoid_: Module, media type

**Work**:
What a user adds and monitors, such as a movie or a series. A Work holds Items, grouped under any levels its Content Type declares, such as seasons.
_In UI_: The Content Type's noun, such as movie or series. "Work" only on screens that mix types, such as the Library.
_Avoid_: Title, show, media

**Item**:
The part of a Work that is acquired and holds files, such as a movie itself or one episode.
_In UI_: The Content Type's noun, such as movie or episode. Never "Item".
_Avoid_: Unit, entry, episode (as a generic term)

**Coverage**:
The Work and the set of its Items that a release or a file holds, such as one episode, a whole season or one album.
_In UI_: none
_Avoid_: Match, scope

**Edition**:
A cut of an Item, such as Standard, Theatrical, Extended or Director's. Every Item has at least one Edition.
_In UI_: "Edition", or the cut's name, such as "Extended".
_Avoid_: Version

**Cut**:
A named kind of Edition, such as Extended, that an Edition list names and that SlipStream reads from release titles and file names by its name and aliases. A cut is built in, added by a Content Type or added by an Admin, and Standard is fixed.
_In UI_: "Cut", as in "Director's Cut".
_Avoid_: Edition type, version

**Target**:
A library-wide, named encoding goal, such as 1080p low bitrate or 2160p HDR, with the qualities it accepts ranked in tiers, a cutoff, hard filters, sizes and Custom Format scores.
_In UI_: "Target", defined on hover as one version of a Work to keep, such as 1080p or 2160p HDR.
_Avoid_: Rendition, slot, version, quality version, variant

**Quality**:
One of the 18 fixed kinds of release or file a Target can accept, such as WEBDL-1080p, set by its source and resolution. CAM, for cinema recordings, is one of them.
_In UI_: "Quality", and each quality's name, such as WEBDL-1080p.
_Avoid_: Resolution (for the whole kind), format, source

**Tier**:
A set of qualities a Target ranks as equal, so neither replaces the other as an upgrade. Files of one tier upgrade only by Revision or Score.
_In UI_: "Tier", in the Target editor only.
_Avoid_: Group, quality group

**Fills**:
A file fills a Target for an Edition when it is the one file that Target holds for that Edition. A file can meet a Target without filling it, when another file ranks higher there.
_In UI_: none. Say what happens to the file: "the 2160p file for Heat (1995)", "no Target uses it".
_Avoid_: Holds, occupies, assigned to

**Kept file**:
A library video file that fills no wanted Target and stays, flagged, such as one whose Edition stopped being wanted or one a Library scan found.
_In UI_: "Kept file", with its hover definition.
_Avoid_: Extra, extras

**Gone file**:
A library file a Library scan didn't find on its Ready root.
_In UI_: "Gone", as a file's state.
_Avoid_: Missing file (missing is a cell state), vanished, deleted

**Companion**:
A file in a video's folder named after the video, as the video's name without its extension, then `.` and a tail, such as `Heat (1995) - 4K HDR.en.srt` or `Heat (1995) - 4K HDR.nfo`. It is never a video or a file SlipStream tracks, and it renames and moves with its video.
_In UI_: none
_Avoid_: Related file, extra file

**Media info**:
What the bundled ffprobe read from a file's streams: the video codec, resolution, bit depth and HDR format, each audio track's codec, channels and language, the subtitle languages and the duration. Targets judge a library file by it.
_In UI_: "Media info".
_Avoid_: MediaInfo, probe data

**Impact**:
What a change to a Target, a Profile, a cut or one Work's Profile or Editions would do, shown before it saves: the Works it changes, the files it leaves filling nothing, the files it renames and the cells it makes wanted. The Admin chooses whether the files that fill nothing are deleted or stay as Kept files.
_In UI_: none. The sheet that shows it says what will change, such as "Review the changes".
_Avoid_: Dry run, conflict check, impact check

**Rename plan**:
The list of library files whose path differs from what the naming templates render, each with its old and new path, and the step that renames them when the Admin confirms. It covers one Work, one Content Type or the whole library, and every Impact shows one in place of a count of renames.
_In UI_: "Renames", as in "Review all renames".
_Avoid_: Mass rename, rename preview

**Cell**:
One Edition at one Target of one Item. A cell's state is failed, missing, downloading, converting, upgradable, unreleased or met.
_In UI_: none. Say "file" or name the episode and Target, such as "S01E03 at 1080p".
_Avoid_: Slot, square

**Availability date**:
The day an Item counts as released, chosen by its Content Type's rule from the release dates its metadata source lists.
_In UI_: "Available on".
_Avoid_: Release date (for the deciding day), available date, out date

**Item status**:
An Item's state, rolled up from its cells with what needs attention first: failed, missing, downloading, converting, upgradable, unreleased, met.
_In UI_: Each state's word: Failed, Missing, Downloading, Converting, Upgrade, Unreleased, Met.
_Avoid_: Available (for an Item or cell), complete

**Needs you**:
The list of what only a person can resolve, such as failed cells, upgrades available, pending and Delayed Requests, and jobs left waiting, each with its cause and recovery. Accounts has its own, which lists Members waiting for a Media Server share and Media Server access SlipStream couldn't remove.
_In UI_: The page name "Needs you", and the Accounts section of that name. A sentence names the problem or says "needs attention".
_Avoid_: Alerts, attention, inbox

**Monitoring preset**:
A rule that sets which of a Work's Items are monitored: All, Future, Missing, Existing, First (level), Latest (level) or None.
_In UI_: Each preset's name: All, Future, Missing, Existing, First season, Latest season, None.
_Avoid_: Monitor mode, monitor option, preset (alone, which reads as a Target's tiers)

**Monitor new Items**:
A Work's switch for whether Items its metadata source adds later arrive monitored.
_In UI_: With the Item noun, such as "Monitor new episodes".
_Avoid_: Monitor future, auto-monitor

**Episode order**:
The numbering a series follows, chosen per Work from Aired (TVDB's default order), DVD and Absolute.
_In UI_: "Episode order", and each order's name: Aired, DVD, Absolute.
_Avoid_: Numbering scheme, season type, episode group

**Aside**:
A code word for an outermost level outside a Work's main run, such as season 0. Copy never says it: it says the Content Type's noun for it, such as Specials.
_In UI_: none. Say the Content Type's noun for it, such as "Specials".
_Avoid_: Extras, special (as the generic term)

**Profile**:
The ordered Targets a Work wants, each with how it is sourced: downloaded, derived from another Target, or downloaded first and derived after a grace period. It also sets the default list of Editions wanted, which a Work can replace with its own list, and the Wanted Languages for subtitles.
_In UI_: "Profile".
_Avoid_: Rendition profile, quality profile, slot

**Conversion**:
One run that turns one source file into one file for one Target, following that Target's Conversion settings, whether or not the video is re-encoded. Making a 1080p file from a 2160p one is a Conversion, and so is dropping extra audio from a download and adding a stereo track.
_In UI_: "Conversion". A Media Server's live conversion is "Transcode".
_Avoid_: Transcode, transcode job, pipeline job, encode, task. Transcode names only a Media Server's live transcoding, never a Conversion

**Conversion settings**:
What a Target's file should come out as when SlipStream makes or adjusts it: video, dynamic range, audio tracks and container. They belong to the Target, and every Profile that derives the Target uses them.
_In UI_: "Conversion settings".
_Avoid_: Recipe, encode settings, preset, transcode profile

**Conversion settings version**:
A number on a Target's Conversion settings that rises with each change and is recorded on every file a Conversion makes, so the files made with older settings can be found and remade.
_In UI_: none. Say "made with older settings".
_Avoid_: Revision, settings hash

**Made from (sourcing)**:
A Profile row whose Target's file SlipStream makes from the file of a Target above it in the same Profile, either always or once a download hasn't come within a wait. The Conversion always reads a downloaded file, never a made one.
_In UI_: "Made from {Target}", and "Downloaded, or made from {Target} after {time}" for the row that waits for a download first.
_Avoid_: Derived from, transcoded from, generated from

**Derived file**:
A library file a Conversion made, which has no release name and keeps the id of the downloaded file it was made from. It counts for every Target it meets.
_In UI_: none. Say "From a Conversion of {source file name}".
_Avoid_: Transcoded file, encode, rendition

**Trial run**:
A short ffmpeg run on a two-second synthetic clip the Node makes, which proves the Node can convert one source class with one settings shape. A Node runs its trial runs again whenever its ffmpeg signature changes.
_In UI_: "Test run".
_Avoid_: Benchmark, capability check, trial (in UI copy)

**Proof**:
A passed trial run, kept per Node, ffmpeg signature, settings shape and source class, that names the pipeline that passed. A Conversion goes only to a Node with a proof for its shape and class, and runs the pipeline that proof names.
_In UI_: none. Say what the Node converts with, such as "Attic converts with Intel Quick Sync."
_Avoid_: Capability, certification

**No capable Node**:
A Conversion that no Node can run now, because no online Node has the Converting Role, none has a proof for it, or its Target runs only on a Node that is away.
_In UI_: "No Node can convert this".
_Avoid_: Unplaceable, stuck, unassigned

**Backfill**:
The last of the Conversion queue's four lanes, which holds the Conversions for Works that lacked a new made-from Target, remakes, files a Library scan adopts and Works moved onto a Profile.
_In UI_: "Backfill", as a lane heading only.
_Avoid_: Bulk conversion, catch-up

**Blocklist**:
Releases known to be bad, keyed by torrent info hash, that no search grabs again for any Work until an Admin removes them. A release fault, Mark failed, Not this Release on a waiting download and a canceled Grab the Admin chose to blocklist each add one.
_In UI_: "Blocklist".
_Avoid_: Blacklist, ignore list

## Topology

**Node**:
One SlipStream install running on one machine.
_In UI_: "Node".
_Avoid_: Instance, server, peer

**Controller**:
The one Node that owns the library database and serves the UI. A single-machine setup is a Controller holding every Role.
_In UI_: "Controller".
_Avoid_: Primary, master, hub

**Worker**:
Any Node other than the Controller. A Worker joins by Pairing and does the work of the Roles assigned to it.
_In UI_: "Worker".
_Avoid_: Agent, client, slave

**Unclaimed**:
A fresh install that is neither a Controller nor a Worker yet. From its first-run page, gated by the Setup code, it starts a new SlipStream, joins one with a Join Code, or restores a Backup.
_In UI_: none. The first-run page says what the person can do.
_Avoid_: Unpaired, uninitialized

**Node link**:
The one connection between two paired Nodes, authenticated by each Node's pinned key, which carries calls both ways whichever side dialed. On a single machine it is an in-process link.
_In UI_: none. Say "connection", such as "seedbox isn't connected to home".
_Avoid_: Channel, tunnel

**Pairing**:
Joining a new Node to a SlipStream with a Join Code from the Controller, after which the two Nodes trust each other's keys. A Node the Controller already knows by its key keeps its Roles when it pairs again.
_In UI_: "Add a Node" for the action, and "paired" for the result.
_Avoid_: Registration, enrollment, adoption

**Join Code**:
A short-lived, single-use code the Controller issues so one new Node can pair with it.
_In UI_: "Join Code".
_Avoid_: Invite, token, pairing key

**Node address**:
A short text naming one Node's key and the addresses it listens on, which an Admin pastes on the Controller so the Controller dials that Node when the Node can't reach the Controller.
_In UI_: "Node address".
_Avoid_: Dial code, Node code, Join Code (for the Node's side)

**Handover**:
Moving the Controller to another paired Node, both running, by shipping it a Backup over the Node link.
_In UI_: none until it ships.
_Avoid_: Failover, promotion, migration

**Backup**:
An encrypted, checked copy of the Controller's library database and library.key, from which a Node can be restored as Controller. It never holds the Node key.
_In UI_: "Backup".
_Avoid_: Config backup, snapshot, dump, export

**Backup destination**:
A folder on a Node that receives a copy of every Backup.
_In UI_: "Backup destination".
_Avoid_: Backup target, backup location, remote

**Backup passphrase**:
The ten generated words that encrypt every new Backup. The Controller keeps a copy, and the Admin needs it only to restore a Backup on another machine.
_In UI_: "Backup passphrase", or "passphrase" inside Settings > Backups. A Node's Backups section and the first-run restore count as outside it.
_Avoid_: Backup password, encryption key, recovery key

**Recovery sheet**:
A text file with the Backup passphrase, each folder that holds Backups and the steps to restore, which the Admin keeps away from the Controller.
_In UI_: "recovery sheet".
_Avoid_: Recovery kit, emergency kit, recovery file

**Reconciling**:
The state every restored Controller starts in, including after a manual rollback, until it matches the Backup to what Nodes hold, with each Node keeping every torrent it holds and the Controller adopting the ones it doesn't know. Until it finishes, work in flight from the Backup pauses and nothing searches, blocklists, or removes or renames a torrent or file.
_In UI_: "Reconciling", with its hover definition.
_Avoid_: Resync, recovery mode

**Superseded**:
The state of a Controller that a restore of its Backup on another machine replaced, proven by a Worker that carries the new Controller's proof, or that an Admin stopped or retired. It keeps every torrent paused, runs nothing and shows one page, which offers to wipe it and join the new Controller as a Worker.
_In UI_: none. The page says "This Controller was replaced".
_Avoid_: Fenced, old Controller, zombie

**Role**:
A job the admin assigns to one or more Nodes. The Roles are Searching, Downloading, Storage, Converting, Subtitles, Metadata, Media Server link, Notifying, Relaying and Serving.
_In UI_: "Role", and each Role's name.
_Avoid_: Service, capability, purpose

**Role Group**:
A named bundle of Roles that the admin ticks together when assigning: Acquire (Searching, Downloading), Store (Storage), Process (Converting, Subtitles), Connect (Metadata, Media Server link, Notifying) and Reach (Relaying, Serving). A Node holds Roles, never Role Groups.
_In UI_: Each group's name: Acquire, Store, Process, Connect, Reach.
_Avoid_: Preset, profile, tier

**Roster**:
Every Node with the Roles it holds and each Role's state, which answers who may hold a Role right now. The Nodes page shows it as one row per Node.
_In UI_: none. Say "the Nodes list".
_Avoid_: Node list, fleet, inventory

**Draining**:
The state of a Role unticked on a Node that still holds work for it. The Node finishes that work and takes nothing new, and ticking the Role again ends the drain.
_In UI_: "Finishing its work", never the bare word.
_Avoid_: Decommissioning, winding down

**Write off**:
Giving up the work a Node holds for a Role, on a live or an offline Node. Its Grabs not yet imported fail and their Items are wanted again, and the Node's files stay on disk.
_In UI_: The button "Write off" only, with a hover definition. A sentence says what happens: "SlipStream stops waiting for nas's downloads and searches again."
_Avoid_: Force, abandon, discard

**Covering**:
The Controller holding a Role whose holders have all been offline past the offline grace, without being assigned it. It applies only to Roles whose work can move to the Controller, and ends once a holder is back.
_In UI_: "Covering", as a Role's state on the Nodes page, with a hover definition. A sentence says "home is doing nas's Storage work while nas is offline".
_Avoid_: Failover, fallback, standby

**Serving**:
The Role that serves the UI and portal on a Node, passing every request to the Controller. The Controller always serves too.
_In UI_: "Serving".
_Avoid_: Hosting, web; never the Media Server's serving of the library

**Relaying**:
The Role that accepts connections from the internet on a Node, usually a rented VPS, and forwards each one to the Node and port it maps to, for a home network that can't accept them itself. At most one Node relays, and that Node is the Relay Node.
_In UI_: "Relaying", and "Relay Node" for the Node.
_Avoid_: Tunnel, proxy, gateway

**Exposure**:
One thing that the admin chooses to make reachable from the internet, each at one public address: the UI and portal, one Media Server, or one named port. The Reachability diagnostic gives each Exposure one verdict, and Remote access draws it as one Route.
_In UI_: none. Name the thing made reachable, such as the UI or Living room Plex.
_Avoid_: Service, endpoint, forward

**Reachability diagnostic**:
The check that tells the admin whether each Exposure can be reached from the internet, over IPv4 and over any IPv6 address DNS or the Media Server offers, and what to do if not: open or map a port, put it behind HTTPS, stop offering an address that doesn't work, or turn on Relaying.
_In UI_: "Reachability check".
_Avoid_: Port check, firewall check, connectivity test

**Route**:
One Exposure's way in from a player outside the home: the hops from the player through DNS, TLS and any proxy to the Node that serves it. Remote access draws each Exposure as one Route.
_In UI_: "Route".
_Avoid_: Path

**Tester**:
The paired Node that connects to an Exposure's public address as a player would, from outside the home, so the Reachability diagnostic sees what a player sees. A Node on the home network never tests.
_In UI_: none. Name the Node, such as "Tested from seedbox".
_Avoid_: Prober, checker

**Transfer**:
A one-way copy of files from one Node to another for a job, such as a completed download going to the storage Node that imports it. The source keeps its copy, and the receiving Node is the destination.
_In UI_: "Transfer".
_Avoid_: Sync, mirror, push; target for the destination

**Lane**:
One TLS connection, or one stream on the Node link, that carries a Transfer's chunks. A Transfer has one or more.
_In UI_: none. Say "connection".
_Avoid_: Channel, pipe

**Send limit**:
A Node's cap on the bandwidth its Transfers send, across all of them.
_In UI_: "Send limit".
_Avoid_: Throttle, upload limit (that is the torrent client's)

**Staging folder**:
The folder on the destination's filesystem where the receiving Node writes a Transfer's files until the job that asked for the Transfer places them.
_In UI_: "Staging folder".
_Avoid_: Temp folder, incoming, holding (that is import's)

**Library root**:
A folder on one Storage Node that holds Works of one Content Type, recorded on the Controller. A Work's folder is stored relative to its root, so editing the root's Node and path re-homes every Work in it.
_In UI_: "Library root".
_Avoid_: Root folder, library folder

**Volume**:
The filesystem a library root sits on, as its Node sees it, such as a drive, a partition, a network share or a dataset. Two roots on one volume share its free space.
_In UI_: "Volume", or the drive or share name.
_Avoid_: Disk, drive, mount (when the volume is meant)

**Warning level**:
The free space below which a volume shows a Warning on Health, set as an amount or a percentage of the volume, once globally and optionally per volume.
_In UI_: "Warning level".
_Avoid_: Threshold, low-space limit

**Error level**:
The free space below which a volume shows an Error on Health, set like the Warning level. A volume below both shows only the Error.
_In UI_: "Error level".
_Avoid_: Critical threshold, hard limit

**Move**:
An admin-started job that relocates one Work's files to another library root, or one torrent to another Download Client. Files that cross Nodes go as Transfers, and the whole Work or torrent switches over at once before the source is deleted.
_In UI_: "Move".
_Avoid_: Migration, relocation, sync, transfer (for the whole job), Handover, cutover

**Switch**:
The step of a Move that makes the destination the only copy of a Work or a torrent, in one Write. Before it the source plays and seeds and the Move can be cancelled, and after it the source is deleted.
_In UI_: "switch", as a verb in the Move's own sentences, such as "Switching to den", "12 of 43 movies switched" and "Switch now".
_Avoid_: Cutover, swap

**Media Server**:
An external player-facing server that serves the library to players, such as Plex, Jellyfin or Emby.
_In UI_: "Media Server", or the server's product name.
_Avoid_: Plex (as a generic term), link, connection

**Reaching Node**:
The one Node a Media Server names to reach it, which is the Controller or a Worker holding Media Server link. It sends the server its Media Server changes and watches its sessions, and it need not be the Node the server runs on.
_In UI_: "Connect from" as the setting, and "Connects from" on a server's row and Details.
_Avoid_: Home Node (that is an Indexer's), host Node, runs-on Node

**Fallback address**:
The address at which the Controller calls a Media Server itself while the server's Reaching Node is offline.
_In UI_: "Fallback address".
_Avoid_: Covering (that is a Role's), failover

**Show to Members**:
Whether a Media Server can be offered to Members. It is off by default, and turning it off changes nobody's Media Server access.
_In UI_: "Show to Members".
_Avoid_: Visible, public, shared (for the setting)

**Media Server access**:
A Member's access to one Media Server, from offered to active. SlipStream records what it shared or made and removes it when asked.
_In UI_: "access", as in "Remove access", and "can watch", as in "3 Members can watch".
_Avoid_: Invite, share (as the record)

**Path mapping**:
How one Media Server sees a library root, such as `/mnt/media` on the storage Node being `/data/media` to Plex.
_In UI_: "Path mapping".
_Avoid_: Path substitution, remote path mapping

**Fresh folder**:
A library folder that a Media Server has never scanned with a file in it. Plex and Jellyfin skip a report or a partial scan into one, so the first change into it waits until a full scan of its library has found a file there.
_In UI_: none. Say what waits and why.
_Avoid_: New root, empty library

**Media Server change**:
One change a Media Server is sent about one library path: a file added, replaced, removed or moved, a Sidecar added, replaced or removed, or a refresh of a whole library. The Controller keeps it until the server accepts it, and merges the changes on one path into one.
_In UI_: none. Say "change", such as "3 changes wait for Living room Plex".
_Avoid_: Notification, event, update, refresh (for the change itself)

**Carry**:
A Media Server change that puts an item's file at a new path, from a rename, an upgrade under a new name or a Move to another library root, and keeps every user's Watch data and the item's collections on the item at the new path. The change lists what the server couldn't keep as lost.
_In UI_: none. Say what the server kept and lost.
_Avoid_: Move (that is the Admin's job that sends one), restore, migration

**Watch data**:
What a Media Server records for each of its users about one item: played, play count, resume position, favorite, last played and rating.
_In UI_: What the server keeps, named one by one: watched, play count, resume position, favorite, rating.
_Avoid_: Watch history, user data

**Play history**:
The plays SlipStream records from each Media Server's sessions: who played which library file on which player, and whether the server direct played, direct streamed or transcoded it, stream by stream, with the reasons the server gave. It keeps them for "Keep play history for", 30 days by default.
_In UI_: "Direct play", "Direct stream", "Transcode" and "Hardware transcode", each with a definition on hover and tap.
_Avoid_: Session history, watch history, stats

**Playback report**:
What SlipStream shows about recent plays on each Media Server: which players' plays the server transcoded, why, and which Target would have played directly.
_In UI_: "Playback report".
_Avoid_: Stats, analytics, player profile

**Player**:
A device and app that plays from a Media Server, known by the id the server gives it.
_In UI_: "player".
_Avoid_: Client, device

**Scheduled task**:
A library-wide job the Controller starts on a cadence, which the Tasks page lists with its last run, result and next run. A due time on one object, a job an event queues, and watching a Node keeps up continuously are not Scheduled tasks.
_In UI_: "Task".
_Avoid_: Job, cron job, background task, timer

**Maintenance window**:
The daily stretch of the Controller's local time when updates install and the nightly Scheduled tasks, such as Backup and Library scan, start.
_In UI_: "Maintenance window".
_Avoid_: Update window, quiet hours

**Library scan**:
The Scheduled task that compares each library root with SlipStream's records, matches found files to Works, sends the rest to Pending imports and reads media info.
_In UI_: "Library scan".
_Avoid_: Rescan, library sync, refresh

**Import**:
The work of bringing a set of video files into one Work from a download, a folder in a library root or a folder the Admin picks. Each file fills the cells it beats, waits for the Admin or is left out with a reason.
_In UI_: "Import".
_Avoid_: Import job, processing, post-processing

**Pending import**:
An Import with a file that waits for the Admin, such as a download whose file isn't better than the one the Work has, or a folder a Library scan can't match to a Work with confidence.
_In UI_: "Pending imports".
_Avoid_: Unmatched, manual import queue, orphan

**Force into a Target**:
Fill a Target with a file that meets it but ranks lower than the file it holds.
_In UI_: "Force into" with the Target's name.
_Avoid_: Force import, downgrade

**Clone**:
A copy that shares its data on disk with the original until either file changes. Btrfs, XFS and APFS can clone.
_In UI_: "Clone".
_Avoid_: Reflink, CoW copy

**Metadata refresh**:
The Scheduled task that keeps every Work's record current from its metadata source.
_In UI_: "Metadata refresh".
_Avoid_: Metadata sync, rescan, update metadata

**Health**:
The list of what is wrong with SlipStream's own machinery right now, by Health category, each Health item at Warning or Error with what to do. Decisions about Works stay in Needs you.
_In UI_: "Health".
_Avoid_: Alerts, problems, issues, system status

**Health category**:
One feature's group of Health items, such as Storage, Nodes or Scheduled tasks, which that feature derives from its own state each time Health asks. A settings section names the categories whose dot it carries.
_In UI_: Each category's name.
_Avoid_: Health check, section

**Health item**:
One thing wrong in a Health category, keyed by its subject's own id, at Warning when something will go wrong if nobody acts or Error when something is wrong now, with what to do first and since when.
_In UI_: "Health item" in notification labels only, such as New Health item and Health item cleared. Elsewhere none. Show the item's own sentence.
_Avoid_: Alert, issue, warning (for an item at either level)

**Notification connection**:
One place an Admin's notifications go, through one provider: a Discord or Slack webhook, a Telegram chat, a Pushover, Gotify, ntfy, Pushbullet or Apprise target, email addresses, a Webhook URL or a Custom Script on one Node. It has its own events, and SlipStream sends each event it wants at least once and in order.
_In UI_: "connection" inside Notifications. Elsewhere the provider and the name, such as Discord 'Family'.
_Avoid_: Notifier, agent, channel, integration

**History entry**:
One thing that happened to the library, such as Grabbed, Imported or New Health item, recorded in the Write that makes it true and kept for "Keep history for". History lists it, and the bell and each Notification connection get the kinds they're turned on for.
_In UI_: "entry" on History, "event" in a connection's event list, and each kind's label.
_Avoid_: Alert, trigger, log line, notification (for the record itself)

**History**:
The Activity page that lists what happened to the library, newest first, kept for "Keep history for".
_In UI_: "History".
_Avoid_: Log, activity log, event log

## Acquisition

**Download Client**:
A torrent client that a downloading Node sends Grabs to, either its built-in one or an external one such as qBittorrent, Transmission, Deluge or rTorrent. An external client runs on its own, and its Node reaches it at one address.
_In UI_: "Download Client", or the client's product name.
_Avoid_: Downloader, torrent app

**Client folder**:
A folder as a Download Client reports it, paired with the path of the same folder on the client's Node.
_In UI_: "Remote path mapping", the Sonarr and Radarr term.
_Avoid_: Path mapping (that is a Media Server's), remote path mapping (except in UI copy)

**Attached torrent**:
A torrent that was already in an external Download Client when a Grab sent the same info hash. SlipStream tracks it without having added it, and removing it in SlipStream stops tracking it and leaves it in the client.
_In UI_: "Attached".
_Avoid_: Adopted torrent, existing torrent

**New Grabs go to**:
A downloading Node's setting that names the one Download Client its Grabs go to: its built-in client or one of its external clients that is on.
_In UI_: "New Grabs go to".
_Avoid_: Default client, grabs to

**Tracker**:
A site a torrent comes from and seeds to. It carries the site's seeding rule, the admin's seeding goals over it, hit and run state and its Tracker routing.
_In UI_: "Tracker" for the site, and lowercase "tracker" for an announce URL, as qBittorrent and Transmission say.
_Avoid_: Site, indexer (for the seeding side), tracker (for an announce URL, except in UI copy)

**Seeding rule**:
What a Tracker requires before a torrent may stop seeding: a ratio, a seeding time, and which of them clears it. SlipStream reads it from the minimum ratio and seeding time the Tracker's Releases state, and the Admin can set it.
_In UI_: "Seeding rule".
_Avoid_: floor, minimums, H&R rule

**Seeding goal**:
What the built-in client seeds a torrent to: a ratio, a time, how they combine (Whichever first, Both or Forever) and what happens when met.
_In UI_: "Seeding goal".
_Avoid_: Seed ratio target, cleanup mode, seeding rule (for the admin's goal)

**Hit and run**:
Stopping or removing a torrent before its Tracker's seeding rule is met. Each seeding torrent counts down to when removing it no longer counts as one.
_In UI_: "Hit and run".
_Avoid_: H&R, HnR

**Hit and run window**:
The time a private torrent owes its Tracker's seeding rule, from its first downloaded byte until the rule is met. It ends met, missed or released.
_In UI_: "Hit and run window", and "Needs N more seeding" for what is left. Never "owes".
_Avoid_: H&R window

**Indexer**:
One configured way to search one or more Trackers, such as a catalog definition, a ported site indexer, a Torznab feed or an RSS feed.
_In UI_: "Indexer".
_Avoid_: Tracker (for the search side), provider, source

**Indexer catalog**:
The list of site definitions the Controller downloads at runtime, from Prowlarr's catalog unless the Admin changes the address.
_In UI_: "Indexer catalog".
_Avoid_: Definitions repo, indexer list

**Definition**:
One catalog or custom file describing how to sign in to, search and download from one site.
_In UI_: "Definition".
_Avoid_: Cardigann file, indexer YAML

**Home Node**:
The one Searching Node an Indexer runs every query, login, RSS fetch, Test and .torrent fetch from.
_In UI_: "Home Node".
_Avoid_: Search node, proxy

**Challenge solver**:
A service speaking FlareSolverr's API, such as Byparr, that clears a site's browser challenge for a Searching Node. Each Searching Node has its own, and none is bundled.
_In UI_: "Challenge solver".
_Avoid_: FlareSolverr (as a generic term), captcha solver

**Tracker routing**:
Which Download Client a Tracker's Grabs go to, set per Node an Indexer of the Tracker searches from. By default it is the built-in client on that Node.
_In UI_: "Tracker routing".
_Avoid_: Client mapping, routing rule

**Release**:
What an Indexer lists for download: a title with the facts it states about itself and one info hash.
_In UI_: "release", lowercase in sentences.
_Avoid_: Result, torrent (for the listing), NZB

**Magnet link**:
A link that names a Release by its info hash, with an optional name and trackers, in place of its .torrent. While the Grab is fetching, the Downloading Node gets the torrent's metadata from peers, and the Grab fails if no peer sends it within the metadata wait.
_In UI_: "Magnet link".
_Avoid_: Magnet URI, magnet (alone, in UI)

**Want**:
One Edition at one Target of a monitored, released Item, still listed by its source, that no file fills, or whose filling file is below the Target's cutoff or below its upgrade-until score, and that no unfinished Grab holds.
_In UI_: none. Say what is missing, such as "missing episodes" or "3 more files".
_Avoid_: Missing item, wanted cell, slot

**Grab**:
The decision to download one Release for a Work and Edition, recorded with the Items it covers and the Targets it is meant to fill, and sent to one Download Client. Once the download completes, an Import takes its files.
_In UI_: "Grab" as a noun on Activity and buttons. A sentence may say "download".
_Avoid_: Snatch, download (for the decision)

**Hold**:
How long a Target waits after a Release is published before grabbing it automatically, so a better one can arrive first. A Release that would end the Target's upgrades doesn't wait.
_In UI_: "Hold", with its hover definition. While it runs, the Work reads "Waiting for a better release until 22:40".
_Avoid_: Delay, delay profile

**Automatic search**:
A search whose Grabs SlipStream chooses itself, whether the Admin's Search, a trigger such as an add or a failed Grab, Search all, Backlog search or RSS sync started it.
_In UI_: "Automatic search" as the Settings section. The Admin's button is "Search".
_Avoid_: Auto search, search now

**Search group**:
The Items of a Work that one query searches, such as a season. Each Content Type defines its own.
_In UI_: The group's own name, such as "season 3". Never "search group".
_Avoid_: Search unit, batch, season (as the generic term)

**Search all**:
Searching every Work that Wanted lists under its filter, or every Work picked in the Library, behind the Admin's searches of one Work.
_In UI_: "Search all" as a search's trigger. Wanted's button names its count in the Content Type's noun, such as "Search 140 series", or "Search 140 Works" on All.
_Avoid_: Bulk search, search missing

**Backlog search**:
The scheduled search for every Want that is due, missing or upgradable, whose searches space further apart while it stays unfound, except in its first days after release. At its longest gap it also looks for listed cuts not yet seen for an Item.
_In UI_: "Backlog search".
_Avoid_: Sweep, scheduled search, auto search

**RSS sync**:
The Scheduled task that reads each Indexer's newest uploads and grabs the ones that meet a Want.
_In UI_: "RSS sync".
_Avoid_: RSS refresh, feed sync

**Search again**:
Searching a cell again after its Grab failed, by SlipStream at once or by the Admin on a stopped cell. A cell stops once "Stop searching again after" failed Grabs land in its window, at any stage, import included, and then only Backlog search searches it.
_In UI_: "Search again", the action on a failed Grab's card.
_Avoid_: Retry (for cells), redownload

**Wanted**:
The Library page that lists every Work with a missing, upgradable, failed or gone cell, with when SlipStream last searched it and when it searches next.
_In UI_: "Wanted".
_Avoid_: Missing page, Missing and Upgradable, wanted list

**Calendar**:
The Library page that shows the release dates of the library's movies and episodes by day, with whether each is in the library.
_In UI_: "Calendar".
_Avoid_: Schedule, upcoming

**Not found**:
A Needs you row for missing Items whose last search found nothing to grab. It is not a cell state.
_In UI_: "not found", after the Items, such as "S02E01 to S02E08 not found".
_Avoid_: Unfound, no results

**Manual search**:
A search an Admin opens to see the Releases for part of a Work and choose what to grab. It asks every Indexer that is on, leads with a Plan and grabs nothing until the Admin presses Grab.
_In UI_: "Manual search".
_Avoid_: Interactive search, search by hand

**Plan**:
In Manual search, the fewest Grabs that fill the missing, failed and upgradable cells in scope, one Release per row, chosen as automatic search would choose them.
_In UI_: "Plan".
_Avoid_: Bulk grab, cover, suggestion

**Custom Format**:
A named set of conditions on what a release says about itself, such as its title, Release group or source, that each Target scores. Its conditions have Radarr's and Sonarr's shape, so a format from TRaSH Guides, Radarr or Sonarr pastes in and runs as written.
_In UI_: "Custom Format".
_Avoid_: Preferred word, release profile, tag

**Score**:
The sum of a Target's scores for the Custom Formats a Release matches. Each Target also holds a minimum score, below which a Release isn't grabbed automatically, an upgrade-until score, which a file at the cutoff keeps upgrading towards, and a minimum increment, by which a Release must beat the file's score to replace it.
_In UI_: "Score".
_Avoid_: Weight, points

**Revision**:
How many times a Release was re-released to fix it, read from PROPER, REPACK, RERIP, vN and REAL.
_In UI_: "Proper" or "Repack", or "Revision" in the Target editor.
_Avoid_: Version (for this)

**Release group**:
The group a Release names as its maker, such as FLUX in a title ending -FLUX.
_In UI_: "Release group".
_Avoid_: Uploader

**Refusal**:
Why a Release can't fill a Want, such as a title that doesn't match the Work or a Score below the Target's minimum.
_In UI_: The refusal's own sentence. Never "Refusal".
_Avoid_: Rejection, block

**TRaSH Guides**:
The community guide whose Custom Formats, scores, quality sizes and naming SlipStream imports and keeps tracking.
_In UI_: "TRaSH Guides".
_Avoid_: TRaSH profile (for a Target), Recyclarr

## Subtitles

**Subtitle Provider**:
An external service SlipStream searches and downloads subtitles from, such as SubDL.
_In UI_: "Subtitle Provider".
_Avoid_: Subtitle source, subtitle indexer

**Wanted Language**:
One entry in a Profile's subtitle list: a language, whether full or forced only, a hearing-impaired preference, and whether it is wanted always or only when no audio track is in that language.
_In UI_: "Language".
_Avoid_: Language profile, subtitle profile

**Sidecar**:
A subtitle file stored beside one library file under its name and a tail such as `en.forced.srt`, which SlipStream downloaded, the Admin uploaded or a survey found there. A Sidecar belongs to that file, not to its Edition.
_In UI_: "Subtitle file".
_Avoid_: External subtitle

**Speech Map**:
Where speech falls in one library file's audio, built on the Node that holds the file while one of its wants has no Sidecar, and used for Timing sync.
_In UI_: none.
_Avoid_: VAD map, audio fingerprint

**Timing sync**:
Shifting and stretching a subtitle's cue times to match a library file's Speech Map. It never changes the cue text.
_In UI_: "Timing sync".
_Avoid_: Alignment, resync

**Mark bad**:
Removing a Sidecar as wrong, which deletes its file, blocks that candidate for that file and searches again at once. An Admin marks one bad from a file's subtitles after a question naming the file, and a Subtitle report does it without one.
_In UI_: "Mark bad".
_Avoid_: Reject, Blocklist (for a subtitle)

**Subtitle search**:
The scheduled search for wanted subtitles that no Sidecar or embedded track meets yet, within each Subtitle Provider's allowance, spacing each one's searches further apart while nothing passes Timing sync.
_In UI_: "Subtitle search".
_Avoid_: Subtitle sweep, subtitle auto search

**Manual subtitle search**:
A search an Admin opens on one file and Wanted Language to see candidate subtitles, ranked by their reasons, and pick one, which downloads it and runs Timing sync.
_In UI_: "Manual search", on a subtitle row.
_Avoid_: Search by hand

## Identity

**Account**:
One person's identity in SlipStream, signed in by its email with a PIN or a passkey. Admins and Members are both Accounts.
_In UI_: "Account".
_Avoid_: User, portal user, login

**Admin**:
An Account holding the admin permission. At least one Admin always exists.
_In UI_: "Admin".
_Avoid_: Administrator, owner, superuser

**Member**:
An Account without the admin permission.
_In UI_: "Member".
_Avoid_: Portal user, user, guest

**Setup code**:
A one-time code a Node prints to its console and log, which unlocks its own page to create the first Admin, join a SlipStream or restore a Backup on a fresh install, or to pair a Worker out of reach of its Controller for a while with another one. A recovery sheet names it as the way into a new machine's restore.
_In UI_: "Setup code".
_Avoid_: Bootstrap token, install key

**First setup**:
The guided steps a new Controller's first Admin walks through after the claim, open until that Admin finishes them.
_In UI_: "Setup".
_Avoid_: Onboarding, setup wizard, first run (for the steps after the claim)

**Signup link**:
A single-use link an Admin issues so one person can create their Account.
_In UI_: "Signup link".
_Avoid_: Invite, invitation, Join Code

**Reset link**:
A single-use link an Admin issues so a person who lost their credentials can set a new PIN and passkey.
_In UI_: "Reset link".
_Avoid_: Recovery email, password reset

## Requests

**Request**:
An Account's ask for a Work, or some of its levels or Items, to be added and monitored, which an Admin approves or auto-approval admits. Its stage is Pending, Approved, Searching, Downloading, Processing, Available or Denied, and Delayed flags it.
_In UI_: "Request", and each stage's name.
_Avoid_: Order, ticket, portal request; Requested, Coming, Ready (for a stage)

**Watcher**:
An Account following another Account's Request, who gets its notices without spending a Request and inherits it if the requester cancels.
_In UI_: none. Say "following", such as "You're following this request".
_Avoid_: Subscriber, follower

**Request limit**:
How many of a Member's Requests per week, per Content Type, are approved automatically. A Request over the limit waits for an Admin.
_In UI_: "Request limit".
_Avoid_: Quota, allowance

**Playable**:
An Item with any library file a Media Server can play, Kept files included.
_In UI_: "Ready to watch".
_Avoid_: Downloaded, complete, available (for an Item)

**Delayed**:
A flag on a Request whose released Items have stayed unplayable too long or hold a failed cell. Search keeps running underneath it.
_In UI_: "Delayed".
_Avoid_: Failed, stuck

**Subtitle report**:
A Member's report that one language's subtitles on an Item are wrong, which SlipStream acts on with Mark bad.
_In UI_: "Report subtitles".
_Avoid_: Complaint, issue, flag

## Development

**Offline mode**:
Running SlipStream against the Fake world instead of the real services it talks to, for developers, agents and CI.
_In UI_: none. Developers only.
_Avoid_: Dev mode, developer mode, mock mode

**Fake world**:
The set of fakes standing in for every external boundary SlipStream crosses, such as metadata providers, trackers and Media Servers, together with the Nodes started against them.
_In UI_: none. Developers only.
_Avoid_: Mocks, sandbox, simulation

**Scenario**:
One declared state and script for the Fake world: the Works, releases, swarms, servers and providers it holds, and the faults it plays on cue.
_In UI_: none. Developers only.
_Avoid_: Fixture, seed data, test case
