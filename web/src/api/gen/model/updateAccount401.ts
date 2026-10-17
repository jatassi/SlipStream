import type { Problem } from "./problem.ts";
import type { UpdateAccount401Code } from "./updateAccount401Code.ts";

export type UpdateAccount401 = Problem & {
  code?: UpdateAccount401Code;
};
