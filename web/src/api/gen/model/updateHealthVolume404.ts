import type { Problem } from "./problem.ts";
import type { UpdateHealthVolume404Code } from "./updateHealthVolume404Code.ts";

export type UpdateHealthVolume404 = Problem & {
  code?: UpdateHealthVolume404Code;
};
