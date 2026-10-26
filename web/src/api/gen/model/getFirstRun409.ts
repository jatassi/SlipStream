import type { GetFirstRun409Code } from "./getFirstRun409Code.ts";
import type { Problem } from "./problem.ts";

export type GetFirstRun409 = Problem & {
  code?: GetFirstRun409Code;
};
