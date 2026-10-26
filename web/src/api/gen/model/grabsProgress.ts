import type { GrabsTransfer } from "./grabsTransfer.ts";

export interface GrabsProgress {
  /**
   * Bytes downloaded, or received and verified while transferring
   * @minimum 0
   */
  done: number;
  /**
   * Bytes wanted
   * @minimum 0
   */
  total: number;
  /** How the Transfer stands, absent unless it is transferring */
  transfer?: GrabsTransfer;
}
