import type { Problem } from "./problem.ts";
import type { StartManualSearch401Code } from "./startManualSearch401Code.ts";

export type StartManualSearch401 = Problem & {
  code?: StartManualSearch401Code;
};
