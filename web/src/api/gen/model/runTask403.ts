import type { Problem } from "./problem.ts";
import type { RunTask403Code } from "./runTask403Code.ts";

export type RunTask403 = Problem & {
  code?: RunTask403Code;
};
