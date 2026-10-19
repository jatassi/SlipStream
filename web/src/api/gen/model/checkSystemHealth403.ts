import type { CheckSystemHealth403Code } from "./checkSystemHealth403Code.ts";
import type { Problem } from "./problem.ts";

export type CheckSystemHealth403 = Problem & {
  code?: CheckSystemHealth403Code;
};
