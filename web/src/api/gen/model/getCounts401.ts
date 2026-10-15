import type { GetCounts401Code } from "./getCounts401Code.ts";
import type { Problem } from "./problem.ts";

export type GetCounts401 = Problem & {
  code?: GetCounts401Code;
};
