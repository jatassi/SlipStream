import type { Problem } from "./problem.ts";
import type { UpdateCustomFormat422Code } from "./updateCustomFormat422Code.ts";

export type UpdateCustomFormat422 = Problem & {
  code?: UpdateCustomFormat422Code;
};
