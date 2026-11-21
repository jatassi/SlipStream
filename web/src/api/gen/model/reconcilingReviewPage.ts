import type { ReconcilingReviewed } from "./reconcilingReviewed.ts";

export interface ReconcilingReviewPage {
  /**
   * The after of the next page; null on the last page. Rows added or removed meanwhile never move it
   * @nullable
   */
  next: string | null;
  torrents: ReconcilingReviewed[];
}
