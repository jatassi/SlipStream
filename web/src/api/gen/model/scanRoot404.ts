import type { Problem } from "./problem.ts";
import type { ScanRoot404Code } from "./scanRoot404Code.ts";

export type ScanRoot404 = Problem & {
  code?: ScanRoot404Code;
};
