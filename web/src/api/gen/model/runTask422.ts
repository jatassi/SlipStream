import type { Problem } from "./problem.ts";
import type { RunTask422Code } from "./runTask422Code.ts";

export type RunTask422 = Problem & {
  code?: RunTask422Code;
};
