import type { Problem } from "./problem.ts";
import type { UnignoreImports401Code } from "./unignoreImports401Code.ts";

export type UnignoreImports401 = Problem & {
  code?: UnignoreImports401Code;
};
