import type { Problem } from "./problem.ts";
import type { ScanRoot403Code } from "./scanRoot403Code.ts";

export type ScanRoot403 = Problem & {
  code?: ScanRoot403Code;
};
