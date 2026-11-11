import type { GrabsPickProblem } from "./grabsPickProblem.ts";

export interface GrabsPickResult {
  /**
   * Null when the pick was refused
   * @nullable
   */
  grabId: string | null;
  /** Null when the pick made a Grab */
  problem: GrabsPickProblem | null;
}
