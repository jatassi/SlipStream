import type { GetHistorySummary500Code } from "./getHistorySummary500Code.ts";
import type { Problem } from "./problem.ts";

export type GetHistorySummary500 = Problem & {
  code?: GetHistorySummary500Code;
};
