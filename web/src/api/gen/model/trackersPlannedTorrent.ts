import type { TrackersTrackerRef } from "./trackersTrackerRef.ts";

export interface TrackersPlannedTorrent {
  id: string;
  name: string;
  /** null for a torrent with no known Tracker */
  tracker: TrackersTrackerRef;
}
