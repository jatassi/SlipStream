import type { Problem } from "./problem.ts";
import type { UpdateMediaServer401Code } from "./updateMediaServer401Code.ts";

export type UpdateMediaServer401 = Problem & {
  code?: UpdateMediaServer401Code;
};
