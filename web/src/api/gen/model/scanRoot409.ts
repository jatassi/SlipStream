import type { Problem } from "./problem.ts";
import type { ScanRoot409Code } from "./scanRoot409Code.ts";

export type ScanRoot409 = Problem & {
  code?: ScanRoot409Code;
};
