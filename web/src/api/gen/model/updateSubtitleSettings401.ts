import type { Problem } from "./problem.ts";
import type { UpdateSubtitleSettings401Code } from "./updateSubtitleSettings401Code.ts";

export type UpdateSubtitleSettings401 = Problem & {
  code?: UpdateSubtitleSettings401Code;
};
