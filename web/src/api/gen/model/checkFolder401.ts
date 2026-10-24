import type { CheckFolder401Code } from "./checkFolder401Code.ts";
import type { Problem } from "./problem.ts";

export type CheckFolder401 = Problem & {
  code?: CheckFolder401Code;
};
