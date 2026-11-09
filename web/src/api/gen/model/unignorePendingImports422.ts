import type { Problem } from "./problem.ts";
import type { UnignorePendingImports422Code } from "./unignorePendingImports422Code.ts";

export type UnignorePendingImports422 = Problem & {
  code?: UnignorePendingImports422Code;
};
