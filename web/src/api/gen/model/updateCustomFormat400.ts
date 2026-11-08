import type { Problem } from "./problem.ts";
import type { UpdateCustomFormat400Code } from "./updateCustomFormat400Code.ts";

export type UpdateCustomFormat400 = Problem & {
  code?: UpdateCustomFormat400Code;
};
