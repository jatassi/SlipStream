import type { GetWorkSearching500Code } from "./getWorkSearching500Code.ts";
import type { Problem } from "./problem.ts";

export type GetWorkSearching500 = Problem & {
  code?: GetWorkSearching500Code;
};
