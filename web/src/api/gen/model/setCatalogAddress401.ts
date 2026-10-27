import type { Problem } from "./problem.ts";
import type { SetCatalogAddress401Code } from "./setCatalogAddress401Code.ts";

export type SetCatalogAddress401 = Problem & {
  code?: SetCatalogAddress401Code;
};
