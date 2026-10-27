import type { Problem } from "./problem.ts";
import type { SearchWorks422Code } from "./searchWorks422Code.ts";

export type SearchWorks422 = Problem & {
  code?: SearchWorks422Code;
};
