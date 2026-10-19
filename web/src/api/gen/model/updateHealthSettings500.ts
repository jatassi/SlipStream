import type { Problem } from "./problem.ts";
import type { UpdateHealthSettings500Code } from "./updateHealthSettings500Code.ts";

export type UpdateHealthSettings500 = Problem & {
  code?: UpdateHealthSettings500Code;
};
