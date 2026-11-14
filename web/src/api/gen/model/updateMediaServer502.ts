import type { Problem } from "./problem.ts";
import type { UpdateMediaServer502Code } from "./updateMediaServer502Code.ts";

export type UpdateMediaServer502 = Problem & {
  code?: UpdateMediaServer502Code;
};
