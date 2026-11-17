import type { GetHistoryEntry422Code } from "./getHistoryEntry422Code.ts";
import type { Problem } from "./problem.ts";

export type GetHistoryEntry422 = Problem & {
  code?: GetHistoryEntry422Code;
};
