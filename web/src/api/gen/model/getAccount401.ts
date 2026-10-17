import type { GetAccount401Code } from "./getAccount401Code.ts";
import type { Problem } from "./problem.ts";

export type GetAccount401 = Problem & {
  code?: GetAccount401Code;
};
