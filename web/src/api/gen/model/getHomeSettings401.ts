import type { GetHomeSettings401Code } from "./getHomeSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetHomeSettings401 = Problem & {
  code?: GetHomeSettings401Code;
};
