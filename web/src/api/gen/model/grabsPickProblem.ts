import type { GrabsPickProblemCode } from "./grabsPickProblemCode.ts";

export interface GrabsPickProblem {
  code: GrabsPickProblemCode;
  detail: string;
  /** cell_held: the Grab downloading that Item and Target */
  grabId?: string;
}
