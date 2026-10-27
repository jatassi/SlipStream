import type { Problem } from "./problem.ts";
import type { SetCatalogAddress500Code } from "./setCatalogAddress500Code.ts";

export type SetCatalogAddress500 = Problem & {
  code?: SetCatalogAddress500Code;
};
