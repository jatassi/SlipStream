import type { CheckMyMediaServerAccess422Code } from "./checkMyMediaServerAccess422Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMyMediaServerAccess422 = Problem & {
  code?: CheckMyMediaServerAccess422Code;
};
