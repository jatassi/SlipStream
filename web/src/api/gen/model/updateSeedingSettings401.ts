import type { Problem } from "./problem.ts";
import type { UpdateSeedingSettings401Code } from "./updateSeedingSettings401Code.ts";

export type UpdateSeedingSettings401 = Problem & {
  code?: UpdateSeedingSettings401Code;
};
