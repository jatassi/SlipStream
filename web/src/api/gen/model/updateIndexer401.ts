import type { Problem } from "./problem.ts";
import type { UpdateIndexer401Code } from "./updateIndexer401Code.ts";

export type UpdateIndexer401 = Problem & {
  code?: UpdateIndexer401Code;
};
