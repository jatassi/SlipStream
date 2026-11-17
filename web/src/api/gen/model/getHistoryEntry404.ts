import type { GetHistoryEntry404Code } from "./getHistoryEntry404Code.ts";
import type { Problem } from "./problem.ts";

export type GetHistoryEntry404 = Problem & {
  code?: GetHistoryEntry404Code;
};
