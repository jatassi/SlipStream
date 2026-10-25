import type { GrabsReason } from "./grabsReason.ts";
import type { GrabsRefusal } from "./grabsRefusal.ts";
import type { GrabsSearchOutcome } from "./grabsSearchOutcome.ts";

export interface GrabsSearch {
  /**
   * The releases in the Indexer's response
   * @minimum 0
   */
  candidates: number;
  /**
   * Null while the run searches
   * @nullable
   */
  finishedAt: string | null;
  /**
   * The Grab the run made, null when none
   * @nullable
   */
  grabId: string | null;
  id: string;
  indexerId: string;
  itemId: string;
  /** The grabbed release's notes, such as size_not_checked, for each Target it was grabbed for */
  notes: GrabsReason[];
  /**
   * Null while the run searches
   * @nullable
   */
  outcome: GrabsSearchOutcome;
  /** The releases it didn't grab for a reason, in the Indexer's order; a release it outranked isn't refused */
  refused: GrabsRefusal[];
  startedAt: string;
}
