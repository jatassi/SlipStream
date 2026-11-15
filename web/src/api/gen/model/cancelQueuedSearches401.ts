import type { CancelQueuedSearches401Code } from "./cancelQueuedSearches401Code.ts";
import type { Problem } from "./problem.ts";

export type CancelQueuedSearches401 = Problem & {
  code?: CancelQueuedSearches401Code;
};
