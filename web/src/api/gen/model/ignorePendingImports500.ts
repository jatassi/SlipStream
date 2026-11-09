import type { IgnorePendingImports500Code } from "./ignorePendingImports500Code.ts";
import type { Problem } from "./problem.ts";

export type IgnorePendingImports500 = Problem & {
  code?: IgnorePendingImports500Code;
};
