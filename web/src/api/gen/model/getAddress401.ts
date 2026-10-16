import type { GetAddress401Code } from "./getAddress401Code.ts";
import type { Problem } from "./problem.ts";

export type GetAddress401 = Problem & {
  code?: GetAddress401Code;
};
