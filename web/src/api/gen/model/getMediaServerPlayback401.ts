import type { GetMediaServerPlayback401Code } from "./getMediaServerPlayback401Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServerPlayback401 = Problem & {
  code?: GetMediaServerPlayback401Code;
};
