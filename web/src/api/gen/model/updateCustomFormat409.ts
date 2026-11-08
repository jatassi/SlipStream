import type { Problem } from "./problem.ts";
import type { UpdateCustomFormat409Code } from "./updateCustomFormat409Code.ts";

export type UpdateCustomFormat409 = Problem & {
  code?: UpdateCustomFormat409Code;
};
