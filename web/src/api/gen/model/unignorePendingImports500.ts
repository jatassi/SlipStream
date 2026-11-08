import type { Problem } from "./problem.ts";
import type { UnignorePendingImports500Code } from "./unignorePendingImports500Code.ts";

export type UnignorePendingImports500 = Problem & {
  code?: UnignorePendingImports500Code;
};
