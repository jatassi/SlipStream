import type { GetHistoryEntry401Code } from "./getHistoryEntry401Code.ts";
import type { Problem } from "./problem.ts";

export type GetHistoryEntry401 = Problem & {
  code?: GetHistoryEntry401Code;
};
