import type { GrabsReason } from "./grabsReason.ts";
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
  /** The Grabs the run made, then those it joined: the decision's, or the rerun's once a dropped Grab reran it */
  grabIds: string[];
  id: string;
  /** Each Indexer the run asked or skipped, by priority */
  indexers: GrabsSearchIndexer[];
  /** The Items the run searched */
  itemIds: string[];
  /** The grabbed Release's notes, such as size_not_checked, for each Target it was grabbed for */
  notes: GrabsReason[];
  /**
   * Null while the run searches. waiting holds a cell for a better Release. nothing_found is a run that found no Release, and nothing_acceptable one whose Releases fit no Target. no_indexer is a run that no Indexer responded to, and no_root one whose Work had no reachable root. listed is a Manual search's run, which takes nothing.
   * @nullable
   */
  outcome: GrabsSearchOutcome;
  startedAt: string;
}
