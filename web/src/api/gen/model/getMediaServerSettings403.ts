import type { GetMediaServerSettings403Code } from "./getMediaServerSettings403Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServerSettings403 = Problem & {
  code?: GetMediaServerSettings403Code;
};
