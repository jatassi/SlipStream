import type { GetAccount500Code } from "./getAccount500Code.ts";
import type { Problem } from "./problem.ts";

export type GetAccount500 = Problem & {
  code?: GetAccount500Code;
};
