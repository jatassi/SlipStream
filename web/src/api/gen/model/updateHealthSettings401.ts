import type { Problem } from "./problem.ts";
import type { UpdateHealthSettings401Code } from "./updateHealthSettings401Code.ts";

export type UpdateHealthSettings401 = Problem & {
  code?: UpdateHealthSettings401Code;
};
