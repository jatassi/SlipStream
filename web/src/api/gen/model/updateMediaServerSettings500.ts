import type { Problem } from "./problem.ts";
import type { UpdateMediaServerSettings500Code } from "./updateMediaServerSettings500Code.ts";

export type UpdateMediaServerSettings500 = Problem & {
  code?: UpdateMediaServerSettings500Code;
};
