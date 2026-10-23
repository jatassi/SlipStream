import type { Problem } from "./problem.ts";
import type { UpdateHealthSettings403Code } from "./updateHealthSettings403Code.ts";

export type UpdateHealthSettings403 = Problem & {
  code?: UpdateHealthSettings403Code;
};
