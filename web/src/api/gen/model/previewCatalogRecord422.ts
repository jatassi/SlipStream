import type { PreviewCatalogRecord422Code } from "./previewCatalogRecord422Code.ts";
import type { Problem } from "./problem.ts";

export type PreviewCatalogRecord422 = Problem & {
  code?: PreviewCatalogRecord422Code;
};
