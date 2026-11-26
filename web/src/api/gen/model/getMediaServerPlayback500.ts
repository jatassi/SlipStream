import type { GetMediaServerPlayback500Code } from "./getMediaServerPlayback500Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServerPlayback500 = Problem & {
  code?: GetMediaServerPlayback500Code;
};
