import type { Problem } from "./problem.ts";
import type { RunTask409Code } from "./runTask409Code.ts";

export type RunTask409 = Problem & {
  code?: RunTask409Code;
};
