import type { NodesOutcome } from "./nodesOutcome.ts";
import type { Problem } from "./problem.ts";
import type { WriteOff409Code } from "./writeOff409Code.ts";

export type WriteOff409 = Problem & {
  code?: WriteOff409Code;
  outcome?: NodesOutcome;
};
