import type { GetTarget401Code } from "./getTarget401Code.ts";
import type { Problem } from "./problem.ts";

export type GetTarget401 = Problem & {
  code?: GetTarget401Code;
};
