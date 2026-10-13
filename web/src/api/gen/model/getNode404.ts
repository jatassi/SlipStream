import type { GetNode404Code } from "./getNode404Code.ts";
import type { Problem } from "./problem.ts";

export type GetNode404 = Problem & {
  code?: GetNode404Code;
};
