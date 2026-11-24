import type { GetMove401Code } from "./getMove401Code.ts";
import type { Problem } from "./problem.ts";

export type GetMove401 = Problem & {
  code?: GetMove401Code;
};
