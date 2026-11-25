import type { CheckMediaServerAccess502Code } from "./checkMediaServerAccess502Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMediaServerAccess502 = Problem & {
  code?: CheckMediaServerAccess502Code;
};
