import type { DuplicateTarget404Code } from "./duplicateTarget404Code.ts";
import type { Problem } from "./problem.ts";

export type DuplicateTarget404 = Problem & {
  code?: DuplicateTarget404Code;
};
