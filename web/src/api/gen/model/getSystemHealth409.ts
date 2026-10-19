import type { GetSystemHealth409Code } from "./getSystemHealth409Code.ts";
import type { Problem } from "./problem.ts";

export type GetSystemHealth409 = Problem & {
  code?: GetSystemHealth409Code;
};
