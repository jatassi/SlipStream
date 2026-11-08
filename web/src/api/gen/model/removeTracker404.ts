import type { Problem } from "./problem.ts";
import type { RemoveTracker404Code } from "./removeTracker404Code.ts";

export type RemoveTracker404 = Problem & {
  code?: RemoveTracker404Code;
};
