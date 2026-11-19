import type { GrabsListedReleaseOutcome } from "./grabsListedReleaseOutcome.ts";
import type { GrabsLostAt } from "./grabsLostAt.ts";
import type { GrabsReleaseFacts } from "./grabsReleaseFacts.ts";
import type { GrabsTargetScore } from "./grabsTargetScore.ts";
import type { GrabsWordedReason } from "./grabsWordedReason.ts";

export interface GrabsListedRelease {
  /**
   * The position of the Release this is a copy of, listed by an Indexer of no better priority; null when it is no copy
   * @minimum 0
   * @nullable
   */
  copyOf: number | null;
  /** The Items of the Work it holds */
  covers: string[];
  facts: GrabsReleaseFacts;
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
  /** grabbed is the run's Grab. not_taken ranked above it, but its Indexer had stopped, reached its limit or no longer listed it. outranked ranked below it. kept ranked, but the run grabbed nothing. refused has a refusal of its own, or no Target accepts it. waiting is a candidate SlipStream keeps during a Target's Hold, until it grabs the best one. */
  outcome: GrabsListedReleaseOutcome;
  /**
   * Its place in the order the Indexers listed the run's Releases
   * @minimum 0
   */
  position: number;
  /**
   * Its place among the releases some Target accepts, from 1; null for one no Target accepts
   * @minimum 1
   * @nullable
   */
  rank: number | null;
  /** Its own refusals, then the failures and notes at each Target, in Profile order */
  reasons: GrabsWordedReason[];
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
