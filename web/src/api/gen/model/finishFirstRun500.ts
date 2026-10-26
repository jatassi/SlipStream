import type { FinishFirstRun500Code } from "./finishFirstRun500Code.ts";
import type { Problem } from "./problem.ts";

export type FinishFirstRun500 = Problem & {
  code?: FinishFirstRun500Code;
};
