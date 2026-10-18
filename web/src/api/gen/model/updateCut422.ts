import type { Problem } from "./problem.ts";
import type { UpdateCut422Code } from "./updateCut422Code.ts";

export type UpdateCut422 = Problem & {
  code?: UpdateCut422Code;
};
