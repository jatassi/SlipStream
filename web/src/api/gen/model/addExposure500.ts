import type { AddExposure500Code } from "./addExposure500Code.ts";
import type { Problem } from "./problem.ts";

export type AddExposure500 = Problem & {
  code?: AddExposure500Code;
};
