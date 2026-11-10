import type { GrabsLiftKeyCode } from "./grabsLiftKeyCode.ts";

export interface GrabsLiftKey {
  code: GrabsLiftKeyCode;
  /** Absent for wrong_title */
  targetId?: string;
}
