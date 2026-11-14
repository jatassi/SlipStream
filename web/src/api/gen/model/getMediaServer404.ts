import type { GetMediaServer404Code } from "./getMediaServer404Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServer404 = Problem & {
  code?: GetMediaServer404Code;
};
