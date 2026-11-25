import type { AddExposure422Code } from "./addExposure422Code.ts";
import type { Problem } from "./problem.ts";

export type AddExposure422 = Problem & {
  code?: AddExposure422Code;
};
