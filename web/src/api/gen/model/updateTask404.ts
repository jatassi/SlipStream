import type { Problem } from "./problem.ts";
import type { UpdateTask404Code } from "./updateTask404Code.ts";

export type UpdateTask404 = Problem & {
  code?: UpdateTask404Code;
};
