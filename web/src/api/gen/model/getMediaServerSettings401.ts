import type { GetMediaServerSettings401Code } from "./getMediaServerSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServerSettings401 = Problem & {
  code?: GetMediaServerSettings401Code;
};
