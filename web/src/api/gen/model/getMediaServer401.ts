import type { GetMediaServer401Code } from "./getMediaServer401Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServer401 = Problem & {
  code?: GetMediaServer401Code;
};
