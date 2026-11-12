import type { IgnoreImports409Code } from "./ignoreImports409Code.ts";
import type { Problem } from "./problem.ts";

export type IgnoreImports409 = Problem & {
  code?: IgnoreImports409Code;
};
