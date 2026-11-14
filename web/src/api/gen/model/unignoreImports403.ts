import type { Problem } from "./problem.ts";
import type { UnignoreImports403Code } from "./unignoreImports403Code.ts";

export type UnignoreImports403 = Problem & {
  code?: UnignoreImports403Code;
};
