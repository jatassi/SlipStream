import type { GetTracker401Code } from "./getTracker401Code.ts";
import type { Problem } from "./problem.ts";

export type GetTracker401 = Problem & {
  code?: GetTracker401Code;
};
