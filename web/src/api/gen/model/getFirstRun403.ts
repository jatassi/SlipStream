import type { GetFirstRun403Code } from "./getFirstRun403Code.ts";
import type { Problem } from "./problem.ts";

export type GetFirstRun403 = Problem & {
  code?: GetFirstRun403Code;
};
