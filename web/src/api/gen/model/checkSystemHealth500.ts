import type { CheckSystemHealth500Code } from "./checkSystemHealth500Code.ts";
import type { Problem } from "./problem.ts";

export type CheckSystemHealth500 = Problem & {
  code?: CheckSystemHealth500Code;
};
