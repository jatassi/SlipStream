import type { Problem } from "./problem.ts";
import type { RunTask401Code } from "./runTask401Code.ts";

export type RunTask401 = Problem & {
  code?: RunTask401Code;
};
