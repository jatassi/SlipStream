import type { IgnorePendingImports409Code } from "./ignorePendingImports409Code.ts";
import type { Problem } from "./problem.ts";

export type IgnorePendingImports409 = Problem & {
  code?: IgnorePendingImports409Code;
};
