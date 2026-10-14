import type { ListSearches401Code } from "./listSearches401Code.ts";
import type { Problem } from "./problem.ts";

export type ListSearches401 = Problem & {
  code?: ListSearches401Code;
};
