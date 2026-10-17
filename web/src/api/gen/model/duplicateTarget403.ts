import type { DuplicateTarget403Code } from "./duplicateTarget403Code.ts";
import type { Problem } from "./problem.ts";

export type DuplicateTarget403 = Problem & {
  code?: DuplicateTarget403Code;
};
