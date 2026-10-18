import type { Problem } from "./problem.ts";
import type { UpdateCut500Code } from "./updateCut500Code.ts";

export type UpdateCut500 = Problem & {
  code?: UpdateCut500Code;
};
