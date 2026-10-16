import type { Problem } from "./problem.ts";
import type { RunTask404Code } from "./runTask404Code.ts";

export type RunTask404 = Problem & {
  code?: RunTask404Code;
};
