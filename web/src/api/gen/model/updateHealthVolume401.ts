import type { Problem } from "./problem.ts";
import type { UpdateHealthVolume401Code } from "./updateHealthVolume401Code.ts";

export type UpdateHealthVolume401 = Problem & {
  code?: UpdateHealthVolume401Code;
};
