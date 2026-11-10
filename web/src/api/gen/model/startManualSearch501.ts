import type { Problem } from "./problem.ts";
import type { StartManualSearch501Code } from "./startManualSearch501Code.ts";

export type StartManualSearch501 = Problem & {
  code?: StartManualSearch501Code;
};
