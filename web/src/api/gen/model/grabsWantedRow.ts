import type { GrabsItemRange } from "./grabsItemRange.ts";
import type { GrabsWantedRowLastResult } from "./grabsWantedRowLastResult.ts";
import type { GrabsWantedRowOthers } from "./grabsWantedRowOthers.ts";
import type { GrabsWantedRowReason } from "./grabsWantedRowReason.ts";
import type { GrabsWantedRowState } from "./grabsWantedRowState.ts";
import type { GrabsWantedWork } from "./grabsWantedWork.ts";

export interface GrabsWantedRow {
  /**
   * How its last search finished; null before one did
   * @nullable
   */
  lastResult: GrabsWantedRowLastResult;
  /**
   * When its last search finished; null before one did
   * @nullable
   */
  lastSearchedAt: string | null;
  /**
   * When Backlog search searches it next; null for a gone row, which only a press searches, and while Backlog search is off
   * @nullable
   */
  nextSearchAt: string | null;
  /** The count of Items under each other reason, by reason */
  others: GrabsWantedRowOthers;
  /** The Items it lists, as runs in tree order; empty for a Work of one Item */
  ranges: GrabsItemRange[];
  /** The reason the row lists, the first in this order the Work has */
  reason: GrabsWantedRowReason;
  state: GrabsWantedRowState;
  work: GrabsWantedWork;
}
