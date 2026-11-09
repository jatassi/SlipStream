import type { PlanRenames401Code } from "./planRenames401Code.ts";
import type { Problem } from "./problem.ts";

export type PlanRenames401 = Problem & {
  code?: PlanRenames401Code;
};
