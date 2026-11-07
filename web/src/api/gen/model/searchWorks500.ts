import type { Problem } from "./problem.ts";
import type { SearchWorks500Code } from "./searchWorks500Code.ts";

export type SearchWorks500 = Problem & {
  code?: SearchWorks500Code;
};
