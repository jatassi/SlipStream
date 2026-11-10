import type { GrabsHeldCellStage } from "./grabsHeldCellStage.ts";

export interface GrabsHeldCell {
  grabId: string;
  stage: GrabsHeldCellStage;
  /** Such as A Grab is downloading Heat.1995.1080p.WEB for 1080p. */
  words: string;
}
