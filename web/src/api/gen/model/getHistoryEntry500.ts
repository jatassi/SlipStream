import type { GetHistoryEntry500Code } from "./getHistoryEntry500Code.ts";
import type { Problem } from "./problem.ts";

export type GetHistoryEntry500 = Problem & {
  code?: GetHistoryEntry500Code;
};
