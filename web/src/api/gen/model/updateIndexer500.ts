import type { Problem } from "./problem.ts";
import type { UpdateIndexer500Code } from "./updateIndexer500Code.ts";

export type UpdateIndexer500 = Problem & {
  code?: UpdateIndexer500Code;
};
