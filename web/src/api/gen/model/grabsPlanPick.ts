import type { GrabsReleaseDetails } from "./grabsReleaseDetails.ts";
import type { GrabsWordedReason } from "./grabsWordedReason.ts";

export interface GrabsPlanPick {
  /**
   * The Edition it is grabbed as; null unless that is another Edition than its cut's
   * @nullable
   */
  edition: string | null;
  /** The soft refusals it is grabbed past; send their codes and Targets back as the pick's lifts */
  lifts: GrabsWordedReason[];
  release: GrabsReleaseDetails;
  /** Where Tracker routing sends it, such as Where Tracker routing sends it (qBittorrent on Den) */
  routed: string;
}
