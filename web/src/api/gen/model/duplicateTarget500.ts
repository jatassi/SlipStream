import type { DuplicateTarget500Code } from "./duplicateTarget500Code.ts";
import type { Problem } from "./problem.ts";

export type DuplicateTarget500 = Problem & {
  code?: DuplicateTarget500Code;
};
