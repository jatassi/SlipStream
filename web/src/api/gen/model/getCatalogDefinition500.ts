import type { GetCatalogDefinition500Code } from "./getCatalogDefinition500Code.ts";
import type { Problem } from "./problem.ts";

export type GetCatalogDefinition500 = Problem & {
  code?: GetCatalogDefinition500Code;
};
