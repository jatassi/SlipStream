import type { CheckFolder403Code } from "./checkFolder403Code.ts";
import type { Problem } from "./problem.ts";

export type CheckFolder403 = Problem & {
  code?: CheckFolder403Code;
};
