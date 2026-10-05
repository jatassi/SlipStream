# SlipStream

SlipStream finds, downloads, processes and files media into a library that media servers play, across one or more machines.

## Media

**Content Type**:
A pluggable kind of media with its own shape and metadata sources, such as movies or TV.
_Avoid_: Module, media type

**Edition**:
A cut of a work, such as theatrical, extended or director's.
_Avoid_: Version, cut

**Rendition**:
An encoding target for a work, such as 1080p low bitrate or 2160p HDR. A Rendition is either downloaded or derived by transcoding another Rendition.
_Avoid_: Slot, version, quality version

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
