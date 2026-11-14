import type { Problem } from "./problem.ts";
import type { UnignoreImports409Code } from "./unignoreImports409Code.ts";

export type UnignoreImports409 = Problem & {
  code?: UnignoreImports409Code;
};
