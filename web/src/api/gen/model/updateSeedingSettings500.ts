import type { Problem } from "./problem.ts";
import type { UpdateSeedingSettings500Code } from "./updateSeedingSettings500Code.ts";

export type UpdateSeedingSettings500 = Problem & {
  code?: UpdateSeedingSettings500Code;
};
