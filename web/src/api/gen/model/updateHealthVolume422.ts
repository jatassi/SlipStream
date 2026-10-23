import type { Problem } from "./problem.ts";
import type { UpdateHealthVolume422Code } from "./updateHealthVolume422Code.ts";

export type UpdateHealthVolume422 = Problem & {
  code?: UpdateHealthVolume422Code;
};
