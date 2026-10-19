import type { Problem } from "./problem.ts";
import type { UpdateRootSettings401Code } from "./updateRootSettings401Code.ts";

export type UpdateRootSettings401 = Problem & {
  code?: UpdateRootSettings401Code;
};
