import type { IgnoreImports422Code } from "./ignoreImports422Code.ts";
import type { Problem } from "./problem.ts";

export type IgnoreImports422 = Problem & {
  code?: IgnoreImports422Code;
};
