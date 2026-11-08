import type { TrackersActAction } from "./trackersActAction.ts";
import type { TrackersPlannedTorrent } from "./trackersPlannedTorrent.ts";

export interface TrackersAct {
  action: TrackersActAction;
  torrent: TrackersPlannedTorrent;
}
