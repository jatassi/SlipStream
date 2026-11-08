import type { TrackersAct } from "./trackersAct.ts";
import type { TrackersKept } from "./trackersKept.ts";
import type { TrackersWait } from "./trackersWait.ts";

export interface TrackersGoalsPreview {
  /** The torrents whose goal acts, in added order */
  acts: TrackersAct[];
  /** The torrents removed keeping their download copy, in added order */
  kept: TrackersKept[];
  /** The torrents whose goal is met and waits, in added order */
  waits: TrackersWait[];
}
