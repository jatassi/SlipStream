import type { CheckMediaServerAccess403Code } from "./checkMediaServerAccess403Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMediaServerAccess403 = Problem & {
  code?: CheckMediaServerAccess403Code;
};
