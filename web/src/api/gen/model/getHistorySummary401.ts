import type { GetHistorySummary401Code } from "./getHistorySummary401Code.ts";
import type { Problem } from "./problem.ts";

export type GetHistorySummary401 = Problem & {
  code?: GetHistorySummary401Code;
};
