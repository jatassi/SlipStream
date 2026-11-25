import type { Problem } from "./problem.ts";
import type { UpdateMediaServer501Code } from "./updateMediaServer501Code.ts";

export type UpdateMediaServer501 = Problem & {
  code?: UpdateMediaServer501Code;
};
