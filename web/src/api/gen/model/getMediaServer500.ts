import type { GetMediaServer500Code } from "./getMediaServer500Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServer500 = Problem & {
  code?: GetMediaServer500Code;
};
