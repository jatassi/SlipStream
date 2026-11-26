import type { GetMediaServerPlayback422Code } from "./getMediaServerPlayback422Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServerPlayback422 = Problem & {
  code?: GetMediaServerPlayback422Code;
};
