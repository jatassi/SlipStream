import type { Problem } from "./problem.ts";
import type { UpdateHealthVolume500Code } from "./updateHealthVolume500Code.ts";

export type UpdateHealthVolume500 = Problem & {
  code?: UpdateHealthVolume500Code;
};
