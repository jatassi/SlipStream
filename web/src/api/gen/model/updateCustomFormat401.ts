import type { Problem } from "./problem.ts";
import type { UpdateCustomFormat401Code } from "./updateCustomFormat401Code.ts";

export type UpdateCustomFormat401 = Problem & {
  code?: UpdateCustomFormat401Code;
};
