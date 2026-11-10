import type { Problem } from "./problem.ts";
import type { StartManualSearch500Code } from "./startManualSearch500Code.ts";

export type StartManualSearch500 = Problem & {
  code?: StartManualSearch500Code;
};
