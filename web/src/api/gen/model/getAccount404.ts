import type { GetAccount404Code } from "./getAccount404Code.ts";
import type { Problem } from "./problem.ts";

export type GetAccount404 = Problem & {
  code?: GetAccount404Code;
};
