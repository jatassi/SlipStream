import type { TrackersPlannedTorrent } from "./trackersPlannedTorrent.ts";
import type { TrackersWaitWhy } from "./trackersWaitWhy.ts";

export interface TrackersWait {
  torrent: TrackersPlannedTorrent;
  /** What holds its met goal back: removing, it is being removed; resumed, an Admin resumed it after its goal paused it, so it seeds with no end; ran, its goal already acted on it; incomplete, its selected files aren't all there; grab-downloading or grab-importing, a Grab on it is still downloading or importing; reconciling, SlipStream is reconciling the library after a restore */
  why: TrackersWaitWhy;
}
