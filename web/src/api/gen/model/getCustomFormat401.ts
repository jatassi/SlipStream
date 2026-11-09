import type { GetCustomFormat401Code } from "./getCustomFormat401Code.ts";
import type { Problem } from "./problem.ts";

export type GetCustomFormat401 = Problem & {
  code?: GetCustomFormat401Code;
};
