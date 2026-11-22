import type { Problem } from "./problem.ts";
import type { StartMoves500Code } from "./startMoves500Code.ts";

export type StartMoves500 = Problem & {
  code?: StartMoves500Code;
};
