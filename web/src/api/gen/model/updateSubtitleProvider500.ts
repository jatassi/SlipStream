import type { Problem } from "./problem.ts";
import type { UpdateSubtitleProvider500Code } from "./updateSubtitleProvider500Code.ts";

export type UpdateSubtitleProvider500 = Problem & {
  code?: UpdateSubtitleProvider500Code;
};
