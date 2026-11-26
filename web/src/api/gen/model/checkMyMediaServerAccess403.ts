import type { CheckMyMediaServerAccess403Code } from "./checkMyMediaServerAccess403Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMyMediaServerAccess403 = Problem & {
  code?: CheckMyMediaServerAccess403Code;
};
