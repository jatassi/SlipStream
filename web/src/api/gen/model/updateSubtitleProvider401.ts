import type { Problem } from "./problem.ts";
import type { UpdateSubtitleProvider401Code } from "./updateSubtitleProvider401Code.ts";

export type UpdateSubtitleProvider401 = Problem & {
  code?: UpdateSubtitleProvider401Code;
};
