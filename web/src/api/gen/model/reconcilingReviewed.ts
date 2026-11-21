import type { ReconcilingReviewedFound } from "./reconcilingReviewedFound.ts";
import type { ReconcilingReviewedReason } from "./reconcilingReviewedReason.ts";

export interface ReconcilingReviewed {
  found: ReconcilingReviewedFound;
  id: string;
  name: string;
  /** The Node it was recorded on */
  nodeId: string;
  /**
   * The Node Re-add puts it on once Reconciling finishes; null until Re-add is pressed
   * @nullable
   */
  readd: string | null;
  reason: ReconcilingReviewedReason;
  /** When it joined the review list */
  since: string;
}
