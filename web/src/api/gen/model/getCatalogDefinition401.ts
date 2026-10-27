import type { GetCatalogDefinition401Code } from "./getCatalogDefinition401Code.ts";
import type { Problem } from "./problem.ts";

export type GetCatalogDefinition401 = Problem & {
  code?: GetCatalogDefinition401Code;
};
