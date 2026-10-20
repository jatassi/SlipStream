import type { PreviewCatalogRecord404Code } from "./previewCatalogRecord404Code.ts";
import type { Problem } from "./problem.ts";

export type PreviewCatalogRecord404 = Problem & {
  code?: PreviewCatalogRecord404Code;
};
