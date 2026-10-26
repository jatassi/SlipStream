import type { Problem } from "./problem.ts";
import type { SearchWorks501Code } from "./searchWorks501Code.ts";

export type SearchWorks501 = Problem & {
  code?: SearchWorks501Code;
};
