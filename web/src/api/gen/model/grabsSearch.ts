import type { GrabsSearchOutcome } from "./grabsSearchOutcome.ts";

export interface GrabsSearch {
  /**
   * The releases the Indexer answered
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
  /**
   * Null while the run searches
   * @nullable
   */
  outcome: GrabsSearchOutcome;
  startedAt: string;
}
