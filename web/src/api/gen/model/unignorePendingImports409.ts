import type { Problem } from "./problem.ts";
import type { UnignorePendingImports409Code } from "./unignorePendingImports409Code.ts";

export type UnignorePendingImports409 = Problem & {
  code?: UnignorePendingImports409Code;
};
