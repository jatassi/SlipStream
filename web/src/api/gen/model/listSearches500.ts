import type { ListSearches500Code } from "./listSearches500Code.ts";
import type { Problem } from "./problem.ts";

export type ListSearches500 = Problem & {
  code?: ListSearches500Code;
};
