import type { Problem } from "./problem.ts";
import type { StartManualSearch422Code } from "./startManualSearch422Code.ts";

export type StartManualSearch422 = Problem & {
  code?: StartManualSearch422Code;
};
