import type { Problem } from "./problem.ts";
import type { UpdateHealthSettings422Code } from "./updateHealthSettings422Code.ts";

export type UpdateHealthSettings422 = Problem & {
  code?: UpdateHealthSettings422Code;
};
