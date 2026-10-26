import type { FinishFirstRun403Code } from "./finishFirstRun403Code.ts";
import type { Problem } from "./problem.ts";

export type FinishFirstRun403 = Problem & {
  code?: FinishFirstRun403Code;
};
