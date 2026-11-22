import type { Problem } from "./problem.ts";
import type { StartMoves401Code } from "./startMoves401Code.ts";

export type StartMoves401 = Problem & {
  code?: StartMoves401Code;
};
