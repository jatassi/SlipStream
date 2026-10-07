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
A cut of an Item, such as Standard, Theatrical, Extended or Director's. Every Item has at least one Edition, and may want several, each wanted at every Target in its Work's Profile. A release with no cut named holds Standard.
_Avoid_: Version, cut

**Target**:
A library-wide, named encoding goal, such as 1080p low bitrate or 2160p HDR, with its own quality rules, Custom Format scores, cutoff and upgrades. A Target is either downloaded or derived by transcoding another Target. One file meets every Target whose rules it passes.
_Avoid_: Rendition, slot, version, quality version, variant

**Profile**:
The set of Targets a Work wants, each with how it is sourced: downloaded, derived from another Target, or downloaded first and derived after a grace period. It also sets the default Editions wanted, which a Work can override. Every Work has one, defaulting per Content Type.
_Avoid_: Rendition profile, quality profile, slot

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
Joining a new Node to a SlipStream with a Join Code from the Controller, after which the two Nodes trust each other's keys.
_Avoid_: Registration, enrollment, adoption

**Join Code**:
A short-lived, single-use code the Controller issues so one new Node can pair with it.
_Avoid_: Invite, token, pairing key

**Handover**:
Moving the Controller to another paired Node, after which every Worker follows the new Controller.
_Avoid_: Failover, promotion, migration

**Role**:
A job the admin assigns to one or more Nodes. The Roles are Searching, Downloading, Storage, Transcoding, Subtitles, Metadata, Media Server link, Notifying, Relaying and Serving. Work that reads the library database on every call stays on the Controller and is never a Role.
_Avoid_: Service, capability, purpose

**Role Group**:
A named bundle of Roles that the admin ticks together when assigning, such as Acquire for Searching and Downloading. A Node holds Roles, never Role Groups.
_Avoid_: Preset, profile, tier

**Serving**:
The Role that serves the UI and portal on a Node, passing every request to the Controller. The Controller always serves too.
_Avoid_: Hosting, web; never the Media Server's serving of the library

**Transfer**:
A one-way copy of files from one Node to another for a job, such as a completed download going to the storage Node that imports it. The source keeps its copy, and the receiving Node is the destination.
_Avoid_: Sync, mirror, push; target for the destination

**Media Server**:
An external player-facing server that serves the library to players, such as Plex, Jellyfin or Emby.
_Avoid_: Plex (as a generic term)

## Acquisition

**Download Client**:
A torrent client that a downloading Node sends grabs to, either the built-in one or an external one such as qBittorrent.
_Avoid_: Downloader, torrent app

**Custom Format**:
A named set of conditions on what a release says about itself, such as its title, release group or source, that each Target scores. A release's score for a Target is the sum of the scores of the Custom Formats it matches.
_Avoid_: Preferred word, release profile, tag

**TRaSH Guides**:
The community guide whose Custom Formats, scores, quality sizes and naming SlipStream imports and keeps tracking.
_Avoid_: TRaSH profile (for a Target), Recyclarr

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
