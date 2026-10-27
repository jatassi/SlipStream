import type { GrabsSearchIndexerOutcome } from "./grabsSearchIndexerOutcome.ts";

export interface GrabsSearchIndexer {
  /**
   * Null once the Indexer is deleted
   * @nullable
   */
  indexerId: string | null;
  /** The Indexer's name when the run started */
  name: string;
  /**
   * Null while its search runs
   * @nullable
   */
  outcome: GrabsSearchIndexerOutcome;
  /**
   * The Releases it found
   * @minimum 0
   */
  releases: number;
  /** The outcome in words, such as Limit reached until 14:05 */
  sentence: string;
}
