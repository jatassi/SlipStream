import type { GetGrab404Code } from "./getGrab404Code.ts";
import type { Problem } from "./problem.ts";

export type GetGrab404 = Problem & {
  code?: GetGrab404Code;
};
