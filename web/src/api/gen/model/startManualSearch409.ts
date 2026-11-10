import type { Problem } from "./problem.ts";
import type { StartManualSearch409Code } from "./startManualSearch409Code.ts";

export type StartManualSearch409 = Problem & {
  code?: StartManualSearch409Code;
};
