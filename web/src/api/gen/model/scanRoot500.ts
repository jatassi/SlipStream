import type { Problem } from "./problem.ts";
import type { ScanRoot500Code } from "./scanRoot500Code.ts";

export type ScanRoot500 = Problem & {
  code?: ScanRoot500Code;
};
