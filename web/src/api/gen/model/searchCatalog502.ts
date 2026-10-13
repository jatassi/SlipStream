import type { Problem } from "./problem.ts";
import type { SearchCatalog502Code } from "./searchCatalog502Code.ts";

export type SearchCatalog502 = Problem & {
  code?: SearchCatalog502Code;
};
