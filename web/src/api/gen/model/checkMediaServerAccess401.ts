import type { CheckMediaServerAccess401Code } from "./checkMediaServerAccess401Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMediaServerAccess401 = Problem & {
  code?: CheckMediaServerAccess401Code;
};
