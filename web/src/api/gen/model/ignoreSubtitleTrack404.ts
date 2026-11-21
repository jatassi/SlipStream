import type { IgnoreSubtitleTrack404Code } from "./ignoreSubtitleTrack404Code.ts";
import type { Problem } from "./problem.ts";

export type IgnoreSubtitleTrack404 = Problem & {
  code?: IgnoreSubtitleTrack404Code;
};
