import type { Problem } from "./problem.ts";
import type { SearchCatalog401Code } from "./searchCatalog401Code.ts";

export type SearchCatalog401 = Problem & {
  code?: SearchCatalog401Code;
};
