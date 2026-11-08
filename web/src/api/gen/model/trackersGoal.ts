import type { TrackersGoalAction } from "./trackersGoalAction.ts";
import type { TrackersGoalCombine } from "./trackersGoalCombine.ts";

export interface TrackersGoal {
  /**
   * What happens to the torrent once the goal is met, where remove-delete also deletes its download copy; null when combine is forever
   * @nullable
   */
  action: TrackersGoalAction;
  /** first is whichever of the ratio and seeding time is reached first; both needs both of them; forever seeds until an Admin removes the torrent. A goal that ends counts a ratio, a seeding time or both, and with only one of them first and both each wait for it */
  combine: TrackersGoalCombine;
  /**
   * At most two decimals; null when combine is forever or the goal doesn't count ratio
   * @nullable
   */
  ratio: number | null;
  /**
   * Seconds; null when combine is forever or the goal doesn't count seeding time
   * @nullable
   */
  seedSeconds: number | null;
}
