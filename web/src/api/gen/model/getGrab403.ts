import type { GetGrab403Code } from "./getGrab403Code.ts";
import type { Problem } from "./problem.ts";

export type GetGrab403 = Problem & {
  code?: GetGrab403Code;
};
