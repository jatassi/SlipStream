import type { CheckMediaServerAccess500Code } from "./checkMediaServerAccess500Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMediaServerAccess500 = Problem & {
  code?: CheckMediaServerAccess500Code;
};
