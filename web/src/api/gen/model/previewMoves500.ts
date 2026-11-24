import type { PreviewMoves500Code } from "./previewMoves500Code.ts";
import type { Problem } from "./problem.ts";

export type PreviewMoves500 = Problem & {
  code?: PreviewMoves500Code;
};
