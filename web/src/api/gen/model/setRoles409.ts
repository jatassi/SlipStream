import type { NodesOutcome } from "./nodesOutcome.ts";
import type { Problem } from "./problem.ts";
import type { SetRoles409Code } from "./setRoles409Code.ts";

export type SetRoles409 = Problem & {
  code?: SetRoles409Code;
  outcome?: NodesOutcome;
};
