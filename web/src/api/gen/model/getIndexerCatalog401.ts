import type { GetIndexerCatalog401Code } from "./getIndexerCatalog401Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCatalog401 = Problem & {
  code?: GetIndexerCatalog401Code;
};
