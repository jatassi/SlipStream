import type { PlanRenames404Code } from "./planRenames404Code.ts";
import type { Problem } from "./problem.ts";

export type PlanRenames404 = Problem & {
  code?: PlanRenames404Code;
};
