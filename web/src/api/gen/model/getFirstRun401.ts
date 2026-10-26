import type { GetFirstRun401Code } from "./getFirstRun401Code.ts";
import type { Problem } from "./problem.ts";

export type GetFirstRun401 = Problem & {
  code?: GetFirstRun401Code;
};
