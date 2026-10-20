import type { PreviewCatalogRecord401Code } from "./previewCatalogRecord401Code.ts";
import type { Problem } from "./problem.ts";

export type PreviewCatalogRecord401 = Problem & {
  code?: PreviewCatalogRecord401Code;
};
