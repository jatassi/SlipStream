import type { GetHome401Code } from "./getHome401Code.ts";
import type { Problem } from "./problem.ts";

export type GetHome401 = Problem & {
  code?: GetHome401Code;
};
