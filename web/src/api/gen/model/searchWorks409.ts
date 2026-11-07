import type { Problem } from "./problem.ts";
import type { SearchWorks409Code } from "./searchWorks409Code.ts";

export type SearchWorks409 = Problem & {
  code?: SearchWorks409Code;
};
