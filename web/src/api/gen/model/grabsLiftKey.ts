import type { GrabsLiftKeyCode } from "./grabsLiftKeyCode.ts";

export interface GrabsLiftKey {
  code: GrabsLiftKeyCode;
  /** Absent for a release's own refusals: wrong_title, not_aired and tried */
  targetId?: string;
}
