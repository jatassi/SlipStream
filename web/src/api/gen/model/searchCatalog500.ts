import type { Problem } from "./problem.ts";
import type { SearchCatalog500Code } from "./searchCatalog500Code.ts";

export type SearchCatalog500 = Problem & {
  code?: SearchCatalog500Code;
};
