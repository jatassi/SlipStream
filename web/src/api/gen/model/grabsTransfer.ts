import type { GrabsTransferState } from "./grabsTransferState.ts";
import type { GrabsWait } from "./grabsWait.ts";

export interface GrabsTransfer {
  /**
   * Seconds until its file is verified; 0 while unknown
   * @minimum 0
   */
  estimateSeconds: number;
  /**
   * Bytes a second it copies at, else the last speed between its two Nodes; 0 while unknown
   * @minimum 0
   */
  rate: number;
  /** copying also covers a file that is verified and waits to be imported */
  state: GrabsTransferState;
  /** What it waits for; absent unless its state is waiting */
  wait?: GrabsWait;
}
