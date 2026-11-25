import type { CheckMyMediaServerAccess500Code } from "./checkMyMediaServerAccess500Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMyMediaServerAccess500 = Problem & {
  code?: CheckMyMediaServerAccess500Code;
};
