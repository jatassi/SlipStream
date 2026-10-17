import type { PlanRoles401Code } from "./planRoles401Code.ts";
import type { Problem } from "./problem.ts";

export type PlanRoles401 = Problem & {
  code?: PlanRoles401Code;
};
