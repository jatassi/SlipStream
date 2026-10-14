import type { GetGrab500Code } from "./getGrab500Code.ts";
import type { Problem } from "./problem.ts";

export type GetGrab500 = Problem & {
  code?: GetGrab500Code;
};
