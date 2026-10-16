import type { ListOpenTaskRuns500Code } from "./listOpenTaskRuns500Code.ts";
import type { Problem } from "./problem.ts";

export type ListOpenTaskRuns500 = Problem & {
  code?: ListOpenTaskRuns500Code;
};
