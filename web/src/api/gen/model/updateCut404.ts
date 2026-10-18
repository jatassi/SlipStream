import type { Problem } from "./problem.ts";
import type { UpdateCut404Code } from "./updateCut404Code.ts";

export type UpdateCut404 = Problem & {
  code?: UpdateCut404Code;
};
