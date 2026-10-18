import type { DuplicateTarget401Code } from "./duplicateTarget401Code.ts";
import type { Problem } from "./problem.ts";

export type DuplicateTarget401 = Problem & {
  code?: DuplicateTarget401Code;
};
