import type { GrabsReason } from "./grabsReason.ts";
import type { GrabsSearchIndexer } from "./grabsSearchIndexer.ts";
import type { GrabsSearchLane } from "./grabsSearchLane.ts";
import type { GrabsSearchOutcome } from "./grabsSearchOutcome.ts";
import type { GrabsSearchTrigger } from "./grabsSearchTrigger.ts";

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
  /**
   * The Grabs it made or joined when it finished, still counted once one is pruned
   * @minimum 0
   */
  grabs: number;
  /**
   * The search group the run searched, to compare and never to parse; null for a Manual search's run
   * @nullable
   */
  groupKey: string | null;
  /**
   * The group in words, such as season 3; null for a Manual search's run, and until its Content Type names its groups
   * @nullable
   */
  groupLabel: string | null;
  id: string;
  /** Each Indexer the run asked or skipped, by priority */
  indexers: GrabsSearchIndexer[];
  /** The Items the run searched */
  itemIds: string[];
  /**
   * The lane it was queued in; null for a Manual search's run
   * @nullable
   */
  lane: GrabsSearchLane;
  /** The grabbed Release's notes, such as size_not_checked, for each Target it was grabbed for */
  notes: GrabsReason[];
  /**
   * Null while the run searches. waiting is a run that waits out a Target's Hold for a better Release. nothing_found is a run that found no Release, and nothing_acceptable one whose Releases fit no Target. no_indexer is a run that no Indexer responded to, and no_root one whose Work had no reachable root. listed is a Manual search's run, which takes nothing.
   * @nullable
   */
  outcome: GrabsSearchOutcome;
  startedAt: string;
  /**
   * What started it; null for a Manual search's run
   * @nullable
   */
  trigger: GrabsSearchTrigger;
}
