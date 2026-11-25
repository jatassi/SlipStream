import type { AddExposure403Code } from "./addExposure403Code.ts";
import type { Problem } from "./problem.ts";

export type AddExposure403 = Problem & {
  code?: AddExposure403Code;
};
