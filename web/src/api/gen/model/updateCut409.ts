import type { Problem } from "./problem.ts";
import type { UpdateCut409Code } from "./updateCut409Code.ts";

export type UpdateCut409 = Problem & {
  code?: UpdateCut409Code;
};
