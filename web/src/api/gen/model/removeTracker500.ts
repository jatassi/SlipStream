import type { Problem } from "./problem.ts";
import type { RemoveTracker500Code } from "./removeTracker500Code.ts";

export type RemoveTracker500 = Problem & {
  code?: RemoveTracker500Code;
};
