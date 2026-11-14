import type { GetMediaServer409Code } from "./getMediaServer409Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServer409 = Problem & {
  code?: GetMediaServer409Code;
};
