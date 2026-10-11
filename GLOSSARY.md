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
A cut of an Item, such as Standard, Theatrical, Extended or Director's. Every Item has at least one Edition. When its Work's list names Standard or any cut, an Item always wants that, and wants each other Edition on the list only once a release or file of it has been seen for the Item. A list naming neither wants its Editions outright. Each wanted Edition is wanted at every Target in its Work's Profile. A release with no cut named holds Standard.
_Avoid_: Version, cut

**Target**:
A library-wide, named encoding goal, such as 1080p low bitrate or 2160p HDR, with its own quality rules, Custom Format scores, cutoff and upgrades. A Target is either downloaded or derived from another Target by a Conversion. One file meets every Target whose rules it passes.
_Avoid_: Rendition, slot, version, quality version, variant

**Fills**:
A file fills a Target for an Edition when it is the one file that Target holds for that Edition. A file can meet a Target without filling it, when another file ranks higher there.
_Avoid_: Holds, occupies, assigned to

**Profile**:
The set of Targets a Work wants, each with how it is sourced: downloaded, derived from another Target, or downloaded first and derived after a grace period. It also sets the default list of Editions wanted, which a Work can replace with its own list, and the Wanted Languages for subtitles. Every Work has one, defaulting per Content Type.
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

**Pairing**:
Joining a new Node to a SlipStream with a Join Code from the Controller, after which the two Nodes trust each other's keys. A Node the Controller already knows by its key keeps its Roles when it pairs again.
_Avoid_: Registration, enrollment, adoption

**Join Code**:
A short-lived, single-use code the Controller issues so one new Node can pair with it.
_Avoid_: Invite, token, pairing key

**Handover**:
Moving the Controller to another paired Node by shipping it a Backup, after which every Worker follows the new Controller.
_Avoid_: Failover, promotion, migration

**Backup**:
An encrypted copy of the Controller's library database, from which a Node can be restored as Controller. It never holds the Controller's key.
_Avoid_: Config backup, snapshot, dump, export

**Backup destination**:
A folder on a Node that receives a copy of every Backup.
_Avoid_: Backup target, backup location, remote

**Role**:
A job the admin assigns to one or more Nodes. The Roles are Searching, Downloading, Storage, Converting, Subtitles, Metadata, Media Server link, Notifying, Relaying and Serving. Work that reads the library database on every call stays on the Controller and is never a Role. Every Role but Relaying can sit on any number of Nodes.
_Avoid_: Service, capability, purpose

**Role Group**:
A named bundle of Roles that the admin ticks together when assigning, such as Acquire for Searching and Downloading. A Node holds Roles, never Role Groups.
_Avoid_: Preset, profile, tier

**Serving**:
The Role that serves the UI and portal on a Node, passing every request to the Controller. The Controller always serves too.
_Avoid_: Hosting, web; never the Media Server's serving of the library

**Relaying**:
The Role that accepts connections from the internet on a Node, usually a rented VPS, and forwards each one to the Node and port it maps to, for a home network that can't accept them itself. At most one Node relays, and that Node is the Relay Node.
_Avoid_: Tunnel, proxy, gateway

**Reachability diagnostic**:
The check that tells the admin whether the UI, portal, each Media Server and each named port can be reached from the internet, and what to do if not: open or map a port, or turn on Relaying.
_Avoid_: Port check, firewall check, connectivity test

**Transfer**:
A one-way copy of files from one Node to another for a job, such as a completed download going to the storage Node that imports it. The source keeps its copy, and the receiving Node is the destination.
_Avoid_: Sync, mirror, push; target for the destination

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

## Acquisition

**Download Client**:
A torrent client that a downloading Node sends grabs to, either the built-in one or an external one such as qBittorrent.
_Avoid_: Downloader, torrent app

**Tracker**:
A site a torrent comes from and seeds to. It carries the site's seeding rule, the admin's seeding goals over it, hit and run state and its Tracker routing. A Tracker outlives every Indexer that reaches it.
_Avoid_: Site, indexer (for the seeding side)

**Indexer**:
One configured way to search one or more Trackers, such as a catalog definition, a ported site indexer, a Torznab feed or an RSS feed. Each Indexer searches from one home Node.
_Avoid_: Tracker (for the search side), provider, source

**Tracker routing**:
Which Download Client a Tracker's Grabs go to.
_Avoid_: Client mapping, routing rule

**Release**:
What an Indexer lists for download: a title with the facts it states about itself and one info hash.
_Avoid_: Result, torrent (for the listing), NZB

**Want**:
One Edition at one Target of a monitored, available Item that no file fills, or whose filling file is below the Target's cutoff. Every search and RSS sync tries to satisfy Wants.
_Avoid_: Missing item, wanted cell, slot

**Grab**:
The decision to download one Release for a Work and Edition, recorded with the Targets it is meant to fill and sent to one Download Client.
_Avoid_: Snatch, download (for the decision)

**Hold**:
How long a Target waits after a Release is published before grabbing it automatically, so a better one can arrive first.
_Avoid_: Delay, delay profile

**Backlog search**:
The scheduled search that works through Wants nothing has found yet, spacing each Want's searches further apart while it stays unfound.
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
A one-time code a fresh install prints to its console and log, which creates the first Admin.
_Avoid_: Bootstrap token, install key

**Signup link**:
A single-use link an Admin issues so one person can create their Account.
_Avoid_: Invite, invitation, Join Code

**Reset link**:
A single-use link an Admin issues so a person who lost their credentials can set a new PIN and passkey.
_Avoid_: Recovery email, password reset

## Requests

**Request**:
A Member's ask for a Work, or some of its levels or Items, to be added and monitored, which an Admin approves or auto-approval admits. A Request is Available once every requested Item is Playable.
_Avoid_: Order, ticket, portal request

**Watcher**:
A Member following another Member's Request, who gets its notices without spending a Request and inherits it if the requester cancels.
_Avoid_: Subscriber, follower

**Request limit**:
How many of a Member's Requests per week, per Content Type, are approved automatically. A Request over the limit waits for an Admin.
_Avoid_: Quota, allowance

**Playable**:
An Item with any library file a Media Server can play, including a file kept after its Edition stopped being wanted.
_Avoid_: Downloaded, complete, available (for an Item)

**Delayed**:
A flag on a Request whose released Items have stayed unplayable too long or whose requested cell failed. Search keeps running underneath it.
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
