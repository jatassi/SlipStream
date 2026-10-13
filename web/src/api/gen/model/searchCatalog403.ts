import type { Problem } from "./problem.ts";
import type { SearchCatalog403Code } from "./searchCatalog403Code.ts";

export type SearchCatalog403 = Problem & {
  code?: SearchCatalog403Code;
};
