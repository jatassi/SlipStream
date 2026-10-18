import type { GrabsReason } from "./grabsReason.ts";

export interface GrabsRefusal {
  /** Each open cell's Target's failures and notes, in Profile order */
  reasons: GrabsReason[];
  title: string;
}
