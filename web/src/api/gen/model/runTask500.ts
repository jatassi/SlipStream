import type { Problem } from "./problem.ts";
import type { RunTask500Code } from "./runTask500Code.ts";

export type RunTask500 = Problem & {
  code?: RunTask500Code;
};
