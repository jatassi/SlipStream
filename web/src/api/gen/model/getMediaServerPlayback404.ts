import type { GetMediaServerPlayback404Code } from "./getMediaServerPlayback404Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServerPlayback404 = Problem & {
  code?: GetMediaServerPlayback404Code;
};
