import type { Problem } from "./problem.ts";
import type { UpdateTaskSettings409Code } from "./updateTaskSettings409Code.ts";

export type UpdateTaskSettings409 = Problem & {
  code?: UpdateTaskSettings409Code;
};
