# Media Servers pick the version, and SlipStream only advises

When an Item has files for several Targets, the Media Server decides which one a player gets. SlipStream reads no player's device profile and changes no server's version order. Jellyfin keeps its own order, highest resolution first. Instead, SlipStream watches each Media Server's sessions, which it already does to suspend Conversions, and keeps 30 days of play history. From that history it shows a Playback report: which players transcoded, the reasons the server gave, and which Target would have played without transcoding. It suggests a Target to add. The ticket had asked which player capabilities decide the Target a player gets. The answer is that none do. Decided in [Media Server contract](https://github.com/jatassi/SlipStream/issues/22), Q20 and Q21. Research in [Plex, Jellyfin and Emby APIs](https://github.com/jatassi/SlipStream/issues/21).

## Considered Options

- **Read player capabilities and steer each player to a Target.** Rejected. No server lets a third party choose the version for a player. Only Jellyfin exposes device profiles, and each one is only as complete as the client app made it. Plex and Emby report only what happened in a session.
- **Read Jellyfin device profiles to predict direct play before anyone plays.** Rejected. It adds a Jellyfin-only path for a prediction the play history already gives on all three servers.
- **Mark one Target per Profile as the first version, and name its file so Jellyfin plays it first.** Rejected by the chef. Jellyfin plays its first version and transcodes it rather than switch, so a weak player gets the 2160p file transcoded. The admin keeps that default and sees the cost in the Playback report.

## Consequences

- The Controller stores 30 days of session history per Media Server.
- On Jellyfin, a weak player may transcode the highest-resolution file even when a lower Target would play directly. The Playback report shows it.
- Target design stays the admin's job, informed by the report rather than automated.
