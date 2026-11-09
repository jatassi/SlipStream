import type { Problem } from "./problem.ts";
import type { UnignorePendingImports401Code } from "./unignorePendingImports401Code.ts";

export type UnignorePendingImports401 = Problem & {
  code?: UnignorePendingImports401Code;
};
