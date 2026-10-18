import type { Problem } from "./problem.ts";
import type { UpdateTarget401Code } from "./updateTarget401Code.ts";

export type UpdateTarget401 = Problem & {
  code?: UpdateTarget401Code;
};
