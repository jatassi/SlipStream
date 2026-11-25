import type { Problem } from "./problem.ts";
import type { RemoveExposure409Code } from "./removeExposure409Code.ts";

export type RemoveExposure409 = Problem & {
  code?: RemoveExposure409Code;
};
