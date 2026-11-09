import type { GetCustomFormat422Code } from "./getCustomFormat422Code.ts";
import type { Problem } from "./problem.ts";

export type GetCustomFormat422 = Problem & {
  code?: GetCustomFormat422Code;
};
