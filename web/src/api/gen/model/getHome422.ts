import type { GetHome422Code } from "./getHome422Code.ts";
import type { Problem } from "./problem.ts";

export type GetHome422 = Problem & {
  code?: GetHome422Code;
};
