import type { GetHistoryEntry409Code } from "./getHistoryEntry409Code.ts";
import type { Problem } from "./problem.ts";

export type GetHistoryEntry409 = Problem & {
  code?: GetHistoryEntry409Code;
};
