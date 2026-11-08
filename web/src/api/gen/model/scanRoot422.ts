import type { Problem } from "./problem.ts";
import type { ScanRoot422Code } from "./scanRoot422Code.ts";

export type ScanRoot422 = Problem & {
  code?: ScanRoot422Code;
};
