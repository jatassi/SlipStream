import type { Problem } from "./problem.ts";
import type { UpdateHomeSettings401Code } from "./updateHomeSettings401Code.ts";

export type UpdateHomeSettings401 = Problem & {
  code?: UpdateHomeSettings401Code;
};
