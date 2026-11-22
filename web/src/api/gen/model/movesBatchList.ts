import type { MovesBatchRow } from "./movesBatchRow.ts";

export interface MovesBatchList {
  /** Open ones first, then newest first */
  batches: MovesBatchRow[];
  /** The after of the next page; absent on the last */
  next?: string;
}
