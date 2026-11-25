import type { CheckMediaServerAccess409Code } from "./checkMediaServerAccess409Code.ts";
import type { Problem } from "./problem.ts";

export type CheckMediaServerAccess409 = Problem & {
  code?: CheckMediaServerAccess409Code;
};
