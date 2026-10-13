import type { Problem } from "./problem.ts";
import type { UpdateNodeSettings401Code } from "./updateNodeSettings401Code.ts";

export type UpdateNodeSettings401 = Problem & {
  code?: UpdateNodeSettings401Code;
};
