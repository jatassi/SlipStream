import type { Problem } from "./problem.ts";
import type { UpdateSeedingSettings409Code } from "./updateSeedingSettings409Code.ts";

export type UpdateSeedingSettings409 = Problem & {
  code?: UpdateSeedingSettings409Code;
};
