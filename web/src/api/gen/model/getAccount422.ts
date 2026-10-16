import type { GetAccount422Code } from "./getAccount422Code.ts";
import type { Problem } from "./problem.ts";

export type GetAccount422 = Problem & {
  code?: GetAccount422Code;
};
