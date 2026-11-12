import type { Problem } from "./problem.ts";
import type { UnignoreImports500Code } from "./unignoreImports500Code.ts";

export type UnignoreImports500 = Problem & {
  code?: UnignoreImports500Code;
};
