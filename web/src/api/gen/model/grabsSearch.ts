import type { GrabsReason } from "./grabsReason.ts";
import type { GrabsRefusal } from "./grabsRefusal.ts";
import type { GrabsSearchIndexer } from "./grabsSearchIndexer.ts";
import type { GrabsSearchOutcome } from "./grabsSearchOutcome.ts";

export interface GrabsSearch {
  /**
   * The Releases the Indexers found, each Release listed twice counted once
   * @minimum 0
   */
  candidates: number;
  /**
   * Null while the run searches
   * @nullable
   */
  finishedAt: string | null;
  /**
   * The run's Grab: the decision's, or the rerun's once a dropped Grab reran it; null when none
   * @nullable
   */
  grabId: string | null;
  id: string;
  /** Each Indexer the run asked or skipped, by priority */
  indexers: GrabsSearchIndexer[];
  itemId: string;
  /** The grabbed Release's notes, such as size_not_checked, for each Target it was grabbed for */
  notes: GrabsReason[];
  /**
   * Null while the run searches
   * @nullable
   */
  outcome: GrabsSearchOutcome;
  /** The Releases it didn't grab for a reason, in the Indexers' order; a Release it outranked isn't refused */
  refused: GrabsRefusal[];
  startedAt: string;
}
