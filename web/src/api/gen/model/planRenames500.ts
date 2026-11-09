import type { PlanRenames500Code } from "./planRenames500Code.ts";
import type { Problem } from "./problem.ts";

export type PlanRenames500 = Problem & {
  code?: PlanRenames500Code;
};
