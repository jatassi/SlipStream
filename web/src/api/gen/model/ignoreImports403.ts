import type { IgnoreImports403Code } from "./ignoreImports403Code.ts";
import type { Problem } from "./problem.ts";

export type IgnoreImports403 = Problem & {
  code?: IgnoreImports403Code;
};
