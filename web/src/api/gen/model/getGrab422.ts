import type { GetGrab422Code } from "./getGrab422Code.ts";
import type { Problem } from "./problem.ts";

export type GetGrab422 = Problem & {
  code?: GetGrab422Code;
};
