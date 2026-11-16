import type { GrabsClientChoice } from "./grabsClientChoice.ts";
import type { GrabsLiftKey } from "./grabsLiftKey.ts";

export interface GrabsGrabPick {
  /** The Grab downloading an Item and Target it covers, canceled first */
  cancelGrabId?: string;
  /**
   * The Items and Targets the row or the release showed
   * @maxItems 100
   */
  cells: string[];
  /** Absent to go where Tracker routing sends it */
  client?: GrabsClientChoice;
  /**
   * The Edition the row showed
   * @minLength 1
   */
  edition?: string;
  /**
   * The lifts the row showed
   * @maxItems 100
   */
  lifts: GrabsLiftKey[];
  releaseId: string;
  /** Remove the Release from the Blocklist and grab it */
  unblock?: boolean;
}
