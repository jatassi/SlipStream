import type { Problem } from "./problem.ts";
import type { RemoveTracker401Code } from "./removeTracker401Code.ts";

export type RemoveTracker401 = Problem & {
  code?: RemoveTracker401Code;
};
