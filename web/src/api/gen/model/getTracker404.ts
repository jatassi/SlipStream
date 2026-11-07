import type { GetTracker404Code } from "./getTracker404Code.ts";
import type { Problem } from "./problem.ts";

export type GetTracker404 = Problem & {
  code?: GetTracker404Code;
};
