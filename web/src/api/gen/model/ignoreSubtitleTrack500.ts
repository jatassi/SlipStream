import type { IgnoreSubtitleTrack500Code } from "./ignoreSubtitleTrack500Code.ts";
import type { Problem } from "./problem.ts";

export type IgnoreSubtitleTrack500 = Problem & {
  code?: IgnoreSubtitleTrack500Code;
};
