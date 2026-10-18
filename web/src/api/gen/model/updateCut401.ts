import type { Problem } from "./problem.ts";
import type { UpdateCut401Code } from "./updateCut401Code.ts";

export type UpdateCut401 = Problem & {
  code?: UpdateCut401Code;
};
