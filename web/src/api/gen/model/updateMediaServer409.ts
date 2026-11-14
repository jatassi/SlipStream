import type { Problem } from "./problem.ts";
import type { UpdateMediaServer409Code } from "./updateMediaServer409Code.ts";

export type UpdateMediaServer409 = Problem & {
  code?: UpdateMediaServer409Code;
};
