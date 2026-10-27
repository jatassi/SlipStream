import type { Problem } from "./problem.ts";
import type { SearchWorks404Code } from "./searchWorks404Code.ts";

export type SearchWorks404 = Problem & {
  code?: SearchWorks404Code;
};
