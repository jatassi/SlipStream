import type { IndexersRssFeedGapKind } from "./indexersRssFeedGapKind.ts";

export interface IndexersRssFeed {
  /**
   * Why the last fetches didn't reach where the one before stopped; null when they did
   * @nullable
   */
  gapKind: IndexersRssFeedGapKind;
  /**
   * Set with gapKind: the first fetch that didn't reach it
   * @nullable
   */
  gapSince: string | null;
  lastFetchAt: string;
}
