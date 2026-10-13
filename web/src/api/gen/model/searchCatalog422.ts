import type { Problem } from "./problem.ts";
import type { SearchCatalog422Code } from "./searchCatalog422Code.ts";

export type SearchCatalog422 = Problem & {
  code?: SearchCatalog422Code;
};
