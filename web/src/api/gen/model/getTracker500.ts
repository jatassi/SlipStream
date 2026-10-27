import type { GetTracker500Code } from "./getTracker500Code.ts";
import type { Problem } from "./problem.ts";

export type GetTracker500 = Problem & {
  code?: GetTracker500Code;
};
