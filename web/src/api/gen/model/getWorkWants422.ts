import type { GetWorkWants422Code } from "./getWorkWants422Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkWants422 = Problem & {
  code?: GetWorkWants422Code;
};
