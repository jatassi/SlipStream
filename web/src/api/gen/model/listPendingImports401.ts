import type { ListPendingImports401Code } from "./listPendingImports401Code.ts";
import type { Problem } from "./problem.ts";

export type ListPendingImports401 = Problem & {
  code?: ListPendingImports401Code;
};
