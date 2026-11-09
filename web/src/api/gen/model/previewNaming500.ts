import type { PreviewNaming500Code } from "./previewNaming500Code.ts";
import type { Problem } from "./problem.ts";

export type PreviewNaming500 = Problem & {
  code?: PreviewNaming500Code;
};
