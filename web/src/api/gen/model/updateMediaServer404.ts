import type { Problem } from "./problem.ts";
import type { UpdateMediaServer404Code } from "./updateMediaServer404Code.ts";

export type UpdateMediaServer404 = Problem & {
  code?: UpdateMediaServer404Code;
};
