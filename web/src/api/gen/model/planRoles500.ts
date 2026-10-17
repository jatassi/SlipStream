import type { PlanRoles500Code } from "./planRoles500Code.ts";
import type { Problem } from "./problem.ts";

export type PlanRoles500 = Problem & {
  code?: PlanRoles500Code;
};
