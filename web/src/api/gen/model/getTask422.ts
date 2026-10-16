import type { GetTask422Code } from "./getTask422Code.ts";
import type { Problem } from "./problem.ts";

export type GetTask422 = Problem & {
  code?: GetTask422Code;
};
