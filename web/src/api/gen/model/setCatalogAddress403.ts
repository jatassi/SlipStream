import type { Problem } from "./problem.ts";
import type { SetCatalogAddress403Code } from "./setCatalogAddress403Code.ts";

export type SetCatalogAddress403 = Problem & {
  code?: SetCatalogAddress403Code;
};
