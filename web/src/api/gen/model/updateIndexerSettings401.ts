import type { Problem } from "./problem.ts";
import type { UpdateIndexerSettings401Code } from "./updateIndexerSettings401Code.ts";

export type UpdateIndexerSettings401 = Problem & {
  code?: UpdateIndexerSettings401Code;
};
