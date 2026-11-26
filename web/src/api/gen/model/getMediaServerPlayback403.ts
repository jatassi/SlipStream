import type { GetMediaServerPlayback403Code } from "./getMediaServerPlayback403Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServerPlayback403 = Problem & {
  code?: GetMediaServerPlayback403Code;
};
