import type { GetMediaServer403Code } from "./getMediaServer403Code.ts";
import type { Problem } from "./problem.ts";

export type GetMediaServer403 = Problem & {
  code?: GetMediaServer403Code;
};
