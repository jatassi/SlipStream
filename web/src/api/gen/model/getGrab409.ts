import type { GetGrab409Code } from "./getGrab409Code.ts";
import type { Problem } from "./problem.ts";

export type GetGrab409 = Problem & {
  code?: GetGrab409Code;
};
