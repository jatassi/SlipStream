import type { GetTarget404Code } from "./getTarget404Code.ts";
import type { Problem } from "./problem.ts";

export type GetTarget404 = Problem & {
  code?: GetTarget404Code;
};
