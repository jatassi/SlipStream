import type { GrabsGrabTargetSettled } from "./grabsGrabTargetSettled.ts";

export interface GrabsGrabTarget {
  /** An Admin forced it: a file goes in even when it isn't an upgrade */
  forced: boolean;
  itemId: string;
  /**
   * How the Grab finished with the cell: a file placed into it, no file filled it, or the Admin declined it. Null while the Grab holds it
   * @nullable
   */
  settled: GrabsGrabTargetSettled;
  targetId: string;
}
