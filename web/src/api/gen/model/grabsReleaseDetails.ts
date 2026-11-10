import type { GrabsLostAt } from "./grabsLostAt.ts";

export interface GrabsReleaseDetails {
  /**
   * Since when and why it is on the Blocklist; null when it isn't
   * @nullable
   */
  blocked: string | null;
  /** The Custom Formats it matched of those the Work's Targets score */
  formatIds: string[];
  freeleech: boolean;
  /** Send it back as a pick's releaseId */
  id: string;
  /** The Indexers that listed it, by priority */
  indexers: string[];
  /** Where it lost to the Release that fills its first cell; null when it didn't */
  lostAt: GrabsLostAt | null;
  /**
   * Null when the Indexer sent no date
   * @nullable
   */
  publishedAt: string | null;
  /** @minimum 0 */
  seeders: number;
  /**
   * Bytes
   * @minimum 0
   */
  size: number;
  title: string;
  /** The Tracker its link reaches, absent when none is known */
  tracker?: string;
  /** Its verdict at each Target of its Item, such as It would fill 1080p. */
  verdicts: string[];
  /** What SlipStream reads from the title, in words */
  words: string;
}
