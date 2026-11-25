import type { AddExposure401Code } from "./addExposure401Code.ts";
import type { Problem } from "./problem.ts";

export type AddExposure401 = Problem & {
  code?: AddExposure401Code;
};
