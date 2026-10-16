import type { Problem } from "./problem.ts";
import type { UpdateTask422Code } from "./updateTask422Code.ts";

export type UpdateTask422 = Problem & {
  code?: UpdateTask422Code;
};
