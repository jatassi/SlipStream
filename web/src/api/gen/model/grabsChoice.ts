import type { GrabsReleaseDetails } from "./grabsReleaseDetails.ts";
import type { GrabsWordedReason } from "./grabsWordedReason.ts";

export interface GrabsChoice {
  /** The cells its row fills once it is pinned, the row's first leading */
  cells: string[];
  /**
   * The Edition it is grabbed as; null unless that is another Edition than its cut's
   * @nullable
   */
  edition: string | null;
  lifts: GrabsWordedReason[];
  /** Such as From TorrentLeech, which automatic search doesn't use. */
  note?: string;
  /** Send it in pins to swap this Release into its row */
  pin: string;
  release: GrabsReleaseDetails;
}
