import type { CheckRoot403Code } from "./checkRoot403Code.ts";
import type { Problem } from "./problem.ts";

export type CheckRoot403 = Problem & {
  code?: CheckRoot403Code;
};
