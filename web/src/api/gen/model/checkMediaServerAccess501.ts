import type { CheckMediaServerAccess501Code } from "./checkMediaServerAccess501Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMediaServerAccess501 = Problem & {
  code?: CheckMediaServerAccess501Code;
};
