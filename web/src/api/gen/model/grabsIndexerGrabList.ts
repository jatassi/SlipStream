import type { GrabsGrab } from "./grabsGrab.ts";

export interface GrabsIndexerGrabList {
  /** Newest first */
  grabs: GrabsGrab[];
  /**
   * The offset of the next page; null on the last
   * @nullable
   */
  nextOffset: number | null;
}
