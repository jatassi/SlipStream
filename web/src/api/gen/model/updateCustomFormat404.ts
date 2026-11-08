import type { Problem } from "./problem.ts";
import type { UpdateCustomFormat404Code } from "./updateCustomFormat404Code.ts";

export type UpdateCustomFormat404 = Problem & {
  code?: UpdateCustomFormat404Code;
};
