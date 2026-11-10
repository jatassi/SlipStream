import type { GrabsGrabOffer } from "./grabsGrabOffer.ts";
import type { GrabsReleaseDetails } from "./grabsReleaseDetails.ts";

export interface GrabsOffer {
  grab: GrabsGrabOffer;
  /**
   * Its quality's place in its best Target's tiers, for sorting; higher is better
   * @minimum 0
   */
  quality: number;
  /**
   * Its place in the ranking from 1; null when no cell's Target takes it
   * @minimum 1
   * @nullable
   */
  rank: number | null;
  release: GrabsReleaseDetails;
  /** Its score at its best Target */
  score: number;
}
