import type { Problem } from "./problem.ts";
import type { UpdateAccount422Code } from "./updateAccount422Code.ts";

export type UpdateAccount422 = Problem & {
  code?: UpdateAccount422Code;
};
