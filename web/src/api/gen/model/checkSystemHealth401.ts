import type { CheckSystemHealth401Code } from "./checkSystemHealth401Code.ts";
import type { Problem } from "./problem.ts";

export type CheckSystemHealth401 = Problem & {
  code?: CheckSystemHealth401Code;
};
