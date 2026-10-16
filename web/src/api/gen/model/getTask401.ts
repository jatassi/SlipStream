import type { GetTask401Code } from "./getTask401Code.ts";
import type { Problem } from "./problem.ts";

export type GetTask401 = Problem & {
  code?: GetTask401Code;
};
