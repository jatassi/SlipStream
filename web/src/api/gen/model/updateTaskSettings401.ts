import type { Problem } from "./problem.ts";
import type { UpdateTaskSettings401Code } from "./updateTaskSettings401Code.ts";

export type UpdateTaskSettings401 = Problem & {
  code?: UpdateTaskSettings401Code;
};
