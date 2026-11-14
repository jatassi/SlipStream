import type { Problem } from "./problem.ts";
import type { UpdateMediaServer403Code } from "./updateMediaServer403Code.ts";

export type UpdateMediaServer403 = Problem & {
  code?: UpdateMediaServer403Code;
};
