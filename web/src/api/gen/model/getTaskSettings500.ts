import type { GetTaskSettings500Code } from "./getTaskSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetTaskSettings500 = Problem & {
  code?: GetTaskSettings500Code;
};
