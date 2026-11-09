import type { IgnorePendingImports422Code } from "./ignorePendingImports422Code.ts";
import type { Problem } from "./problem.ts";

export type IgnorePendingImports422 = Problem & {
  code?: IgnorePendingImports422Code;
};
