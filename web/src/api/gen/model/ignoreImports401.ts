import type { IgnoreImports401Code } from "./ignoreImports401Code.ts";
import type { Problem } from "./problem.ts";

export type IgnoreImports401 = Problem & {
  code?: IgnoreImports401Code;
};
