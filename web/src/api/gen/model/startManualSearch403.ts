import type { Problem } from "./problem.ts";
import type { StartManualSearch403Code } from "./startManualSearch403Code.ts";

export type StartManualSearch403 = Problem & {
  code?: StartManualSearch403Code;
};
