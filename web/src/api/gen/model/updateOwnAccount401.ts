import type { Problem } from "./problem.ts";
import type { UpdateOwnAccount401Code } from "./updateOwnAccount401Code.ts";

export type UpdateOwnAccount401 = Problem & {
  code?: UpdateOwnAccount401Code;
};
