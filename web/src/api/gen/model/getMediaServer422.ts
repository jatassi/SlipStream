import type { GetMediaServer422Code } from "./getMediaServer422Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServer422 = Problem & {
  code?: GetMediaServer422Code;
};
