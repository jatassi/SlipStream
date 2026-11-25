import type { CheckMediaServerAccess422Code } from "./checkMediaServerAccess422Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMediaServerAccess422 = Problem & {
  code?: CheckMediaServerAccess422Code;
};
