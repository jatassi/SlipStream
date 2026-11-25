import type { CheckMediaServerAccess404Code } from "./checkMediaServerAccess404Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMediaServerAccess404 = Problem & {
  code?: CheckMediaServerAccess404Code;
};
