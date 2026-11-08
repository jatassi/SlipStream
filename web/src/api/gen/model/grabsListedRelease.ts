import type { GrabsListedReleaseOutcome } from "./grabsListedReleaseOutcome.ts";
import type { GrabsLostAt } from "./grabsLostAt.ts";
import type { GrabsReason } from "./grabsReason.ts";
import type { GrabsTargetScore } from "./grabsTargetScore.ts";

export interface GrabsListedRelease {
  /**
   * The position of the Release this is a copy of, listed by an Indexer of no better priority; null when it is no copy
   * @minimum 0
   * @nullable
   */
  copyOf: number | null;
  /** The Items of the Work it holds */
  covers: string[];
  /** The Custom Formats it matched of those the Work's Targets score, as the run saw them */
  formatIds: string[];
  /**
   * Null once the Indexer is deleted
   * @nullable
   */
  indexerId: string | null;
  /** The Indexer's name when the run started */
  indexerName: string;
  /** Where an outranked or kept Release lost to the reference Release; null for the others and for one that ties it at every step */
  lostAt: GrabsLostAt | null;
  /** grabbed is the run's Grab. not_taken ranked above it, and its Indexer stopped, reached its limit or dropped it. outranked ranked below it. kept ranked, and the run grabbed nothing. refused has a refusal of its own or is grabbable at no cell. */
  outcome: GrabsListedReleaseOutcome;
  /**
   * Its place in the order the Indexers listed the run's Releases
   * @minimum 0
   */
  position: number;
  /**
   * Its place among the Releases grabbable at some cell, from 1; null for one grabbable at none
   * @minimum 1
   * @nullable
   */
  rank: number | null;
  /** Its own refusals, then each cell's failures and notes, in Profile order */
  reasons: GrabsReason[];
  /** Its score at each Target it was judged at, in Profile order */
  scores: GrabsTargetScore[];
  /** @minimum 0 */
  seeders: number;
  /**
   * Bytes
   * @minimum 0
   */
  size: number;
  title: string;
  /** What SlipStream reads from the title, in words, such as WEBDL-1080p · Dolby Vision · DD+ 5.1 · FLUX · 6.2 GB */
  words: string;
}
