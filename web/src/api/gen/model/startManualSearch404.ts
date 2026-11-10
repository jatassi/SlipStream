import type { Problem } from "./problem.ts";
import type { StartManualSearch404Code } from "./startManualSearch404Code.ts";

export type StartManualSearch404 = Problem & {
  code?: StartManualSearch404Code;
};
