import type { GrabsManualCell } from "./grabsManualCell.ts";

export interface GrabsManualScope {
  /** The one Item and Target it plans for, absent for a wider scope */
  cell?: GrabsManualCell;
  itemIds: string[];
  /** Each level stands for every Item under it */
  levelIds: string[];
}
