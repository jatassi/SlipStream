import type { TransfersTransfer } from "./transfersTransfer.ts";

export interface TransfersTransferList {
  /** Oldest first */
  transfers: TransfersTransfer[];
}
