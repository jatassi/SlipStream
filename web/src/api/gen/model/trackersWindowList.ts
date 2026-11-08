import type { TrackersWindow } from "./trackersWindow.ts";

export interface TrackersWindowList {
  /**
   * The offset of the next page; null on the last
   * @nullable
   */
  nextOffset: number | null;
  /** Newest opened first, each as it reads now */
  windows: TrackersWindow[];
}
