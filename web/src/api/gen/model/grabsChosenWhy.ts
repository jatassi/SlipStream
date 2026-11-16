import type { GrabsChosenWhyKind } from "./grabsChosenWhyKind.ts";

export interface GrabsChosenWhy {
  /**
   * How many of the group's Items it was chosen for
   * @minimum 1
   */
  fills: number;
  /** fewer_releases is a Release with more of the group's Items, so the group needs fewer Releases */
  kind: GrabsChosenWhyKind;
}
