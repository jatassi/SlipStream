import type { PreviewCatalogRecord500Code } from "./previewCatalogRecord500Code.ts";
import type { Problem } from "./problem.ts";

export type PreviewCatalogRecord500 = Problem & {
  code?: PreviewCatalogRecord500Code;
};
