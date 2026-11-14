import type { Problem } from "./problem.ts";
import type { UpdateMediaServer500Code } from "./updateMediaServer500Code.ts";

export type UpdateMediaServer500 = Problem & {
  code?: UpdateMediaServer500Code;
};
