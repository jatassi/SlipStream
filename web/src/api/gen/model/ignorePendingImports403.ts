import type { IgnorePendingImports403Code } from "./ignorePendingImports403Code.ts";
import type { Problem } from "./problem.ts";

export type IgnorePendingImports403 = Problem & {
  code?: IgnorePendingImports403Code;
};
