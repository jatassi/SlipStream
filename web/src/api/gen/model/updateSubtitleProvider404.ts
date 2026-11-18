import type { Problem } from "./problem.ts";
import type { UpdateSubtitleProvider404Code } from "./updateSubtitleProvider404Code.ts";

export type UpdateSubtitleProvider404 = Problem & {
  code?: UpdateSubtitleProvider404Code;
};
