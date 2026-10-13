import type { Problem } from "./problem.ts";
import type { SearchCatalog404Code } from "./searchCatalog404Code.ts";

export type SearchCatalog404 = Problem & {
  code?: SearchCatalog404Code;
};
