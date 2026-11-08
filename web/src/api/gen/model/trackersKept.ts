import type { TrackersKeptWhy } from "./trackersKeptWhy.ts";
import type { TrackersPlannedTorrent } from "./trackersPlannedTorrent.ts";

export interface TrackersKept {
  torrent: TrackersPlannedTorrent;
  /** not-imported: a Grab on it didn't import, so removing it keeps the download copy; draining: its Node is draining Downloading, so it is removed keeping its download copy in place of a pause */
  why: TrackersKeptWhy;
}
