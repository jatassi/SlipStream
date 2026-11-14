import type { Problem } from "./problem.ts";
import type { UpdateMediaServer422Code } from "./updateMediaServer422Code.ts";

export type UpdateMediaServer422 = Problem & {
  code?: UpdateMediaServer422Code;
};
