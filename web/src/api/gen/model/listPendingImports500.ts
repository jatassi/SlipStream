import type { ListPendingImports500Code } from "./listPendingImports500Code.ts";
import type { Problem } from "./problem.ts";

export type ListPendingImports500 = Problem & {
  code?: ListPendingImports500Code;
};
