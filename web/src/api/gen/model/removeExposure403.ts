import type { Problem } from "./problem.ts";
import type { RemoveExposure403Code } from "./removeExposure403Code.ts";

export type RemoveExposure403 = Problem & {
  code?: RemoveExposure403Code;
};
