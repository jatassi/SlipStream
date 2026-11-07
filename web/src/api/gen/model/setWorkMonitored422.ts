import type { Problem } from "./problem.ts";
import type { SetWorkMonitored422Code } from "./setWorkMonitored422Code.ts";

export type SetWorkMonitored422 = Problem & {
  code?: SetWorkMonitored422Code;
};
