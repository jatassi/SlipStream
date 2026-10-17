import type { Problem } from "./problem.ts";
import type { UpdateAccount404Code } from "./updateAccount404Code.ts";

export type UpdateAccount404 = Problem & {
  code?: UpdateAccount404Code;
};
