import type { Problem } from "./problem.ts";
import type { UpdateHomeSettings500Code } from "./updateHomeSettings500Code.ts";

export type UpdateHomeSettings500 = Problem & {
  code?: UpdateHomeSettings500Code;
};
