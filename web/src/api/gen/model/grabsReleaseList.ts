import type { GrabsListedRelease } from "./grabsListedRelease.ts";

export interface GrabsReleaseList {
  /** Ranked ones first, in rank order, then the rest in the order they were listed. Empty once a later run of the Item finished. */
  releases: GrabsListedRelease[];
}
