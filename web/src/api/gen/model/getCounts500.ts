import type { GetCounts500Code } from "./getCounts500Code.ts";
import type { Problem } from "./problem.ts";

export type GetCounts500 = Problem & {
  code?: GetCounts500Code;
};
