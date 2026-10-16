import type { Problem } from "./problem.ts";
import type { UpdateTaskSettings500Code } from "./updateTaskSettings500Code.ts";

export type UpdateTaskSettings500 = Problem & {
  code?: UpdateTaskSettings500Code;
};
