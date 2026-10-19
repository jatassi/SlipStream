import type { GetSystemHealth401Code } from "./getSystemHealth401Code.ts";
import type { Problem } from "./problem.ts";

export type GetSystemHealth401 = Problem & {
  code?: GetSystemHealth401Code;
};
