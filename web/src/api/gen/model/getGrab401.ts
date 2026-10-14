import type { GetGrab401Code } from "./getGrab401Code.ts";
import type { Problem } from "./problem.ts";

export type GetGrab401 = Problem & {
  code?: GetGrab401Code;
};
