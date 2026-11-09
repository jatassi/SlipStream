import type { Problem } from "./problem.ts";
import type { UnignorePendingImports403Code } from "./unignorePendingImports403Code.ts";

export type UnignorePendingImports403 = Problem & {
  code?: UnignorePendingImports403Code;
};
