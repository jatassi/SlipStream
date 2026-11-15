import type { GetWorkSearching401Code } from "./getWorkSearching401Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkSearching401 = Problem & {
  code?: GetWorkSearching401Code;
};
