import type { Problem } from "./problem.ts";
import type { UpdateAccount500Code } from "./updateAccount500Code.ts";

export type UpdateAccount500 = Problem & {
  code?: UpdateAccount500Code;
};
