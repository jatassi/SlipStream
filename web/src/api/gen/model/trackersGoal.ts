import type { TrackersGoalAction } from "./trackersGoalAction.ts";
import type { TrackersGoalCombine } from "./trackersGoalCombine.ts";

export interface TrackersGoal {
  /**
   * What happens to the torrent once the goal is met, where remove-delete also deletes its download copy; null when combine is forever
   * @nullable
   */
  action: TrackersGoalAction;
  /** first is whichever of the ratio and seeding time is reached first; both needs both of them; forever seeds until an Admin removes the torrent */
  combine: TrackersGoalCombine;
  /**
   * At most two decimals; null when combine is forever
   * @nullable
   */
  ratio: number | null;
  /**
   * Seconds; null when combine is forever
   * @nullable
   */
  seedSeconds: number | null;
}
