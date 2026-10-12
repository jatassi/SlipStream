import type { GetSession401Code } from "./getSession401Code.ts";
import type { Problem } from "./problem.ts";

export type GetSession401 = Problem & {
  code?: GetSession401Code;
};
