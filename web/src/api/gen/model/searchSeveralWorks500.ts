import type { Problem } from "./problem.ts";
import type { SearchSeveralWorks500Code } from "./searchSeveralWorks500Code.ts";

export type SearchSeveralWorks500 = Problem & {
  code?: SearchSeveralWorks500Code;
};
