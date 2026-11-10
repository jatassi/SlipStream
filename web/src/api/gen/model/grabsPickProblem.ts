import type { GrabsPickProblemCode } from "./grabsPickProblemCode.ts";

export interface GrabsPickProblem {
  code: GrabsPickProblemCode;
  detail: string;
  /** cell_held: the Grab downloading the cell */
  grabId?: string;
}
