import type { GrabsOffer } from "./grabsOffer.ts";

export interface GrabsOfferList {
  /** One per group of copies, in rank order */
  releases: GrabsOffer[];
}
