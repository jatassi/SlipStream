import type { ListSearches404Code } from "./listSearches404Code.ts";
import type { Problem } from "./problem.ts";

export type ListSearches404 = Problem & {
  code?: ListSearches404Code;
};
