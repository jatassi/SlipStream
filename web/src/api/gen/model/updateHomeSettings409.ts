import type { Problem } from "./problem.ts";
import type { UpdateHomeSettings409Code } from "./updateHomeSettings409Code.ts";

export type UpdateHomeSettings409 = Problem & {
  code?: UpdateHomeSettings409Code;
};
