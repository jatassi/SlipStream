import type { CheckHealthCategory403Code } from "./checkHealthCategory403Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthCategory403 = Problem & {
  code?: CheckHealthCategory403Code;
};
