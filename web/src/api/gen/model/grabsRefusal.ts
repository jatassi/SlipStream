import type { GrabsReason } from "./grabsReason.ts";

export interface GrabsRefusal {
  /** The release's own refusals, then each cell's failures and notes, in Profile order */
  reasons: GrabsReason[];
  title: string;
}
