import type { Problem } from "./problem.ts";
import type { UpdateHealthVolume409Code } from "./updateHealthVolume409Code.ts";

export type UpdateHealthVolume409 = Problem & {
  code?: UpdateHealthVolume409Code;
};
