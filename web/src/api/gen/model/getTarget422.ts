import type { GetTarget422Code } from "./getTarget422Code.ts";
import type { Problem } from "./problem.ts";

export type GetTarget422 = Problem & {
  code?: GetTarget422Code;
};
