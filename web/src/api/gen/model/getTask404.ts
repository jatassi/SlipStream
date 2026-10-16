import type { GetTask404Code } from "./getTask404Code.ts";
import type { Problem } from "./problem.ts";

export type GetTask404 = Problem & {
  code?: GetTask404Code;
};
