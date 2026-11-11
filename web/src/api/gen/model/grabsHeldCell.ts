import type { GrabsHeldCellStage } from "./grabsHeldCellStage.ts";

export interface GrabsHeldCell {
  grabId: string;
  stage: GrabsHeldCellStage;
  /** Such as A Grab is downloading WEBDL-1080p · FLUX · 6.2 GB for 1080p. */
  words: string;
}
