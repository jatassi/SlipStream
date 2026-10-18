import type { Problem } from "./problem.ts";
import type { UpdateTarget404Code } from "./updateTarget404Code.ts";

export type UpdateTarget404 = Problem & {
  code?: UpdateTarget404Code;
};
