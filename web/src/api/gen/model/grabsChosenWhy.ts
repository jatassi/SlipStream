import type { GrabsChosenWhyKind } from "./grabsChosenWhyKind.ts";

export interface GrabsChosenWhy {
  /**
   * The Items it was chosen to fill
   * @minimum 1
   */
  fills: number;
  /** fewer_releases is a Release that fills more Items, so fewer Releases fill the group */
  kind: GrabsChosenWhyKind;
}
