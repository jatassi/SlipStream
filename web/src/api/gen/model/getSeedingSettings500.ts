import type { GetSeedingSettings500Code } from "./getSeedingSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetSeedingSettings500 = Problem & {
  code?: GetSeedingSettings500Code;
};
