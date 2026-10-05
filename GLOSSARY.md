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
A cut of an Item, such as Standard, Theatrical, Extended or Director's. Every Item has at least one Edition, and may want several, each held in every Rendition its Work wants. A release with no cut named holds Standard.
_Avoid_: Version, cut

**Rendition**:
A library-wide, named encoding target, such as 1080p low bitrate or 2160p HDR, with its own quality rules, cutoff and upgrades. A Rendition is either downloaded or derived by transcoding another Rendition. One file fills every Rendition whose rules it meets.
_Avoid_: Slot, version, quality version

**Rendition profile**:
The set of Renditions a Work wants, each with how it is sourced: downloaded, derived from another Rendition, or downloaded first and derived after a grace period. It also sets the default Editions wanted, which a Work can override. Every Work has one, defaulting per Content Type.
_Avoid_: Quality profile, slot

**Blocklist**:
Releases known to be bad, keyed by torrent info hash, that no Item ever grabs again until removed by hand.
_Avoid_: Blacklist, ignore list

## Topology

**Node**:
One SlipStream install running on one machine.
_Avoid_: Instance, server, peer

**Controller**:
The one Node that owns the library database and serves the UI. A single-machine setup is a Controller holding every Role.
_Avoid_: Primary, master, hub

**Role**:
A job a Node is assigned, such as downloading, searching, transcoding, storage or relaying.
_Avoid_: Service, capability, purpose

**Media Server**:
An external player-facing server that serves the library to players, such as Plex, Jellyfin or Emby.
_Avoid_: Plex (as a generic term)

## Acquisition

**Download Client**:
A torrent client that a downloading Node sends grabs to, either the built-in one or an external one such as qBittorrent.
_Avoid_: Downloader, torrent app
