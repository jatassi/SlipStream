import type { PreviewNaming404Code } from "./previewNaming404Code.ts";
import type { Problem } from "./problem.ts";

export type PreviewNaming404 = Problem & {
  code?: PreviewNaming404Code;
};
