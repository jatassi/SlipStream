import type { CheckMyMediaServerAccess404Code } from "./checkMyMediaServerAccess404Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMyMediaServerAccess404 = Problem & {
  code?: CheckMyMediaServerAccess404Code;
};
