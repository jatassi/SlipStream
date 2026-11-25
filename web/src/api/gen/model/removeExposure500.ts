import type { Problem } from "./problem.ts";
import type { RemoveExposure500Code } from "./removeExposure500Code.ts";

export type RemoveExposure500 = Problem & {
  code?: RemoveExposure500Code;
};
