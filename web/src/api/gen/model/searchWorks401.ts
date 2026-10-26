import type { Problem } from "./problem.ts";
import type { SearchWorks401Code } from "./searchWorks401Code.ts";

export type SearchWorks401 = Problem & {
  code?: SearchWorks401Code;
};
