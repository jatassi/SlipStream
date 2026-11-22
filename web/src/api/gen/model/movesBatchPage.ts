import type { MovesBatchRow } from "./movesBatchRow.ts";
import type { MovesMoveUnit } from "./movesMoveUnit.ts";

export interface MovesBatchPage {
  batch: MovesBatchRow;
  /** The after of the next page; absent on the last */
  next?: number;
  /** In the batch's order */
  units: MovesMoveUnit[];
}
