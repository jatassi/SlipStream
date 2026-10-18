import type { GetWorkWants401Code } from "./getWorkWants401Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkWants401 = Problem & {
  code?: GetWorkWants401Code;
};
