import type { Problem } from "./problem.ts";
import type { SearchCatalog409Code } from "./searchCatalog409Code.ts";

export type SearchCatalog409 = Problem & {
  code?: SearchCatalog409Code;
};
