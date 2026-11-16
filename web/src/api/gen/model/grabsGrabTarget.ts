import type { GrabsGrabTargetSettled } from "./grabsGrabTargetSettled.ts";

export interface GrabsGrabTarget {
  /** An Admin forced it: a file goes in even when it isn't an upgrade */
  forced: boolean;
  itemId: string;
  /**
   * How the Grab finished at this Target: it placed a file, it placed none, or the Admin declined it. Null while the Grab is still downloading for it
   * @nullable
   */
  settled: GrabsGrabTargetSettled;
  targetId: string;
}
