import type { CheckMyMediaServerAccess401Code } from "./checkMyMediaServerAccess401Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMyMediaServerAccess401 = Problem & {
  code?: CheckMyMediaServerAccess401Code;
};
