import type { CancelQueuedSearches422Code } from "./cancelQueuedSearches422Code.ts";
import type { Problem } from "./problem.ts";

export type CancelQueuedSearches422 = Problem & {
  code?: CancelQueuedSearches422Code;
};
