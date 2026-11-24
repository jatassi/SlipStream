import type { Problem } from "./problem.ts";
import type { RemoveExposure401Code } from "./removeExposure401Code.ts";

export type RemoveExposure401 = Problem & {
  code?: RemoveExposure401Code;
};
