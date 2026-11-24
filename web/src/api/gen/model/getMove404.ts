import type { GetMove404Code } from "./getMove404Code.ts";
import type { Problem } from "./problem.ts";

export type GetMove404 = Problem & {
  code?: GetMove404Code;
};
