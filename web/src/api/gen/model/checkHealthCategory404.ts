import type { CheckHealthCategory404Code } from "./checkHealthCategory404Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthCategory404 = Problem & {
  code?: CheckHealthCategory404Code;
};
