import type { Problem } from "./problem.ts";
import type { UpdateIndexer404Code } from "./updateIndexer404Code.ts";

export type UpdateIndexer404 = Problem & {
  code?: UpdateIndexer404Code;
};
