import type { GetMediaServerSettings500Code } from "./getMediaServerSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServerSettings500 = Problem & {
  code?: GetMediaServerSettings500Code;
};
