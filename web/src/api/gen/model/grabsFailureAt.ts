import type { GrabsFailureAtCode } from "./grabsFailureAtCode.ts";

export interface GrabsFailureAt {
  at: string;
  code: GrabsFailureAtCode;
  /**
   * The failed Grab; null once it is pruned
   * @nullable
   */
  grabId: string | null;
}
