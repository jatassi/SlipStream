import type { Problem } from "./problem.ts";
import type { UpdateMediaServerSettings401Code } from "./updateMediaServerSettings401Code.ts";

export type UpdateMediaServerSettings401 = Problem & {
  code?: UpdateMediaServerSettings401Code;
};
