import type { GetTaskSettings401Code } from "./getTaskSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetTaskSettings401 = Problem & {
  code?: GetTaskSettings401Code;
};
