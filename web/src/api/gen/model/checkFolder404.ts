import type { CheckFolder404Code } from "./checkFolder404Code.ts";
import type { Problem } from "./problem.ts";

export type CheckFolder404 = Problem & {
  code?: CheckFolder404Code;
};
