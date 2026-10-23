import type { Problem } from "./problem.ts";
import type { UpdateHealthSettings409Code } from "./updateHealthSettings409Code.ts";

export type UpdateHealthSettings409 = Problem & {
  code?: UpdateHealthSettings409Code;
};
