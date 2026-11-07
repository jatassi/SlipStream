import type { Problem } from "./problem.ts";
import type { UpdateIndexerSettings500Code } from "./updateIndexerSettings500Code.ts";

export type UpdateIndexerSettings500 = Problem & {
  code?: UpdateIndexerSettings500Code;
};
