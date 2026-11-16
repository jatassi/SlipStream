import type { CancelQueuedSearches500Code } from "./cancelQueuedSearches500Code.ts";
import type { Problem } from "./problem.ts";

export type CancelQueuedSearches500 = Problem & {
  code?: CancelQueuedSearches500Code;
};
