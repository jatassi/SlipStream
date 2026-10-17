import type { NodesOutcome } from "./nodesOutcome.ts";
import type { PlanRoles409Code } from "./planRoles409Code.ts";
import type { Problem } from "./problem.ts";

export type PlanRoles409 = Problem & {
  code?: PlanRoles409Code;
  outcome?: NodesOutcome;
};
