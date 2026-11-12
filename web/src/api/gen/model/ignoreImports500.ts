import type { IgnoreImports500Code } from "./ignoreImports500Code.ts";
import type { Problem } from "./problem.ts";

export type IgnoreImports500 = Problem & {
  code?: IgnoreImports500Code;
};
