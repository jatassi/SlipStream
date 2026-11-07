import type { GetIndexerCatalog500Code } from "./getIndexerCatalog500Code.ts";
import type { Problem } from "./problem.ts";

export type GetIndexerCatalog500 = Problem & {
  code?: GetIndexerCatalog500Code;
};
