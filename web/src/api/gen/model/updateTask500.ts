import type { Problem } from "./problem.ts";
import type { UpdateTask500Code } from "./updateTask500Code.ts";

export type UpdateTask500 = Problem & {
  code?: UpdateTask500Code;
};
