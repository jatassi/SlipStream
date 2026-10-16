import type { Problem } from "./problem.ts";
import type { UpdateTask401Code } from "./updateTask401Code.ts";

export type UpdateTask401 = Problem & {
  code?: UpdateTask401Code;
};
