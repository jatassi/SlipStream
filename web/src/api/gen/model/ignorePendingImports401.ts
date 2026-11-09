import type { IgnorePendingImports401Code } from "./ignorePendingImports401Code.ts";
import type { Problem } from "./problem.ts";

export type IgnorePendingImports401 = Problem & {
  code?: IgnorePendingImports401Code;
};
