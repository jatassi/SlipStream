import type { Problem } from "./problem.ts";
import type { ScanRoot401Code } from "./scanRoot401Code.ts";

export type ScanRoot401 = Problem & {
  code?: ScanRoot401Code;
};
