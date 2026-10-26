import type { GetFirstRun500Code } from "./getFirstRun500Code.ts";
import type { Problem } from "./problem.ts";

export type GetFirstRun500 = Problem & {
  code?: GetFirstRun500Code;
};
